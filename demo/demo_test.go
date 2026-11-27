package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/golaravel/golaravel/pkg/golaravel"
)

// testApp serves the demo panel's API (no Flutter build needed for that).
func testApp(t *testing.T) http.Handler {
	t.Helper()
	resetStore(time.Now())
	resetFeatures()
	app := golaravel.New()
	fsys := fstest.MapFS{"index.html": {Data: []byte(`<html><head><base href="/"></head></html>`)}}
	if err := newPanel("admin", "/admin").FlutterFS(fsys).Register(app); err != nil {
		t.Fatal(err)
	}
	app.GET("/invoices/:id", invoiceHandler)
	return sandbox(app)
}

func call(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "203.0.113.9:1234"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, h http.Handler, email string) string {
	t.Helper()
	rec := call(t, h, "POST", "/admin/api/login", "", `{"email":"`+email+`","password":"demo"}`)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	tok, _ := out["token"].(string)
	if rec.Code != http.StatusOK || tok == "" {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body.String())
	}
	return tok
}

func TestSeedIsDeterministicAndPlausible(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	resetStore(now)
	mu.RLock()
	np, no, nv := len(products.rows), len(orders.rows), len(variants.rows)
	first := *orders.rows["1"]
	mu.RUnlock()
	if np != len(catalog) || no < 500 || nv < np {
		t.Fatalf("seed sizes: %d products, %d orders, %d variants", np, no, nv)
	}
	resetStore(now)
	mu.RLock()
	again := *orders.rows["1"]
	mu.RUnlock()
	if first.Number != again.Number || first.Total != again.Total || first.CustomerID != again.CustomerID {
		t.Fatalf("seed changed between resets: %+v vs %+v", first, again)
	}
}

func TestResetUndoesChanges(t *testing.T) {
	resetStore(time.Now())
	mu.Lock()
	products.rows["1"].Name = "Vandalised"
	delete(customers.rows, "1")
	mu.Unlock()
	resetStore(time.Now())
	mu.RLock()
	defer mu.RUnlock()
	if products.rows["1"].Name == "Vandalised" || customers.rows["1"] == nil {
		t.Fatal("reset did not restore the sample data")
	}
	if until := time.Until(lastReset.Add(resetEvery)); until < 59*time.Minute {
		t.Fatalf("next reset in %s", until)
	}
}

func TestListFiltersSearchesSortsAndPages(t *testing.T) {
	resetStore(time.Now())
	mu.RLock()
	defer mu.RUnlock()
	rows, total := list(products.all(), listQuery{search: "laptop", fields: []string{"name"}, trashed: trashed})
	if total != 2 || len(rows) != 2 {
		t.Fatalf("search laptop: %d", total)
	}
	rows, _ = list(products.all(), listQuery{sortCol: "price", sortDir: "desc", perPage: 1, trashed: trashed})
	if rows[0].Name != "Atlas 16 Laptop" {
		t.Fatalf("most expensive: %s", rows[0].Name)
	}
	_, active := list(products.all(), listQuery{filters: map[string]string{"status": "active"}, trashed: trashed})
	_, all := list(products.all(), listQuery{trashed: trashed})
	_, gone := list(products.all(), listQuery{filters: map[string]string{"trashed": "only"}, trashed: trashed})
	if active >= all || gone != 1 {
		t.Fatalf("filters: active %d of %d, trashed %d", active, all, gone)
	}
	_, featured := list(products.all(), listQuery{filters: map[string]string{"featured": "1"}, trashed: trashed})
	if featured == 0 || featured >= all {
		t.Fatalf("featured filter: %d of %d", featured, all)
	}
}

func TestApplyConvertsFormValues(t *testing.T) {
	p := &Product{Name: "x", Price: 1, Stock: 2}
	if err := apply(p, map[string]any{"price": "19.50", "stock": "7", "featured": "on", "name": "Lamp", "id": "999", "nope": 1}); err != nil {
		t.Fatal(err)
	}
	if p.Price != 19.5 || p.Stock != 7 || !p.Featured || p.Name != "Lamp" || p.ID != "" {
		t.Fatalf("applied: %+v", p)
	}
}

