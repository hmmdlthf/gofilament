package tests

import (
	"context"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/golaravel/gofilament/pkg/entdriver"
)

// fakePredicate / fakeOrder mirror Ent's generated predicate.X and x.OrderOption types.
type (
	fakePredicate func(*sql.Selector)
	fakeOrder     func(*sql.Selector)
)

// fakeQuery mimics the parts of a generated Ent query builder the adapter uses.
type fakeQuery struct {
	preds     []fakePredicate
	orders    []fakeOrder
	limit     int
	offset    int
	countSQL  *string
	selectSQL *string
}

func (q *fakeQuery) Where(ps ...fakePredicate) *fakeQuery { q.preds = append(q.preds, ps...); return q }
func (q *fakeQuery) Order(os ...fakeOrder) *fakeQuery     { q.orders = append(q.orders, os...); return q }
func (q *fakeQuery) Limit(n int) *fakeQuery               { q.limit = n; return q }
func (q *fakeQuery) Offset(n int) *fakeQuery              { q.offset = n; return q }

func (q *fakeQuery) Clone() *fakeQuery {
	c := *q
	c.preds = append([]fakePredicate(nil), q.preds...)
	c.orders = append([]fakeOrder(nil), q.orders...)
	return &c
}

func (q *fakeQuery) build() (string, []any) {
	sel := sql.Dialect(dialect.Postgres).Select("*").From(sql.Table("products"))
	for _, p := range q.preds {
		p(sel)
	}
	for _, o := range q.orders {
		o(sel)
	}
	if q.limit > 0 {
		sel.Limit(q.limit)
	}
	if q.offset > 0 {
		sel.Offset(q.offset)
	}
	return sel.Query()
}

func (q *fakeQuery) Count(ctx context.Context) (int, error) {
	s, _ := q.build()
	*q.countSQL = s
	return 42, nil
}

func (q *fakeQuery) All(ctx context.Context) ([]*MockUser, error) {
	s, _ := q.build()
	*q.selectSQL = s
	return []*MockUser{{ID: 1, Name: "A"}}, nil
}

func newFakeQuery() (*fakeQuery, *string, *string) {
	var c, s string
	return &fakeQuery{countSQL: &c, selectSQL: &s}, &c, &s
}

func TestEntDriverListSearchSortFilter(t *testing.T) {
	q, countSQL, selectSQL := newFakeQuery()

	records, total, err := entdriver.ExecuteListQuery[*MockUser](context.Background(), q, entdriver.ListOptions{
		Search:        "chain",
		SearchColumns: []string{"name", "plu_code"},
		SortColumn:    "name",
		SortDirection: "desc",
		Filters:       map[string]string{"is_active": "true", "category_id": "7", "_advanced_search": "x", "empty": ""},
	}, 25, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 42 || len(records) != 1 {
		t.Fatalf("unexpected result: total=%d records=%d", total, len(records))
	}

	for _, want := range []string{`"name" ILIKE`, `"plu_code" ILIKE`, ` OR `, `"is_active"`, `"category_id" =`, `ORDER BY "products"."name" DESC`, `LIMIT 25`, `OFFSET 50`} {
		if !strings.Contains(*selectSQL, want) {
			t.Errorf("select SQL is missing %q:\n%s", want, *selectSQL)
		}
	}
	for _, reject := range []string{"_advanced_search", `"empty"`} {
		if strings.Contains(*selectSQL, reject) {
			t.Errorf("select SQL must not contain %q:\n%s", reject, *selectSQL)
		}
	}
	// the total is counted with the filters but without ORDER BY / LIMIT
	if !strings.Contains(*countSQL, "ILIKE") || strings.Contains(*countSQL, "ORDER BY") || strings.Contains(*countSQL, "LIMIT") {
		t.Errorf("unexpected count SQL:\n%s", *countSQL)
	}
}

func TestEntDriverListRejectsUnsafeAndDisallowedColumns(t *testing.T) {
	q, _, selectSQL := newFakeQuery()

	_, _, err := entdriver.ExecuteListQuery[*MockUser](context.Background(), q, entdriver.ListOptions{
		Search:        "x",
		SearchColumns: []string{"name; DROP TABLE users"},
		SortColumn:    `name"; DROP TABLE users; --`,
		SortColumns:   []string{"name"},
		DefaultSort:   "id desc",
		Filters:       map[string]string{"secret": "1", "status": "open", "bad col": "1"},
		FilterColumns: []string{"status"},
	}, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(*selectSQL, "DROP") || strings.Contains(*selectSQL, "secret") || strings.Contains(*selectSQL, "bad col") {
		t.Fatalf("unsafe or disallowed input reached the query:\n%s", *selectSQL)
	}
	if !strings.Contains(*selectSQL, `"status" =`) {
		t.Errorf("allowed filter missing:\n%s", *selectSQL)
	}
	if !strings.Contains(*selectSQL, `ORDER BY "products"."id" DESC`) {
		t.Errorf("default sort should apply when the requested sort is rejected:\n%s", *selectSQL)
	}
}

// formBuilder mimics generated Ent create / update builders.
type formBuilder struct {
	categoryID    *int64
	imageURL      string
	price         float64
	name          string
	cleared       []string
	nameSet       bool
	categoryIDSet bool
}

func (b *formBuilder) SetCategoryID(v int64) *formBuilder { b.categoryID = &v; b.categoryIDSet = true; return b }
func (b *formBuilder) ClearCategoryID() *formBuilder {
	b.cleared = append(b.cleared, "category_id")
	return b
}
func (b *formBuilder) SetImageURL(v string) *formBuilder { b.imageURL = v; return b }
func (b *formBuilder) ClearImageURL() *formBuilder {
	b.cleared = append(b.cleared, "image_url")
	return b
}
func (b *formBuilder) SetRetailPrice(v float64) *formBuilder { b.price = v; return b }
func (b *formBuilder) SetName(v string) *formBuilder          { b.name = v; b.nameSet = true; return b }

func TestEntDriverApplyFormDataInitialismsAndEmptyValues(t *testing.T) {
	b := &formBuilder{}
	err := entdriver.ApplyFormData(b, map[string]any{
		"category_id":  float64(1000000), // JSON number
		"image_url":    "https://example.com/a.png",
		"retail_price": "150.50",
		"name":         "Chain",
		"_token":       "ignored",
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.categoryID == nil || *b.categoryID != 1000000 {
		t.Errorf("category_id not applied through SetCategoryID: %v", b.categoryID)
	}
	if b.imageURL != "https://example.com/a.png" || b.price != 150.50 || b.name != "Chain" {
		t.Errorf("fields not applied: %+v", b)
	}

	// empty optional values are cleared, required strings keep the empty value
	b2 := &formBuilder{}
	if err := entdriver.ApplyFormData(b2, map[string]any{"category_id": "", "image_url": nil, "retail_price": "", "name": ""}); err != nil {
		t.Fatal(err)
	}
	if b2.categoryIDSet || len(b2.cleared) != 2 {
		t.Errorf("expected category_id and image_url to be cleared, got set=%v cleared=%v", b2.categoryIDSet, b2.cleared)
	}
	if b2.price != 0 {
		t.Errorf("empty numeric input must be skipped")
	}
	if !b2.nameSet {
		t.Errorf("empty required string must still be passed to the setter")
	}
}
