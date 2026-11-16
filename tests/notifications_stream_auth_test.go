package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestNotificationStreamRequiresAuth(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").DatabaseNotifications().Login().AuthValidator(loginValidator)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "http://example.com/admin/notifications/stream", nil)
	req.Header.Set("X-User-ID", "victim@example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("anonymous stream must be refused (redirect to login), got %d", rec.Code)
	}
}
