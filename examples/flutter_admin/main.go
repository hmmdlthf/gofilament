package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/flutterclient"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func ensureFlutterClientReady(targetDir string, panelName, apiBaseURL string) error {
	pubspecPath := filepath.Join(targetDir, "pubspec.yaml")
	if _, err := os.Stat(pubspecPath); os.IsNotExist(err) {
		fmt.Printf("📦 Scaffolding Mobile-First Flutter admin client in %s...\n", targetDir)
		if err := flutterclient.GenerateFlutterAdmin(targetDir, panelName, apiBaseURL); err != nil {
			return fmt.Errorf("failed to scaffold flutter client: %w", err)
		}
	}

	// Prepare pre-compiled web SPA dist directory
	webDistDir := filepath.Join(targetDir, "build", "web")
	if err := os.MkdirAll(webDistDir, 0755); err != nil {
		return err
	}

	distIndexPath := filepath.Join(webDistDir, "index.html")
	if _, err := os.Stat(distIndexPath); os.IsNotExist(err) {
		// Embed a responsive Flutter Material 3 Web application shell
		spaHTML := `<!DOCTYPE html>
<html lang="en">
<head>
  <base href="/admin/">
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>Pulse Fleet - Flutter Multiplatform Admin</title>
  <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://fonts.googleapis.com/icon?family=Material+Icons">
  <style>
    :root {
      --primary: #f59e0b;
      --bg: #0f172a;
      --card-bg: #1e293b;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --border: #334155;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; font-family: 'Roboto', sans-serif; }
    body { background-color: var(--bg); color: var(--text); min-height: 100vh; display: flex; flex-direction: column; }
    .app-bar { height: 56px; background-color: var(--card-bg); border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; padding: 0 16px; position: sticky; top: 0; z-index: 100; }
    .brand { display: flex; align-items: center; gap: 8px; font-weight: 700; font-size: 18px; color: var(--primary); }
    .chip { font-size: 11px; padding: 2px 8px; border-radius: 999px; background: rgba(245, 158, 11, 0.15); color: var(--primary); border: 1px solid var(--primary); }
    .main-layout { display: flex; flex: 1; min-height: calc(100vh - 56px); }
    .nav-rail { width: 72px; background-color: var(--card-bg); border-right: 1px solid var(--border); display: flex; flex-direction: column; align-items: center; padding: 16px 0; gap: 16px; }
    .rail-btn { width: 48px; height: 48px; border-radius: 12px; display: flex; flex-direction: column; align-items: center; justify-content: center; color: var(--text-muted); cursor: pointer; border: none; background: transparent; transition: all 0.2s; font-size: 10px; gap: 2px; }
    .rail-btn.active { background: rgba(245, 158, 11, 0.2); color: var(--primary); }
    .content-pane { flex: 1; padding: 24px; overflow-y: auto; max-width: 1200px; margin: 0 auto; width: 100%; }
    .kpi-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 24px; }
    .kpi-card { background: var(--card-bg); border: 1px solid var(--border); border-radius: 16px; padding: 20px; display: flex; flex-direction: column; gap: 8px; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); }
    .kpi-val { font-size: 28px; font-weight: 700; color: #fff; }
    .kpi-label { font-size: 12px; color: var(--text-muted); }
    .section-title { font-size: 20px; font-weight: 700; margin-bottom: 16px; display: flex; align-items: center; justify-content: space-between; }
    .card { background: var(--card-bg); border: 1px solid var(--border); border-radius: 16px; overflow: hidden; margin-bottom: 24px; }
    .table-container { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
    table { width: 100%; border-collapse: collapse; text-align: left; font-size: 14px; min-width: 600px; }
    th { padding: 12px 16px; background: rgba(0,0,0,0.2); color: var(--text-muted); font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; border-bottom: 1px solid var(--border); }
    td { padding: 14px 16px; border-bottom: 1px solid var(--border); }
    tr:hover { background-color: rgba(255,255,255,0.02); }
    .badge { display: inline-block; padding: 4px 10px; border-radius: 9999px; font-size: 11px; font-weight: 600; text-transform: uppercase; }
    .badge-online { background: rgba(16, 185, 129, 0.2); color: #34d399; }
    .badge-offline { background: rgba(239, 68, 68, 0.2); color: #f87171; }
    .badge-maint { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
    .btn { padding: 8px 16px; border-radius: 8px; font-size: 13px; font-weight: 600; cursor: pointer; border: none; display: inline-flex; align-items: center; gap: 6px; }
    .btn-primary { background: var(--primary); color: #000; }
    .mobile-bottom-bar { display: none; height: 56px; background: var(--card-bg); border-top: 1px solid var(--border); position: sticky; bottom: 0; align-items: center; justify-content: space-around; }
    @media (max-width: 768px) {
      .nav-rail { display: none; }
      .mobile-bottom-bar { display: flex; }
      .content-pane { padding: 16px; }
      .kpi-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="app-bar">
    <div class="brand">
      <span class="material-icons" style="color: var(--primary);">devices</span>
      <span>Pulse Fleet</span>
      <span class="chip">Flutter Multiplatform</span>
    </div>
    <div style="display: flex; gap: 8px; align-items: center;">
      <a href="/admin/api/schema" target="_blank" class="btn" style="background: rgba(255,255,255,0.1); color: #fff;">
        <span class="material-icons" style="font-size: 16px;">api</span> JSON API
      </a>
    </div>
  </div>

  <div class="main-layout">
    <aside class="nav-rail">
      <button class="rail-btn active" onclick="switchTab('devices', this)">
        <span class="material-icons">router</span>
        <span>Devices</span>
      </button>
      <button class="rail-btn" onclick="switchTab('operators', this)">
        <span class="material-icons">badge</span>
        <span>Team</span>
      </button>
    </aside>

    <main class="content-pane">
      <div class="kpi-grid">
        <div class="kpi-card">
          <div class="kpi-label">Connected Hardware Fleet</div>
          <div class="kpi-val" id="kpi-devices">5 Active</div>
          <div style="color: #34d399; font-size: 12px;">● 4 Online Devices (80%)</div>
        </div>
        <div class="kpi-card">
          <div class="kpi-label">Network Ping Latency</div>
          <div class="kpi-val">18 ms</div>
          <div style="color: var(--text-muted); font-size: 12px;">Ultra-fast edge response</div>
        </div>
        <div class="kpi-card">
          <div class="kpi-label">Kernel Engine</div>
          <div class="kpi-val" style="color: var(--primary);">GoLaravel</div>
          <div style="color: var(--text-muted); font-size: 12px;">Zero-allocation Radix Router</div>
        </div>
      </div>

      <div id="pane-devices">
        <div class="section-title">
          <span>IoT Hardware Nodes</span>
          <button class="btn btn-primary" onclick="alert('To run full interactive Material 3 app natively: cd examples/flutter_admin/client && flutter run')">
            <span class="material-icons" style="font-size: 16px;">add</span> Add Device
          </button>
        </div>
        <div class="card">
          <div class="table-container">
            <table>
              <thead>
                <tr>
                  <th>Device Name</th>
                  <th>Status</th>
                  <th>Serial Number</th>
                  <th>Battery</th>
                  <th>Location</th>
                </tr>
              </thead>
              <tbody id="devices-table-body">
                <tr><td colspan="5" style="text-align: center; color: var(--text-muted);">Loading live data from GoFilament REST API...</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div id="pane-operators" style="display: none;">
        <div class="section-title">
          <span>Operations Engineers</span>
        </div>
        <div class="card">
          <div class="table-container">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Email</th>
                  <th>Access Role</th>
                  <th>Department</th>
                </tr>
              </thead>
              <tbody id="operators-table-body">
                <tr><td colspan="4" style="text-align: center; color: var(--text-muted);">Loading team data...</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div style="margin-top: 32px; padding: 20px; border-radius: 12px; background: rgba(0,0,0,0.3); border: 1px dashed var(--border);">
        <h4 style="margin-bottom: 8px; color: var(--primary);">🚀 Native Multiplatform Compilation</h4>
        <p style="font-size: 13px; color: var(--text-muted); line-height: 1.6;">
          This admin panel is fully backed by GoFilament's JSON REST API. To run it as a compiled native desktop application or mobile iOS/Android app:
        </p>
        <pre style="background: #090d16; padding: 12px; border-radius: 8px; margin-top: 8px; font-size: 12px; color: #38bdf8; overflow-x: auto;">cd examples/flutter_admin/client && flutter run</pre>
      </div>
    </main>
  </div>

  <nav class="mobile-bottom-bar">
    <button class="rail-btn active" onclick="switchTab('devices', this)"><span class="material-icons">router</span></button>
    <button class="rail-btn" onclick="switchTab('operators', this)"><span class="material-icons">badge</span></button>
  </nav>

  <script>
    function switchTab(name, btn) {
      document.querySelectorAll('.rail-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      document.getElementById('pane-devices').style.display = name === 'devices' ? 'block' : 'none';
      document.getElementById('pane-operators').style.display = name === 'operators' ? 'block' : 'none';
    }

    async function fetchDevices() {
      try {
        const res = await fetch('/admin/api/resources/devices');
        if (res.ok) {
          const data = await res.json();
          const items = data.data || [];
          document.getElementById('kpi-devices').innerText = items.length + ' Active';
          const tbody = document.getElementById('devices-table-body');
          tbody.innerHTML = items.map(d => {
            const badgeClass = d.status === 'online' ? 'badge-online' : d.status === 'offline' ? 'badge-offline' : 'badge-maint';
            return '<tr>' +
              '<td style="font-weight: 600;">' + d.name + '</td>' +
              '<td><span class="badge ' + badgeClass + '">' + d.status + '</span></td>' +
              '<td style="font-family: monospace;">' + d.serial_no + '</td>' +
              '<td>' + d.battery_level + '%</td>' +
              '<td>' + d.location + '</td>' +
              '</tr>';
          }).join('');
        }
      } catch (e) {
        console.error('API load error:', e);
      }
    }

    async function fetchOperators() {
      try {
        const res = await fetch('/admin/api/resources/users');
        if (res.ok) {
          const data = await res.json();
          const items = data.data || [];
          const tbody = document.getElementById('operators-table-body');
          tbody.innerHTML = items.map(u => {
            return '<tr>' +
              '<td style="font-weight: 600;">' + u.name + '</td>' +
              '<td>' + u.email + '</td>' +
              '<td><span class="chip">' + u.role + '</span></td>' +
              '<td>' + u.department + '</td>' +
              '</tr>';
          }).join('');
        }
      } catch (e) {}
    }

    fetchDevices();
    fetchOperators();
  </script>
</body>
</html>`
		if err := os.WriteFile(distIndexPath, []byte(spaHTML), 0644); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// 1. Seed in-memory databases
	seedDatabase()

	// 2. Prepare Flutter client project & web build files
	clientDir := "examples/flutter_admin/client"
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	apiBaseURL := fmt.Sprintf("http://localhost:%s/admin", port)

	if err := ensureFlutterClientReady(clientDir, "Pulse", apiBaseURL); err != nil {
		log.Printf("⚠️ Flutter client setup notice: %v", err)
	}

	// 3. Initialize GoLaravel Default App
	app := golaravel.Default()

	// 4. Build Admin Panel in Flutter Multiplatform Engine Mode
	adminPanel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("Pulse Fleet").
		Engine(filament.EngineFlutter).
		Flutter(filepath.Join(clientDir, "build", "web")).
		Colors(map[string]any{
			colors.RolePrimary: colors.Amber,
			colors.RoleGray:    colors.Slate,
		}).
		NavigationGroups([]string{
			"IoT Fleet Operations",
			"Operations & Engineers",
		})

	// 5. Mount Resources
	adminPanel.Resource(buildDeviceResource())
	adminPanel.Resource(buildFleetUserResource())

	// 6. Register Panel into GoLaravel
	if err := adminPanel.Register(app); err != nil {
		log.Fatalf("Failed to mount admin panel: %v", err)
	}

	// 7. Route / to /admin
	app.GET("/", func(c *gcontext.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/admin")
	})

	fmt.Printf("\n⚡ Pulse Fleet (Flutter Multiplatform Admin) running at http://localhost:%s/admin\n", port)
	fmt.Printf("📱 Mobile-first client project located in: %s\n", clientDir)
	fmt.Printf("📲 To run natively on iOS / Android / Desktop: cd %s && flutter run\n\n", clientDir)

	if err := app.Run(":" + port); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
