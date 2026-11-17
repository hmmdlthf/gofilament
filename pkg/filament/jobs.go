package filament

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/cache"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
	"github.com/golaravel/golaravel/pkg/storage"
)

// Background jobs: long exports and imports run after the request returns.
// The user sees their progress, gets a notification when they finish, and
// downloads the file from the panel. Job state lives in golaravel's cache
// (use a shared store such as Redis when the panel runs on several
// instances); files go to a golaravel storage disk. A job runs in the
// process that started it, so a restart stops the jobs running in it.
//
//	panel.BackgroundJobs(filament.JobsConfig{Disk: "local"})

// JobsConfig configures background jobs.
type JobsConfig struct {
	// Disk is the storage disk for result files ("" = the default disk).
	Disk string
	// Dir is the folder of result files (default "gofilament-jobs").
	Dir string
	// Concurrency is how many jobs run at once (default 2).
	Concurrency int
	// TTL is how long jobs are listed (default 7 days). Delete old files
	// from Dir yourself, e.g. with a scheduled task.
	TTL time.Duration
}

// Job is a background job's state.
type Job struct {
	ID          string    `json:"id"`
	Panel       string    `json:"-"`
	Owner       string    `json:"-"`
	Kind        string    `json:"kind"` // "export", "import"
	Title       string    `json:"title"`
	Status      string    `json:"status"` // queued, running, done, failed
	Done        int       `json:"done"`
	Total       int       `json:"total"`
	Progress    int       `json:"progress"` // 0-100
	Message     string    `json:"message,omitempty"`
	File        string    `json:"-"`
	FileName    string    `json:"file_name,omitempty"`
	ContentType string    `json:"-"`
	Result      any       `json:"result,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
}

type jobRunner struct {
	cfg JobsConfig
	sem chan struct{}
	mu  sync.Mutex // guards the per-user job lists
}

// BackgroundJobs turns on background exports and imports.
func (p *Panel) BackgroundJobs(cfg JobsConfig) *Panel {
	if cfg.Dir == "" {
		cfg.Dir = "gofilament-jobs"
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 2
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 7 * 24 * time.Hour
	}
	p.jobs = &jobRunner{cfg: cfg, sem: make(chan struct{}, cfg.Concurrency)}
	return p
}

func (p *Panel) jobKey(id string) string { return "gofilament:" + p.id + ":job:" + id }
func (p *Panel) jobListKey(owner string) string {
	return "gofilament:" + p.id + ":jobs:" + importHashKey(owner)
}

func (p *Panel) saveJob(j *Job) {
	raw, err := json.Marshal(struct {
		*Job
		Panel       string `json:"panel"`
		Owner       string `json:"owner"`
		File        string `json:"file"`
		ContentType string `json:"content_type"`
	}{j, j.Panel, j.Owner, j.File, j.ContentType})
	if err == nil {
		_ = cache.Put(p.jobKey(j.ID), string(raw), p.jobs.cfg.TTL)
	}
}

// GetJob loads a job ("" owner: any owner).
func (p *Panel) GetJob(id, owner string) (*Job, bool) {
	raw := cache.GetString(p.jobKey(id))
	if raw == "" {
		return nil, false
	}
	var full struct {
		Job
		Panel       string `json:"panel"`
		Owner       string `json:"owner"`
		File        string `json:"file"`
		ContentType string `json:"content_type"`
	}
	if json.Unmarshal([]byte(raw), &full) != nil {
		return nil, false
	}
	j := full.Job
	j.Panel, j.Owner, j.File, j.ContentType = full.Panel, full.Owner, full.File, full.ContentType
	if j.Panel != p.id || (owner != "" && j.Owner != owner) {
		return nil, false
	}
	return &j, true
}

// UserJobs lists a user's recent jobs, newest first.
func (p *Panel) UserJobs(owner string) []*Job {
	var ids []string
	_ = json.Unmarshal([]byte(cache.GetString(p.jobListKey(owner))), &ids)
	var out []*Job
	for _, id := range ids {
		if j, ok := p.GetJob(id, owner); ok {
			out = append(out, j)
		}
	}
	return out
}

func (p *Panel) addUserJob(owner, id string) {
	p.jobs.mu.Lock()
	defer p.jobs.mu.Unlock()
	var ids []string
	_ = json.Unmarshal([]byte(cache.GetString(p.jobListKey(owner))), &ids)
	ids = append([]string{id}, ids...)
	if len(ids) > 50 {
		ids = ids[:50]
	}
	raw, _ := json.Marshal(ids)
	_ = cache.Put(p.jobListKey(owner), string(raw), p.jobs.cfg.TTL)
}

// detach copies what a job needs from the request: user, tenant, locale,
// URL and headers. The job's response goes to w.
func (p *Panel) detach(ctx *Context, w http.ResponseWriter) *Context {
	gc := gcontext.NewContext()
	req := ctx.Request.Clone(context.Background())
	req.Body = http.NoBody
	gc.Reset(w, req)
	if u := ctx.User(); u != nil {
		gc.Set(auth.ContextUserKey, u)
	}
	for _, k := range []string{"locale"} {
		if v, ok := ctx.Get(k); ok {
			gc.Set(k, v)
		}
	}
	return &Context{Context: gc, panel: p, tenant: ctx.tenant}
}

// fileWriter is the http.ResponseWriter of a job: the body goes to a file.
type fileWriter struct {
	f      *os.File
	header http.Header
}

func (w *fileWriter) Header() http.Header         { return w.header }
func (w *fileWriter) Write(b []byte) (int, error) { return w.f.Write(b) }
func (w *fileWriter) WriteHeader(int)             {}

const ctxProgress = "filament_job_progress"

// reportProgress lets long operations report how far they are.
func reportProgress(ctx *Context, done, total int) {
	if v, ok := ctx.Get(ctxProgress); ok {
		if fn, ok := v.(func(done, total int)); ok {
			fn(done, total)
		}
	}
}

func newJobID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "job_" + hex.EncodeToString(b)
}

var dispositionName = regexp.MustCompile(`filename="([^"]+)"`)

