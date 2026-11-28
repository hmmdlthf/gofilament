package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/storage"
	gltest "github.com/golaravel/golaravel/pkg/testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

type mediaItem struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Photo    string            `json:"photo"`
	Cover    string            `json:"cover"`
	Docs     []string          `json:"docs"`
	DocNames map[string]string `json:"doc_names"`
}

type mediaStore struct {
	mu       sync.Mutex
	items    map[string]*mediaItem
	failNext bool
}

func strList(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := []string{}
		for _, e := range x {
			out = append(out, fmt.Sprint(e))
		}
		return out
	}
	return nil
}

func (s *mediaStore) apply(it *mediaItem, data map[string]any) {
	if v, ok := data["title"]; ok {
		it.Title = fmt.Sprint(v)
	}
	if v, ok := data["photo"]; ok {
		it.Photo = fmt.Sprint(v)
	}
	if v, ok := data["cover"]; ok {
		it.Cover = fmt.Sprint(v)
	}
	if v, ok := data["docs"]; ok {
		it.Docs = strList(v)
	}
	if v, ok := data["doc_names"].(map[string]string); ok {
		it.DocNames = v
	}
}

func newMediaApp(t *testing.T) (*golaravel.Engine, *mediaStore, *gltest.FakeDisk, *gltest.FakeDisk) {
	t.Helper()
	media := gltest.FakeStorage(t, "media")
	media.SetVisibility(storage.VisibilityPublic)
	vault := gltest.FakeStorage(t, "vault") // private
	store := &mediaStore{items: map[string]*mediaItem{}}

	res := filament.NewResource[*mediaItem]("media")
	res.RecordID(func(m *mediaItem) string { return m.ID })
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("title"),
			forms.ImageUpload("photo").Disk("media").Directory("photos"),
			forms.ImageUpload("cover").Disk("media").Directory("covers").
				ImageResizeTargetWidth(20).ImageCropAspectRatio("1:1"),
			forms.FileUpload("docs").Disk("vault").Directory("docs").Multiple().
				AcceptedFileTypes("application/pdf").StoreFileNamesIn("doc_names").AppendFiles(),
		)
	})
	res.Table(func(tb *tables.Table[*mediaItem]) *tables.Table[*mediaItem] {
		return tb.Columns(tables.TextColumn[*mediaItem]("title"), tables.ImageColumn[*mediaItem]("photo").Disk("media"))
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*mediaItem, int, error) {
		store.mu.Lock()
		defer store.mu.Unlock()
		var out []*mediaItem
		for _, it := range store.items {
			out = append(out, it)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
		return out, len(out), nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*mediaItem, error) {
		store.mu.Lock()
		defer store.mu.Unlock()
		if it, ok := store.items[id]; ok {
			cp := *it
			return &cp, nil
		}
		return nil, errors.New("not found")
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.failNext {
			store.failNext = false
			return errors.New("database is down")
		}
		it := &mediaItem{ID: fmt.Sprint(len(store.items) + 1)}
		store.apply(it, data)
		store.items[it.ID] = it
		filament.SetCreatedID(ctx, it.ID)
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.failNext {
			store.failNext = false
			return errors.New("database is down")
		}
		store.apply(store.items[id], data)
		return nil
	})

	p := filament.NewPanel("media").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "u1"}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, store, media, vault
}

func mediaUpload(t *testing.T, app http.Handler, field string, f gltest.FakeFile) string {
	t.Helper()
	res := gltest.New(t, app).Request("POST", "/admin/api/resources/media/uploads?field="+field).
		WithToken("good").WithUpload("file", f).Send()
	res.AssertStatus(http.StatusCreated)
	var out map[string]any
	_ = json.Unmarshal(res.Body(), &out)
	return fmt.Sprint(out["value"])
}

