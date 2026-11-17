package sqlstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/filament"
)

func TestSavedViews(t *testing.T) {
	s, path := open(t)
	ctx := context.Background()
	v := s.Views()
	save := func(id, owner, name string, shared bool) {
		t.Helper()
		err := v.SaveView(ctx, &filament.SavedView{ID: id, Panel: "admin", Resource: "orders", Name: name, OwnerID: owner, Shared: shared,
			State: filament.ViewState{Search: name, Filters: map[string]string{"status": "paid"}}, CreatedAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
	}
	save("v1", "ann", "Paid", false)
	save("v2", "bob", "Bob's", false)
	save("v3", "bob", "All big", true)

	got, err := reopen(t, path).Views().ListViews(ctx, "admin", "orders", "ann")
	if err != nil || len(got) != 2 || got[0].Name != "All big" || got[1].State.Filters["status"] != "paid" {
		t.Fatalf("ann's views: %v %+v", err, got)
	}
	if other, _ := v.ListViews(ctx, "admin", "customers", "ann"); len(other) != 0 {
		t.Fatalf("other resource: %+v", other)
	}
	save("v1", "ann", "Paid (renamed)", false)
	if one, err := v.GetView(ctx, "v1"); err != nil || one.Name != "Paid (renamed)" {
		t.Fatalf("get: %v %+v", err, one)
	}
	if err := v.DeleteView(ctx, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.GetView(ctx, "v1"); !errors.Is(err, filament.ErrViewNotFound) {
		t.Fatalf("deleted view: %v", err)
	}
}
