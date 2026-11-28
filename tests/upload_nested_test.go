package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/storage"
	gltest "github.com/golaravel/golaravel/pkg/testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

type track struct {
	Title string `json:"title"`
	Cover string `json:"cover"`
}

type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Tracks []track `json:"tracks"`
}

type albumNote struct {
	ID             string `json:"id"`
	Body           string `json:"body"`
	Attachment     string `json:"attachment"`
	AttachmentName string `json:"attachment_name"`
}

type albumStore struct {
	mu     sync.Mutex
	albums map[string]*album
	notes  map[string]*albumNote
}

func fill(dst any, data map[string]any) {
	b, _ := json.Marshal(data)
	_ = json.Unmarshal(b, dst)
}

func newAlbumApp(t *testing.T) (*golaravel.Engine, *albumStore, *gltest.FakeDisk) {
	t.Helper()
	disk := gltest.FakeStorage(t, "albums")
	disk.SetVisibility(storage.VisibilityPublic)
	st := &albumStore{albums: map[string]*album{"1": {ID: "1", Title: "Blue"}}, notes: map[string]*albumNote{}}

	notes := filament.NewRelationManager[*albumNote]("notes", "Notes").
		RecordID(func(n *albumNote) string { return n.ID }).
		Form(func(f *forms.Form) *forms.Form {
			return f.Schema(
				forms.TextInput("body"),
				forms.FileUpload("attachment").Disk("albums").Directory("notes").StoreFileNamesIn("attachment_name"),
			)
		}).
		Table(func(tb *tables.Table[*albumNote]) *tables.Table[*albumNote] {
			return tb.Columns(tables.TextColumn[*albumNote]("body"), tables.ImageColumn[*albumNote]("attachment").Disk("albums"))
		}).
		OnList(func(ctx *filament.Context, parentID, search, sort, dir string, page, perPage int) ([]*albumNote, int, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			var out []*albumNote
			for _, n := range st.notes {
				out = append(out, n)
			}
			return out, len(out), nil
		}).
		OnFind(func(ctx *filament.Context, parentID, childID string) (*albumNote, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			if n, ok := st.notes[childID]; ok {
				cp := *n
				return &cp, nil
			}
			return nil, errors.New("not found")
		}).
		OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error {
			st.mu.Lock()
			defer st.mu.Unlock()
			n := &albumNote{ID: fmt.Sprint(len(st.notes) + 1)}
			fill(n, data)
			st.notes[n.ID] = n
			return nil
		}).
		OnUpdate(func(ctx *filament.Context, parentID, childID string, data map[string]any) error {
			st.mu.Lock()
			defer st.mu.Unlock()
			fill(st.notes[childID], data)
			return nil
		})

	res := filament.NewResource[*album]("albums")
	res.RecordID(func(a *album) string { return a.ID })
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("title"),
			forms.Repeater("tracks").Schema(
				forms.TextInput("title"),
				forms.ImageUpload("cover").Disk("albums").Directory("covers"),
			),
		)
	})
	res.Relations(notes)
	res.OnFind(func(ctx *filament.Context, id string) (*album, error) {
		st.mu.Lock()
		defer st.mu.Unlock()
		if a, ok := st.albums[id]; ok {
			cp := *a
			cp.Tracks = append([]track(nil), a.Tracks...)
			return &cp, nil
		}
		return nil, errors.New("not found")
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		st.mu.Lock()
		defer st.mu.Unlock()
		a := st.albums[id]
		a.Tracks = nil
		fill(a, data)
		return nil
	})

	p := filament.NewPanel("albums").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
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
	return app, st, disk
}

func uploadTo(t *testing.T, app http.Handler, path string, f gltest.FakeFile) string {
	t.Helper()
	res := gltest.New(t, app).Request("POST", path).WithToken("good").WithUpload("file", f).Send()
	res.AssertStatus(http.StatusCreated)
	var out map[string]any
	_ = json.Unmarshal(res.Body(), &out)
	return fmt.Sprint(out["value"])
}

