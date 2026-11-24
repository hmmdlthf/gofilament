// Command demo is the public GoFilament demo (demo.gofilament.elridhma.com):
// an online store's back office as a Flutter admin app (demo/client, built
// on the gofilament_admin package), served with its API from one Go binary
// at /admin.
//
// Everything lives in memory and goes back to the sample data every hour.
// Sign in with one of the demo accounts on the login screen.
//
//	go run ./demo                 # http://localhost:8080
//	PORT=8095 go run ./demo
//
// The Flutter app is compiled into the binary from demo/flutterweb (see
// demo/README.md).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/filament"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

const brand = "Apex Store"

// newPanel builds one panel over the shared store.
func newPanel(id, path string) *filament.Panel {
	p := filament.NewPanel(id).
		Path(path).
		BrandName(brand).
		BrandLogo(path+"/icons/Icon-192.png"). // demo/client/web/icons
		Favicon(path+"/favicon.png").
		Colors(map[string]any{colors.RolePrimary: colors.Emerald, colors.RoleGray: colors.Zinc}).
		DarkMode(true).
		SidebarCollapsibleOnDesktop().
		SidebarCollapseModes(filament.SidebarIcons, filament.SidebarHidden).
		GlobalSearch().
		// The app's own words come in these languages (Settings → Language);
		// Arabic switches the layout to right-to-left.
		Locales("en", "ar", "de", "es", "fr").
		DefaultLocale("en").
		LocaleLabels(map[string]string{"en": "English", "ar": "العربية", "de": "Deutsch", "es": "Español", "fr": "Français"}).
		NavigationGroups([]string{"Sales", "Catalog", "System"}).
		Login().
		AuthValidator(validateLogin).
		TokenValidator(validateToken).
		LoginThrottle(10, time.Minute).
		Roles(roleStore, filament.RolesConfig{
			FindUsers: func(ctx *filament.Context, query string) ([]filament.UserRef, error) {
				var out []filament.UserRef
				for _, u := range demoUsers {
					out = append(out, filament.UserRef{ID: u.email, Name: u.name + " · " + u.email})
				}
				return out, nil
			},
		})
	p.Widgets(dashboardWidgets()...)
	p.Resource(activitylog.EnableAuditLog(orderResource()))
	p.Resource(activitylog.EnableAuditLog(customerResource()))
	p.Resource(activitylog.EnableAuditLog(productResource()))
	p.Resource(activitylog.NewActivityResource().NavigationGroup("System").NavigationIcon("heroicon-history"))
	return p
}

func buildApp() (http.Handler, error) {
	app := golaravel.New()

	fsys, flutterBuilt := flutterBuild()
	if flutterBuilt {
		if err := newPanel("admin", "/admin").FlutterFS(fsys).Register(app); err != nil {
			return nil, err
		}
	} else {
		app.GET("/admin", func(c *gcontext.Context) error {
			return c.String(http.StatusServiceUnavailable, "The Flutter admin is not built into this binary; see demo/README.md.")
		})
	}

	app.GET("/", func(c *gcontext.Context) error { return c.HTML(http.StatusOK, homePage(flutterBuilt)) })
	app.GET("/healthz", func(c *gcontext.Context) error { return c.String(http.StatusOK, "ok") })
	app.GET("/invoices/:id", invoiceHandler)
	// The Flutter app's banner reads the countdown to the next reset.
	app.GET("/demo/status", func(c *gcontext.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"next_reset_minutes":  int(time.Until(nextReset()).Minutes()) + 1,
			"reset_every_minutes": int(resetEvery.Minutes()),
		})
	})
	return sandbox(app), nil
}

func main() {
	resetStore(time.Now())
	handler, err := buildApp()
	if err != nil {
		log.Fatalf("demo: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go resetLoop(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Printf("%s demo on http://localhost:%s/admin", brand, port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("demo: %v", err)
	}
}

// resetLoop puts the sample data back every resetEvery.
func resetLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(nextReset())):
			resetStore(time.Now())
			log.Print("demo data reset")
		}
	}
}
