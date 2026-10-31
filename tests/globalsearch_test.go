package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/globalsearch"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type MockSearchRecord struct {
	ID    string
	Name  string
	Email string
	Role  string
}

func TestFuzzySearchScoringAndHighlight(t *testing.T) {
	// 1. Exact match
	score, matched := globalsearch.Score("customer", "customer")
	if !matched || score != 1.0 {
		t.Fatalf("expected exact match score 1.0, got %f (matched: %v)", score, matched)
	}

	// 2. Prefix match
	scorePref, matchedPref := globalsearch.Score("cust", "Customer")
	if !matchedPref || scorePref < 0.90 {
		t.Fatalf("expected prefix match score >= 0.90, got %f", scorePref)
	}

	// 3. Substring match
	scoreSub, matchedSub := globalsearch.Score("tom", "Customer")
	if !matchedSub || scoreSub < 0.80 {
		t.Fatalf("expected substring match score >= 0.80, got %f", scoreSub)
	}

	// 4. Subsequence / Acronym match (e.g. 'co' matching 'Customer Order')
	scoreSeq, matchedSeq := globalsearch.Score("co", "Customer Order")
	if !matchedSeq || scoreSeq < 0.70 {
		t.Fatalf("expected subsequence match score >= 0.70, got %f", scoreSeq)
	}

	// 5. Typo tolerance via Levenshtein distance ('custmer' -> 'customer')
	scoreTypo, matchedTypo := globalsearch.Score("custmer", "customer")
	if !matchedTypo || scoreTypo < 0.50 {
		t.Fatalf("expected typo match score >= 0.50, got %f", scoreTypo)
	}

	// 6. Non-match
	_, matchedNone := globalsearch.Score("xyz", "customer")
	if matchedNone {
		t.Fatalf("expected no match for 'xyz' against 'customer'")
	}

	// 7. Substring Highlight
	highlighted := globalsearch.Highlight("Customer Orders", "Order")
	if !strings.Contains(highlighted, `<mark class="spotlight-highlight">Order</mark>`) {
		t.Fatalf("expected highlight mark in: %s", highlighted)
	}

	// 8. Ranking
	items := []string{"Other Record", "Customer Orders", "Custom Page"}
	ranked := globalsearch.Rank(items, "cust", func(s string) string { return s })
	if len(ranked) < 2 {
		t.Fatalf("expected at least 2 ranked items, got %d", len(ranked))
	}
	if !strings.HasPrefix(ranked[0], "Cust") {
		t.Fatalf("expected top ranked item to start with Cust, got %s", ranked[0])
	}
}

func TestSpotlightModalHTMLRendering(t *testing.T) {
	// 1. Trigger HTML
	triggerHTML := globalsearch.RenderTriggerHTML("AdminApp", "/admin", "k")
	if !strings.Contains(triggerHTML, "spotlight-trigger") {
		t.Fatalf("expected spotlight-trigger class in trigger HTML")
	}
	if !strings.Contains(triggerHTML, "⌘K") {
		t.Fatalf("expected ⌘K shortcut badge in trigger HTML")
	}

	// 2. Modal HTML
	modalHTML := globalsearch.RenderModalHTML("AdminApp", "/admin", "k")
	if !strings.Contains(modalHTML, "filament-spotlight-modal") {
		t.Fatalf("expected filament-spotlight-modal in modal HTML")
	}
	if !strings.Contains(modalHTML, "filament-spotlight-input") {
		t.Fatalf("expected search input in modal HTML")
	}
	if !strings.Contains(modalHTML, "filament-spotlight-results") {
		t.Fatalf("expected results container in modal HTML")
	}

	// 3. Results HTML
	groups := []globalsearch.GroupDTO{
		{
			Name:  "Users",
			Count: 1,
			Items: []globalsearch.ItemDTO{
				{
					Title: "John Doe",
					URL:   "/admin/users/1",
					Group: "Users",
					Details: map[string]string{
						"Email": "john@example.com",
					},
					Actions: []globalsearch.ActionDTO{
						{Label: "Edit", URL: "/admin/users/1/edit"},
					},
				},
			},
		},
	}

	renderedResults := globalsearch.RenderResultsHTML(groups, "john")
	if !strings.Contains(renderedResults, "USERS") {
		t.Fatalf("expected USERS group header in results HTML")
	}
	if !strings.Contains(renderedResults, "John") || !strings.Contains(renderedResults, "Doe") {
		t.Fatalf("expected John Doe in results HTML")
	}
	if !strings.Contains(renderedResults, "john@example.com") {
		t.Fatalf("expected details email in results HTML")
	}
	if !strings.Contains(renderedResults, "Edit") {
		t.Fatalf("expected quick action button in results HTML")
	}
}

func TestPanelGlobalSearchConfiguration(t *testing.T) {
	p := filament.NewPanel("admin").
		GlobalSearch(true).
		GlobalSearchKeyBinding("p").
		GlobalSearchResultsLimit(25).
		GlobalSearchFuzzy(true)

	if p.GetGlobalSearchKeyBinding() != "p" {
		t.Fatalf("expected key binding 'p', got '%s'", p.GetGlobalSearchKeyBinding())
	}
	if p.GetGlobalSearchResultsLimit() != 25 {
		t.Fatalf("expected limit 25, got %d", p.GetGlobalSearchResultsLimit())
	}
	if !p.GetGlobalSearchFuzzy() {
		t.Fatalf("expected fuzzy search to be true")
	}
}