func TestPickedFilesAreStoredWhenTheRecordIsSaved(t *testing.T) {
	app, store, media, vault := newMediaApp(t)
	photo := mediaUpload(t, app, "photo", gltest.FakeImage("me.png", 8, 8))
	doc1 := mediaUpload(t, app, "docs", gltest.FakeUpload("Contract 2026.pdf", 2))
	media.AssertCount("tmp-uploads/media/photo", 1).AssertCount("photos", 0)

	code, out := relDo(t, app, "POST", "/admin/api/resources/media", "good",
		map[string]any{"title": "A", "photo": photo, "docs": []any{doc1}})
	if code >= 300 {
		t.Fatalf("create: %d %v", code, out)
	}
	it := store.items["1"]
	if !strings.HasPrefix(it.Photo, "photos/") || !strings.HasSuffix(it.Photo, ".png") {
		t.Fatalf("photo moved into its directory: %q", it.Photo)
	}
	media.AssertExists(it.Photo).AssertDirectoryEmpty("tmp-uploads")
	if len(it.Docs) != 1 || !strings.HasPrefix(it.Docs[0], "docs/") || it.DocNames[it.Docs[0]] != "Contract 2026.pdf" {
		t.Fatalf("docs and their original names: %v %v", it.Docs, it.DocNames)
	}
	vault.AssertExists(it.Docs[0])

	// a token is used up once saved
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/media", "good", map[string]any{"photo": photo}); code != 422 {
		t.Fatalf("reusing a saved upload: %d", code)
	}
	// values that were never uploaded to the field are refused
	for _, forged := range []string{"photos/someone-elses.png", "../../secret.env", "tmp:" + strings.Repeat("a", 32) + ".png/x.png"} {
		if code, _ := relDo(t, app, "POST", "/admin/api/resources/media", "good", map[string]any{"photo": forged}); code != 422 {
			t.Fatalf("%q: want 422, got %d", forged, code)
		}
	}
	// a token of another field can't be used
	other := mediaUpload(t, app, "cover", gltest.FakeImage("c.png", 30, 30))
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/media", "good", map[string]any{"photo": other}); code != 422 {
		t.Fatalf("cross-field token: %d", code)
	}
}

func TestReplacedFilesAreDeletedAfterSave(t *testing.T) {
	app, store, media, vault := newMediaApp(t)
	photo := mediaUpload(t, app, "photo", gltest.FakeImage("a.png", 4, 4))
	doc1 := mediaUpload(t, app, "docs", gltest.FakeUpload("one.pdf", 1))
	relDo(t, app, "POST", "/admin/api/resources/media", "good", map[string]any{"photo": photo, "docs": []any{doc1}})
	old := *store.items["1"]

	// a failed save puts the new file back and keeps the old one
	newPhoto := mediaUpload(t, app, "photo", gltest.FakeImage("b.png", 4, 4))
	store.failNext = true
	if code, _ := relDo(t, app, "PUT", "/admin/api/resources/media/1", "good", map[string]any{"photo": newPhoto}); code < 400 {
		t.Fatalf("expected the failed save to be reported: %d", code)
	}
	media.AssertExists(old.Photo).AssertCount("tmp-uploads", 1).AssertCount("photos", 1)

	// the same upload can be sent again; the old photo goes after the save
	doc2 := mediaUpload(t, app, "docs", gltest.FakeUpload("two.pdf", 1))
	code, out := relDo(t, app, "PUT", "/admin/api/resources/media/1", "good",
		map[string]any{"photo": newPhoto, "docs": []any{old.Docs[0], doc2}})
	if code >= 300 {
		t.Fatalf("update: %d %v", code, out)
	}
	it := store.items["1"]
	if it.Photo == old.Photo {
		t.Fatal("photo not replaced")
	}
	media.AssertMissing(old.Photo).AssertExists(it.Photo).AssertCount("photos", 1)
	if len(it.Docs) != 2 || it.Docs[0] != old.Docs[0] || it.DocNames[it.Docs[1]] != "two.pdf" || it.DocNames[it.Docs[0]] != "one.pdf" {
		t.Fatalf("appended docs keep their names: %v %v", it.Docs, it.DocNames)
	}

	// removing a file from the field deletes it
	docs := append([]string(nil), it.Docs...)
	relDo(t, app, "PUT", "/admin/api/resources/media/1", "good", map[string]any{"docs": []any{docs[1]}})
	vault.AssertMissing(docs[0]).AssertExists(docs[1])
}