func TestRolesLimitWarehouseStaff(t *testing.T) {
	h := testApp(t)
	admin := login(t, h, "admin@example.com")
	staff := login(t, h, "warehouse@example.com")

	for _, path := range []string{"/admin/api/resources/customers", "/admin/api/resources/activity-logs"} {
		if rec := call(t, h, "GET", path, staff, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("staff GET %s: %d", path, rec.Code)
		}
		if rec := call(t, h, "GET", path, admin, ""); rec.Code != http.StatusOK {
			t.Fatalf("admin GET %s: %d", path, rec.Code)
		}
	}
	if rec := call(t, h, "DELETE", "/admin/api/resources/products/1", staff, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("staff delete: %d", rec.Code)
	}
	if rec := call(t, h, "PUT", "/admin/api/resources/products/2", staff, `{"stock":"12"}`); rec.Code != http.StatusOK {
		t.Fatalf("staff update: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, "POST", "/admin/api/login", "", `{"email":"admin@example.com","password":"wrong"}`); rec.Code == http.StatusOK {
		t.Fatal("wrong password accepted")
	}
}

func TestOrdersCannotBeCreatedAndShipLogsHistory(t *testing.T) {
	h := testApp(t)
	admin := login(t, h, "admin@example.com")
	if rec := call(t, h, "POST", "/admin/api/resources/orders", admin, `{"status":"paid"}`); rec.Code < 400 {
		t.Fatalf("order create allowed: %d", rec.Code)
	}
	mu.Lock()
	var paid *Order
	for _, o := range orders.rows {
		if o.Status == "paid" {
			paid = o
			break
		}
	}
	mu.Unlock()
	if paid == nil {
		t.Skip("no paid order in the sample")
	}
	shipOrders([]string{paid.ID}, "test")
	mu.RLock()
	status, tracking := paid.Status, paid.Tracking
	mu.RUnlock()
	if status != "shipped" || tracking == "" {
		t.Fatalf("after ship: %s %q", status, tracking)
	}
}

func TestSandboxLimits(t *testing.T) {
	var got int
	h := sandbox(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, maxBody+10)
		n, _ := r.Body.Read(buf)
		for err := error(nil); err == nil && n < len(buf); {
			var m int
			m, err = r.Body.Read(buf[n:])
			n += m
		}
		got = n
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("x", maxBody+5))))
	if got > maxBody {
		t.Fatalf("read %d bytes past the %d cap", got, maxBody)
	}
	if rec.Header().Get("X-Robots-Tag") == "" {
		t.Fatal("no X-Robots-Tag")
	}

	lim := &limiter{hits: map[string]*bucket{}}
	now := time.Now()
	for i := 0; i < writesPerMin; i++ {
		if !lim.allow("a", true, now) {
			t.Fatalf("write %d refused", i)
		}
	}
	if lim.allow("a", true, now) {
		t.Fatal("write over the limit allowed")
	}
	if !lim.allow("b", true, now) {
		t.Fatal("another address was limited")
	}
	if !lim.allow("a", true, now.Add(limiterWindow+time.Second)) {
		t.Fatal("limit did not reset after the window")
	}
}

func TestClientIPUsesProxyHop(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:5000"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 198.51.100.7")
	if ip := clientIP(r); ip != "198.51.100.7" {
		t.Fatalf("client ip: %s", ip)
	}
}