func TestGlobalSearchWebAndRESTEndpoints(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").
		Path("/admin").
		GlobalSearch(true).
		GlobalSearchKeyBinding("k")

	users := []*MockSearchRecord{
		{ID: "usr_1", Name: "Alice Smith", Email: "alice@example.com", Role: "Admin"},
		{ID: "usr_2", Name: "Bob Jones", Email: "bob@example.com", Role: "Manager"},
	}

	userResource := filament.NewResource[*MockSearchRecord]("users").
		ModelName("User").
		PluralModelName("Users").
		GloballySearchable(true).
		RecordID(func(r *MockSearchRecord) string { return r.ID }).
		GlobalSearchResultTitle(func(r *MockSearchRecord) string { return r.Name }).
		GlobalSearchResultDetails(func(r *MockSearchRecord) map[string]string {
			return map[string]string{
				"Email": r.Email,
				"Role":  r.Role,
			}
		}).
		GlobalSearchResultActions(func(r *MockSearchRecord) []filament.GlobalSearchAction {
			return []filament.GlobalSearchAction{
				{Label: "Edit", URL: "/admin/users/" + r.ID + "/edit"},
			}
		}).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*MockSearchRecord, int, error) {
			var out []*MockSearchRecord
			for _, u := range users {
				if search == "" || strings.Contains(strings.ToLower(u.Name), strings.ToLower(search)) || strings.Contains(strings.ToLower(u.Email), strings.ToLower(search)) {
					out = append(out, u)
				}
			}
			return out, len(out), nil
		})

	panel.Resource(userResource)
	_ = panel.Register(app)

	// 1. Web HTML search endpoint (Spotlight AJAX)
	reqHTML := httptest.NewRequest(http.MethodGet, "/admin/global-search?query=alice", nil)
	recHTML := httptest.NewRecorder()
	app.ServeHTTP(recHTML, reqHTML)

	if recHTML.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for HTML search, got %d", recHTML.Code)
	}
	htmlBody := recHTML.Body.String()
	if !strings.Contains(htmlBody, "Alice") || !strings.Contains(htmlBody, "Smith") {
		t.Fatalf("expected 'Alice Smith' in HTML search results, got:\n%s", htmlBody)
	}
	if !strings.Contains(htmlBody, "alice@example.com") {
		t.Fatalf("expected 'alice@example.com' details tag in HTML search results")
	}
	if !strings.Contains(htmlBody, "spotlight-result-item") {
		t.Fatalf("expected spotlight-result-item in HTML search results")
	}

	// 2. Web JSON search endpoint (backward compatibility)
	reqJSON := httptest.NewRequest(http.MethodGet, "/admin/global-search?query=bob", nil)
	reqJSON.Header.Set("Accept", "application/json")
	recJSON := httptest.NewRecorder()
	app.ServeHTTP(recJSON, reqJSON)

	if recJSON.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for JSON search, got %d", recJSON.Code)
	}
	var jsonResults []filament.GlobalSearchResult
	if err := json.NewDecoder(recJSON.Body).Decode(&jsonResults); err != nil {
		t.Fatalf("failed to decode JSON search results: %v", err)
	}
	if len(jsonResults) == 0 || jsonResults[0].Title != "Bob Jones" {
		t.Fatalf("expected Bob Jones in JSON results, got %+v", jsonResults)
	}

	// 3. REST API endpoint /admin/api/global-search
	reqAPI := httptest.NewRequest(http.MethodGet, "/admin/api/global-search?query=alice", nil)
	recAPI := httptest.NewRecorder()
	app.ServeHTTP(recAPI, reqAPI)

	if recAPI.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from REST API search, got %d", recAPI.Code)
	}
	var apiResp struct {
		Query   string                       `json:"query"`
		Total   int                          `json:"total"`
		Groups  []filament.GlobalSearchGroup `json:"groups"`
		Results []filament.GlobalSearchResult `json:"results"`
	}
	if err := json.NewDecoder(recAPI.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode REST API search JSON: %v", err)
	}
	if apiResp.Total == 0 {
		t.Fatalf("expected at least 1 result in REST API search, got 0")
	}
	if len(apiResp.Groups) == 0 {
		t.Fatalf("expected categorized groups in REST API search")
	}

	// 4. REST API Schema metadata
	reqSchema := httptest.NewRequest(http.MethodGet, "/admin/api/schema", nil)
	recSchema := httptest.NewRecorder()
	app.ServeHTTP(recSchema, reqSchema)

	if recSchema.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from schema endpoint, got %d", recSchema.Code)
	}
	var schemaResp map[string]any
	if err := json.NewDecoder(recSchema.Body).Decode(&schemaResp); err != nil {
		t.Fatalf("failed to decode schema JSON: %v", err)
	}

	gsMeta, ok := schemaResp["global_search"].(map[string]any)
	if !ok || gsMeta == nil {
		t.Fatalf("expected global_search in schema JSON, got %v", schemaResp["global_search"])
	}
	if gsMeta["enabled"] != true {
		t.Fatalf("expected global_search.enabled to be true")
	}
	if gsMeta["key_binding"] != "k" {
		t.Fatalf("expected global_search.key_binding to be 'k', got %v", gsMeta["key_binding"])
	}
}

func TestSpotlightModalInjectedIntoAdminPages(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").
		Path("/admin").
		GlobalSearch(true)

	_ = panel.Register(app)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from dashboard, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Topbar trigger present
	if !strings.Contains(body, "spotlight-trigger") {
		t.Fatalf("expected spotlight-trigger in topbar")
	}

	// Modal markup present in body
	if !strings.Contains(body, "filament-spotlight-modal") {
		t.Fatalf("expected filament-spotlight-modal markup in page body")
	}
	if !strings.Contains(body, "filament-spotlight-input") {
		t.Fatalf("expected filament-spotlight-input in modal")
	}
}