// startJob runs fn in the background. fn writes its file (if any) to the
// detached context's response; the result is kept on the job.
func (p *Panel) startJob(ctx *Context, kind, title string, fn func(jctx *Context) (any, error)) (*Job, error) {
	if p.jobs == nil {
		return nil, fmt.Errorf("background jobs are not enabled")
	}
	owner := viewOwner(ctx)
	j := &Job{ID: newJobID(), Panel: p.id, Owner: owner, Kind: kind, Title: title, Status: "queued", CreatedAt: time.Now().UTC()}
	tmp, err := os.CreateTemp("", "gofilament-job-*")
	if err != nil {
		return nil, err
	}
	w := &fileWriter{f: tmp, header: http.Header{}}
	jctx := p.detach(ctx, w)
	var mu sync.Mutex
	last := time.Time{}
	jctx.Set(ctxProgress, func(done, total int) {
		mu.Lock()
		defer mu.Unlock()
		j.Done, j.Total = done, total
		if total > 0 {
			j.Progress = min(99, done*100/total)
		}
		if time.Since(last) > 500*time.Millisecond {
			last = time.Now()
			p.saveJob(j)
		}
	})
	p.saveJob(j)
	p.addUserJob(owner, j.ID)
	snapshot := *j

	go func() {
		defer os.Remove(tmp.Name())
		defer tmp.Close()
		p.jobs.sem <- struct{}{}
		defer func() { <-p.jobs.sem }()
		mu.Lock()
		j.Status = "running"
		p.saveJob(j)
		mu.Unlock()

		result, err := func() (res any, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("job crashed: %v", r)
				}
			}()
			return fn(jctx)
		}()
		mu.Lock()
		defer mu.Unlock()
		j.Result = result
		if err == nil {
			if info, _ := tmp.Stat(); info != nil && info.Size() > 0 {
				name := "result"
				if m := dispositionName.FindStringSubmatch(w.header.Get("Content-Disposition")); m != nil {
					name = m[1]
				}
				j.FileName = path.Base(name)
				j.ContentType = w.header.Get("Content-Type")
				j.File = path.Join(p.jobs.cfg.Dir, j.ID+"-"+j.FileName)
				if _, err = tmp.Seek(0, io.SeekStart); err == nil {
					err = storage.Disk(p.jobs.cfg.Disk).PutReader(j.File, tmp)
				}
			}
		}
		j.FinishedAt = time.Now().UTC()
		n := notifications.Make(title)
		if err != nil {
			j.Status, j.Message = "failed", err.Error()
			log.Printf("filament: job %s failed: %v", j.ID, err)
			n.Danger().Body("It could not be finished: " + err.Error())
		} else {
			j.Status, j.Progress = "done", 100
			if j.Total > 0 {
				j.Done = j.Total
			}
			n.Success().Body("It is ready.")
			if j.File != "" {
				n.Action("Download", p.path+"/jobs/"+j.ID+"/download")
			} else {
				n.Action("View", p.path+"/jobs/"+j.ID)
			}
		}
		p.saveJob(j)
		if owner != "" {
			_ = notifications.Send(owner, n)
		}
	}()
	return &snapshot, nil
}