func TestRefinedFiltersAndFacets(t *testing.T) {
	h := testApp(t)
	admin := login(t, h, "admin@example.com")
	rec := call(t, h, "GET", "/admin/api/resources/orders?filter_status=paid,shipped&per_page=5", admin, "")
	var out struct {
		Total  int                       `json:"total"`
		Data   []map[string]any          `json:"data"`
		Facets map[string]map[string]int `json:"facets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	for _, o := range out.Data {
		if s := o["status"]; s != "paid" && s != "shipped" {
			t.Fatalf("status filter let %v through", s)
		}
	}
	st := out.Facets["status"]
	if st["paid"]+st["shipped"] != out.Total || st["delivered"] == 0 {
		t.Fatalf("status facets %v for total %d", st, out.Total)
	}
	if len(out.Facets["channel"]) == 0 {
		t.Fatalf("no channel facets: %v", out.Facets)
	}
	rec = call(t, h, "GET", "/admin/api/resources/products", admin, "")
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Facets["brand"]["Sonance"] == 0 {
		t.Fatalf("brand facets from data: %v", out.Facets["brand"])
	}
}

func TestInvoiceLinksAreSignedAndRenderFast(t *testing.T) {
	h := testApp(t)
	mu.RLock()
	var id string
	for k := range orders.rows {
		id = k
		break
	}
	mu.RUnlock()
	good := invoiceURL(id, time.Now().Add(time.Hour))
	rec := call(t, h, "GET", good, "", "")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Invoice") || strings.Contains(body, renderTimeMark) {
		t.Fatalf("invoice: %d %.200s", rec.Code, body)
	}
	if !strings.Contains(rec.Header().Get("Server-Timing"), "render") {
		t.Fatal("no Server-Timing header")
	}
	for name, u := range map[string]string{
		"tampered": strings.Replace(good, "sig=", "sig=0", 1),
		"expired":  invoiceURL(id, time.Now().Add(-time.Minute)),
		"other id": strings.Replace(good, "/invoices/"+id, "/invoices/x"+id, 1),
	} {
		if rec := call(t, h, "GET", u, "", ""); rec.Code != http.StatusForbidden {
			t.Fatalf("%s link: %d", name, rec.Code)
		}
	}
}

func TestUSD(t *testing.T) {
	for in, want := range map[float64]string{0: "$0.00", 9.9: "$9.90", 1799: "$1,799.00", 1234567.891: "$1,234,567.89", -1500: "-$1,500.00"} {
		if got := usd(in); got != want {
			t.Errorf("usd(%v) = %s, want %s", in, got, want)
		}
	}
}

func TestProductCategoryCanBeCreatedFromTheSelect(t *testing.T) {
	h := testApp(t)
	admin := login(t, h, "admin@example.com")
	staff := login(t, h, "warehouse@example.com")

	rec := call(t, h, "POST", "/admin/api/resources/products/form/options/create?field=category", admin, `{"name":"Garden"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"value":"Garden"`) {
		t.Fatalf("create category: %d %s", rec.Code, rec.Body.String())
	}
	mu.RLock()
	_, ok := cats.rows["Garden"]
	mu.RUnlock()
	if !ok {
		t.Fatal("category not stored")
	}
	if rec := call(t, h, "POST", "/admin/api/resources/products/form/options/create?field=category", admin, `{"name":"garden"}`); rec.Code < 400 {
		t.Fatalf("duplicate accepted: %d", rec.Code)
	}
	// warehouse staff may not create categories: no create entry, and refused
	if rec := call(t, h, "POST", "/admin/api/resources/products/form/options/create?field=category", staff, `{"name":"Toys"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("staff create: %d", rec.Code)
	}
}

func orderActions(t *testing.T, h http.Handler, token string) string {
	t.Helper()
	rec := call(t, h, "GET", "/admin/api/schema", token, "")
	var out struct {
		Meta map[string]struct {
			Table struct {
				Actions []struct {
					Name string `json:"name"`
				} `json:"actions"`
			} `json:"table"`
		} `json:"resource_meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, a := range out.Meta["orders"].Table.Actions {
		names = append(names, a.Name)
	}
	return strings.Join(names, ",")
}

func TestTenantSettingsToggleOrderActions(t *testing.T) {
	h := testApp(t)
	t.Cleanup(resetFeatures)
	admin := login(t, h, "admin@example.com")
	staff := login(t, h, "warehouse@example.com")

	rec := call(t, h, "GET", "/admin/api/tenant-settings", admin, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"orders.shipping"`) || strings.Contains(rec.Body.String(), `"audit.log"`) {
		t.Fatalf("settings: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, "GET", "/admin/api/tenant-settings", staff, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("staff may not change settings: %d", rec.Code)
	}
	if got := orderActions(t, h, admin); !strings.Contains(got, "ship") || !strings.Contains(got, "invoice") {
		t.Fatalf("actions before: %s", got)
	}
	if rec := call(t, h, "PUT", "/admin/api/tenant-settings", admin, `{"values":{"orders.shipping":false}}`); rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	if got := orderActions(t, h, admin); strings.Contains(got, "ship") || !strings.Contains(got, "invoice") {
		t.Fatalf("actions after turning shipping off: %s", got)
	}
	// The audit log is the platform's to turn off, not the shop's.
	if rec := call(t, h, "PUT", "/admin/api/tenant-settings", admin, `{"values":{"audit.log":false}}`); rec.Code != http.StatusForbidden {
		t.Fatalf("SuperAdmin feature: %d", rec.Code)
	}
	if rec := call(t, h, "GET", "/admin/api/resources/activity-logs", admin, ""); rec.Code != http.StatusOK {
		t.Fatalf("audit log: %d", rec.Code)
	}
}