func TestFilesInsideRepeaters(t *testing.T) {
	app, st, disk := newAlbumApp(t)
	c1 := uploadTo(t, app, "/admin/api/resources/albums/uploads?field=cover", gltest.FakeImage("one.png", 4, 4))
	c2 := uploadTo(t, app, "/admin/api/resources/albums/uploads?field=cover", gltest.FakeImage("two.png", 4, 4))

	code, out := relDo(t, app, "PUT", "/admin/api/resources/albums/1", "good", map[string]any{
		"tracks": []any{map[string]any{"title": "Intro", "cover": c1}, map[string]any{"title": "Outro", "cover": c2}},
	})
	if code >= 300 {
		t.Fatalf("save: %d %v", code, out)
	}
	a := st.albums["1"]
	if len(a.Tracks) != 2 || !strings.HasPrefix(a.Tracks[0].Cover, "covers/") || !strings.HasPrefix(a.Tracks[1].Cover, "covers/") {
		t.Fatalf("covers moved into place: %+v", a.Tracks)
	}
	disk.AssertExists(a.Tracks[0].Cover, a.Tracks[1].Cover).AssertDirectoryEmpty("tmp-uploads")

	// the form lists every row's files under "tracks.cover"
	_, form := relDo(t, app, "GET", "/admin/api/resources/albums/form?id=1", "good", nil)
	files, _ := form["files"].(map[string]any)
	if list, _ := files["tracks.cover"].([]any); len(list) != 2 {
		t.Fatalf("repeater file links: %v", form["files"])
	}

	// rows may be reordered (a row keeps any file of the repeater); a
	// dropped row's file is deleted; a forged path is refused with the row's key
	old := a.Tracks
	code, out = relDo(t, app, "PUT", "/admin/api/resources/albums/1", "good", map[string]any{
		"tracks": []any{map[string]any{"title": "Outro", "cover": old[1].Cover}, map[string]any{"title": "x", "cover": "covers/evil.png"}},
	})
	errs, _ := out["errors"].(map[string]any)
	if code != 422 || errs["tracks.1.cover"] == nil {
		t.Fatalf("forged value in row 1: %d %v", code, out)
	}
	disk.AssertExists(old[0].Cover)
	relDo(t, app, "PUT", "/admin/api/resources/albums/1", "good", map[string]any{
		"tracks": []any{map[string]any{"title": "Outro", "cover": old[1].Cover}},
	})
	disk.AssertMissing(old[0].Cover).AssertExists(old[1].Cover)
}

func TestFilesInRelationManagers(t *testing.T) {
	app, st, disk := newAlbumApp(t)
	base := "/admin/api/resources/albums/1/relations/notes"
	up := uploadTo(t, app, base+"/uploads?field=attachment", gltest.FakeImage("liner notes.png", 4, 4))
	disk.AssertCount("tmp-uploads/albums.notes/attachment", 1)

	// a resource upload can't be used in the relation form
	other := uploadTo(t, app, "/admin/api/resources/albums/uploads?field=cover", gltest.FakeImage("c.png", 4, 4))
	if code, _ := relDo(t, app, "POST", base, "good", map[string]any{"attachment": other}); code != 422 {
		t.Fatalf("cross-form token: %d", code)
	}

	if code, out := relDo(t, app, "POST", base, "good", map[string]any{"body": "hi", "attachment": up}); code >= 300 {
		t.Fatalf("create: %d %v", code, out)
	}
	n := st.notes["1"]
	if !strings.HasPrefix(n.Attachment, "notes/") || n.AttachmentName != "liner notes.png" {
		t.Fatalf("relation file: %+v", n)
	}
	disk.AssertExists(n.Attachment)

	_, form := relDo(t, app, "GET", base+"/form?child_id=1", "good", nil)
	if f, _ := form["files"].(map[string]any); f["attachment"] == nil {
		t.Fatalf("relation form file links: %v", form["files"])
	}
	_, list := relDo(t, app, "GET", base, "good", nil)
	meta, _ := list["row_meta"].(map[string]any)
	if row, _ := meta["1"].(map[string]any); row == nil || row["images"].(map[string]any)["attachment"] != "/storage/albums/"+n.Attachment {
		t.Fatalf("relation row image links: %v", list["row_meta"])
	}

	// replacing it deletes the old file after the save
	oldPath := n.Attachment
	up2 := uploadTo(t, app, base+"/uploads?field=attachment", gltest.FakeImage("v2.png", 4, 4))
	if code, out := relDo(t, app, "PUT", base+"/1", "good", map[string]any{"attachment": up2}); code >= 300 {
		t.Fatalf("update: %d %v", code, out)
	}
	disk.AssertMissing(oldPath).AssertExists(st.notes["1"].Attachment)
}