func TestRecordAndRowsCarryFileLinks(t *testing.T) {
	app, store, _, _ := newMediaApp(t)
	photo := mediaUpload(t, app, "photo", gltest.FakeImage("a.png", 4, 4))
	doc := mediaUpload(t, app, "docs", gltest.FakeUpload("nda.pdf", 1))
	relDo(t, app, "POST", "/admin/api/resources/media", "good", map[string]any{"photo": photo, "docs": []any{doc}})
	it := store.items["1"]

	code, out := relDo(t, app, "GET", "/admin/api/resources/media/1", "good", nil)
	if code != 200 {
		t.Fatalf("find: %d", code)
	}
	files, _ := out["files"].(map[string]any)
	ph := files["photo"].([]any)[0].(map[string]any)
	if ph["value"] != it.Photo || ph["url"] != "/storage/media/"+it.Photo {
		t.Fatalf("public photo link: %v", ph)
	}
	dc := files["docs"].([]any)[0].(map[string]any)
	if dc["name"] != "nda.pdf" || !strings.Contains(fmt.Sprint(dc["url"]), "signature=") {
		t.Fatalf("private doc gets a signed link and its name: %v", dc)
	}

	_, form := relDo(t, app, "GET", "/admin/api/resources/media/form?id=1", "good", nil)
	if ff, _ := form["files"].(map[string]any); ff["photo"] == nil {
		t.Fatalf("the edit form carries file links too: %v", form["files"])
	}

	_, list := relDo(t, app, "GET", "/admin/api/resources/media", "good", nil)
	meta := list["row_meta"].(map[string]any)["1"].(map[string]any)
	if meta["images"].(map[string]any)["photo"] != "/storage/media/"+it.Photo {
		t.Fatalf("row image links: %v", meta["images"])
	}
}

func TestImagesAreCroppedAndResizedOnUpload(t *testing.T) {
	app, _, media, _ := newMediaApp(t)
	mediaUpload(t, app, "cover", gltest.FakeImage("wide.png", 80, 40))
	files := media.Files("tmp-uploads/media/cover")
	if len(files) != 1 {
		t.Fatalf("staged: %v", files)
	}
	b, _ := media.Get(files[0])
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Size(); got.X != 20 || got.Y != 20 {
		t.Fatalf("cropped to 1:1 and shrunk to 20px: %v", got)
	}
}

func TestOldTemporaryUploadsArePruned(t *testing.T) {
	app, _, media, _ := newMediaApp(t)
	mediaUpload(t, app, "photo", gltest.FakeImage("a.png", 4, 4))
	if n := filament.PruneTemporaryUploads("media", time.Hour); n != 0 {
		t.Fatalf("fresh uploads are kept: %d", n)
	}
	if n := filament.PruneTemporaryUploads("media", -time.Second); n != 1 {
		t.Fatalf("old uploads are removed: %d", n)
	}
	media.AssertDirectoryEmpty("tmp-uploads")
}

func TestFileFieldOptionsReachTheSchema(t *testing.T) {
	gltest.FakeStorage(t, "public")
	f := forms.FileUpload("docs").Multiple().MinFiles(2).MinSize(1).Downloadable().Openable().
		Reorderable().AppendFiles().Visibility("private")
	img := forms.ImageUpload("avatar").Avatar().ImageCropAspectRatio("16:9").ImageResizeTargetWidth(400)
	nodes := forms.Describe(nil, []forms.Component{f, img})
	d, a := nodes[0], nodes[1]
	if d.MinFiles != 2 || d.MinSize != 1 || !d.Downloadable || !d.Openable || !d.Reorderable || !d.AppendFiles || !d.Private {
		t.Fatalf("file node: %+v", d)
	}
	if !a.Avatar || a.CropAspectRatio != "16:9" || a.ImageResizeWidth != 400 || a.Private {
		t.Fatalf("image node: %+v", a)
	}
	if f.GetDisk() != "local" {
		t.Fatalf("private files go to the local disk: %q", f.GetDisk())
	}
	if msg := f.CheckUpload("a.pdf", "application/pdf", 10); !strings.Contains(msg, "at least 1 KB") {
		t.Fatalf("MinSize: %q", msg)
	}
	_, err := forms.Validate(nil, []forms.Component{f}, map[string]any{"docs": []any{"a"}}, true)
	var verr *forms.ValidationError
	if !errors.As(err, &verr) || !strings.Contains(verr.Errors["docs"], "at least 2 files") {
		t.Fatal("MinFiles should fail with one file")
	}
}
