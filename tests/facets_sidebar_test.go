package tests

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type shirt struct {
	ID    string `json:"id"`
	Color string `json:"color"`
	Size  string `json:"size"`
}

var shirts = []*shirt{
	{"1", "red", "S"}, {"2", "red", "M"}, {"3", "blue", "M"}, {"4", "blue", "L"}, {"5", "green", "M"},
}

// shirtResource filters like a real backend: a refined filter's value is a
// comma-separated list of accepted values.
func shirtResource() *filament.BaseResource[*shirt] {
	res := filament.NewResource[*shirt]("shirts").RecordID(func(s *shirt) string { return s.ID })
	res.Table(func(t *tables.Table[*shirt]) *tables.Table[*shirt] {
		return t.Columns(tables.TextColumn[*shirt]("color"), tables.TextColumn[*shirt]("size")).
			Filters(
				tables.RefinedFilter("color").Options(map[string]string{"red": "Red", "blue": "Blue", "green": "Green"}),
				tables.RefinedFilter("size"),
			)
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*shirt, int, error) {
		var out []*shirt
		for _, s := range shirts {
			if v := filters["color"]; v != "" && !strings.Contains(","+v+",", ","+s.Color+",") {
				continue
			}
			if v := filters["size"]; v != "" && !strings.Contains(","+v+",", ","+s.Size+",") {
				continue
			}
			out = append(out, s)
		}
		return out, len(out), nil
	})
	return res
}

func TestRefinedFilterFacetsIgnoreTheirOwnFilter(t *testing.T) {
	res := shirtResource()
	facets := res.ListFacets(nil, "", map[string]string{"color": "red"})
	// size counts apply the color filter...
	if got := facets["size"]; got["S"] != 1 || got["M"] != 1 || got["L"] != 0 {
		t.Fatalf("size facets with color=red: %v", got)
	}
	// ...but color counts ignore it, so other colors still show their counts.
	if got := facets["color"]; got["red"] != 2 || got["blue"] != 2 || got["green"] != 1 {
		t.Fatalf("color facets: %v", got)
	}

	res.FacetCounts(func(ctx *filament.Context, column, search string, filters map[string]string) (map[string]int, error) {
		return map[string]int{"custom": 7}, nil
	})
	if got := res.ListFacets(nil, "", nil)["color"]["custom"]; got != 7 {
		t.Fatalf("FacetCounts not used: %v", got)
	}
}

func TestListResponseCarriesFacets(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter)
	p.Resource(shirtResource())
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/api/resources/shirts?filter_size=M", nil))
	var out struct {
		Total  int                       `json:"total"`
		Facets map[string]map[string]int `json:"facets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if out.Total != 3 || out.Facets["color"]["blue"] != 1 || out.Facets["size"]["L"] != 1 {
		t.Fatalf("list: total %d facets %v", out.Total, out.Facets)
	}
}

func TestRefinedOptionsAreOrderedByLabel(t *testing.T) {
	f := tables.RefinedFilter("x").Options(map[string]string{"c": "Cherry", "a": "apple", "b": "Banana"})
	var got []string
	for _, o := range f.GetOptions() {
		got = append(got, o.Label)
	}
	if strings.Join(got, ",") != "apple,Banana,Cherry" {
		t.Fatalf("order: %v", got)
	}
}

func TestSidebarModes(t *testing.T) {
	p := filament.NewPanel("admin")
	if got := p.SidebarModes(); len(got) != 3 || p.SidebarStartMode() != filament.SidebarExpanded {
		t.Fatalf("defaults: %v %s", got, p.SidebarStartMode())
	}
	p.SidebarCollapseModes(filament.SidebarIcons).SidebarDefaultMode(filament.SidebarIcons)
	if got := p.SidebarModes(); len(got) != 2 || got[1] != filament.SidebarIcons || p.SidebarStartMode() != filament.SidebarIcons {
		t.Fatalf("icons only: %v %s", got, p.SidebarStartMode())
	}
	p.SidebarDefaultMode(filament.SidebarHidden) // not offered: falls back
	if p.SidebarStartMode() != filament.SidebarExpanded {
		t.Fatalf("unoffered default: %s", p.SidebarStartMode())
	}
	p.SidebarCollapsibleOnDesktop(false)
	if got := p.SidebarModes(); len(got) != 1 {
		t.Fatalf("not collapsible: %v", got)
	}
}