// serveJobFile streams a finished job's file to its owner.
func (p *Panel) serveJobFile(c *gcontext.Context, j *Job) error {
	if j.Status != "done" || j.File == "" {
		return c.String(http.StatusNotFound, "Not found")
	}
	rc, err := storage.Disk(p.jobs.cfg.Disk).GetStream(j.File)
	if err != nil {
		return c.String(http.StatusGone, "The file is no longer available")
	}
	defer rc.Close()
	ct := j.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	c.Response.Header().Set("Content-Type", ct)
	c.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(j.FileName, `"`, "")))
	c.Response.WriteHeader(http.StatusOK)
	_, err = io.Copy(&c.Response, rc)
	return err
}

func (p *Panel) mountJobRoutes(r *router.Router) {
	if p.jobs == nil {
		return
	}
	api := p.path + "/api"
	load := func(c *gcontext.Context) (*Job, bool, error) {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return nil, false, err
		}
		j, ok := p.GetJob(c.Param("job"), viewOwner(ctx))
		return j, ok, nil
	}
	r.GET(api+"/jobs", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		jobs := p.UserJobs(viewOwner(ctx))
		if jobs == nil {
			jobs = []*Job{}
		}
		return c.JSON(http.StatusOK, map[string]any{"jobs": jobs})
	})
	r.GET(api+"/jobs/:job", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		j, ok, err := load(c)
		if err != nil {
			return err
		}
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Job not found"})
		}
		return c.JSON(http.StatusOK, j)
	})
	r.GET(api+"/jobs/:job/download", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		j, ok, err := load(c)
		if err != nil {
			return err
		}
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Job not found"})
		}
		return p.serveJobFile(c, j)
	})
	r.GET(p.path+"/jobs/:job/download", func(c *gcontext.Context) error {
		j, ok, err := load(c)
		if err != nil {
			return err
		}
		if !ok {
			return c.String(http.StatusNotFound, "Not found")
		}
		return p.serveJobFile(c, j)
	})
	r.GET(p.path+"/jobs/:job", func(c *gcontext.Context) error {
		j, ok, err := load(c)
		if err != nil {
			return err
		}
		if !ok {
			return c.String(http.StatusNotFound, "Not found")
		}
		return p.renderJobPage(NewContext(c, p), j)
	})
}

func (p *Panel) renderJobPage(ctx *Context, j *Job) error {
	esc := html.EscapeString
	var b strings.Builder
	refresh := ""
	fmt.Fprintf(&b, `<div class="filament-card" style="max-width: 40rem; margin: 2rem auto;" data-job-status="%s"><h1 class="filament-title">%s</h1>`, esc(j.Status), esc(j.Title))
	switch j.Status {
	case "queued", "running":
		refresh = `<meta http-equiv="refresh" content="2">`
		progress := fmt.Sprintf("%d%%", j.Progress)
		if j.Total > 0 {
			progress = fmt.Sprintf("%d of %d (%d%%)", j.Done, j.Total, j.Progress)
		}
		fmt.Fprintf(&b, `<p style="margin: 1rem 0;">Working on it: %s. You can leave this page; you will get a notification when it is ready.</p><div style="height: 0.5rem; background: var(--gray-200); border-radius: 9999px;"><div style="height: 100%%; width: %d%%; background: var(--primary-600); border-radius: 9999px;"></div></div>`, esc(progress), j.Progress)
	case "done":
		b.WriteString(alertHTML("ok", "Finished."))
		if j.File != "" {
			fmt.Fprintf(&b, `<a href="%s/jobs/%s/download" class="btn btn-primary">Download %s</a>`, p.path, esc(j.ID), esc(j.FileName))
		}
		if res, ok := j.Result.(map[string]any); ok {
			fmt.Fprintf(&b, `<p style="margin-top: 1rem;">%v rows imported, %v skipped.</p>`, res["imported"], res["failed"])
		}
	default:
		b.WriteString(alertHTML("err", "It could not be finished: "+j.Message))
	}
	fmt.Fprintf(&b, `<p style="margin-top: 1rem;"><a href="%s">Back to the dashboard</a></p></div>`, p.path)
	page := fmt.Sprintf(`<!DOCTYPE html><html lang="%s"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">%s<title>%s</title><link rel="stylesheet" href="%s/assets/filament.css"></head><body style="background: var(--gray-50);">%s</body></html>`,
		ctx.Locale(), refresh, esc(j.Title), p.path, b.String())
	return ctx.HTML(http.StatusOK, page)
}
