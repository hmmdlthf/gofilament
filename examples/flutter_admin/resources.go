package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

// ----------------------------------------------------
// 1. Device Resource
// ----------------------------------------------------

func buildDeviceResource() *filament.BaseResource[*Device] {
	res := filament.NewResource[*Device]("devices").
		ModelName("Device").
		PluralModelName("Devices").
		NavigationGroup("IoT Fleet Operations").
		RecordID(func(d *Device) string {
			if d == nil {
				return ""
			}
			return d.ID
		})

	res.Table(func(t *tables.Table[*Device]) *tables.Table[*Device] {
		return t.
			Columns(
				tables.TextColumn[*Device]("name").
					Label("Device Name").
					Weight("bold").
					Searchable().
					Sortable(),
				tables.BadgeColumn[*Device]("status").
					Label("Status").
					Colors(map[string]string{
						"online":      "success",
						"offline":     "danger",
						"maintenance": "warning",
					}),
				tables.TextColumn[*Device]("serial_no").
					Label("Serial Number").
					Copyable(),
				tables.TextColumn[*Device]("battery_level").
					Label("Battery (%)").
					Sortable(),
				tables.TextColumn[*Device]("location").
					Label("Current Location").
					Searchable(),
			)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Grid(2).Schema(
				forms.TextInput("name").Label("Device Name").Required(),
				forms.TextInput("serial_no").Label("Hardware Serial No").Required(),
				forms.TextInput("model").Label("Hardware Model"),
				forms.TextInput("os").Label("Operating System / RTOS"),
			),
			forms.Grid(3).Schema(
				forms.Select("status").
					Label("Device Status").
					Options(map[string]string{
						"online":      "Online (Active)",
						"offline":     "Offline (Unreachable)",
						"maintenance": "Under Maintenance",
					}),
				forms.TextInput("battery_level").Numeric().Label("Battery Level (%)"),
				forms.TextInput("location").Label("Deployment Location"),
			),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Device, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		searchLower := strings.ToLower(strings.TrimSpace(search))
		var matched []*Device

		for _, d := range devices {
			if searchLower != "" {
				if !strings.Contains(strings.ToLower(d.Name), searchLower) &&
					!strings.Contains(strings.ToLower(d.SerialNo), searchLower) &&
					!strings.Contains(strings.ToLower(d.Location), searchLower) {
					continue
				}
			}
			matched = append(matched, d)
		}

		sort.Slice(matched, func(i, j int) bool {
			return matched[i].ID < matched[j].ID
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*Device{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*Device, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if d, ok := devices[id]; ok {
			return d, nil
		}
		return nil, fmt.Errorf("device %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(devices)+1)
		batt, _ := strconv.Atoi(fmt.Sprintf("%v", data["battery_level"]))

		dev := &Device{
			ID:           newID,
			Name:         fmt.Sprintf("%v", data["name"]),
			SerialNo:     fmt.Sprintf("%v", data["serial_no"]),
			Model:        fmt.Sprintf("%v", data["model"]),
			OS:           fmt.Sprintf("%v", data["os"]),
			BatteryLevel: batt,
			Status:       fmt.Sprintf("%v", data["status"]),
			Location:     fmt.Sprintf("%v", data["location"]),
			LastPing:     time.Now(),
			CreatedAt:    time.Now(),
		}
		if dev.Status == "" || dev.Status == "<nil>" {
			dev.Status = "online"
		}
		devices[newID] = dev
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		d, ok := devices[id]
		if !ok {
			return fmt.Errorf("device %s not found", id)
		}

		if v, exists := data["name"]; exists && v != nil {
			d.Name = fmt.Sprintf("%v", v)
		}
		if v, exists := data["status"]; exists && v != nil {
			d.Status = fmt.Sprintf("%v", v)
		}
		if v, exists := data["location"]; exists && v != nil {
			d.Location = fmt.Sprintf("%v", v)
		}
		if v, exists := data["battery_level"]; exists && v != nil {
			d.BatteryLevel, _ = strconv.Atoi(fmt.Sprintf("%v", v))
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(devices, id)
		return nil
	})

	return res
}

// ----------------------------------------------------
// 2. Fleet User Resource
// ----------------------------------------------------

func buildFleetUserResource() *filament.BaseResource[*FleetUser] {
	res := filament.NewResource[*FleetUser]("users").
		ModelName("Operator").
		PluralModelName("Operators").
		NavigationGroup("Operations & Engineers").
		RecordID(func(u *FleetUser) string {
			if u == nil {
				return ""
			}
			return u.ID
		})

	res.Table(func(t *tables.Table[*FleetUser]) *tables.Table[*FleetUser] {
		return t.
			Columns(
				tables.TextColumn[*FleetUser]("name").
					Label("Engineer / Operator").
					Weight("bold").
					Searchable(),
				tables.TextColumn[*FleetUser]("email").
					Label("Email").
					Copyable(),
				tables.BadgeColumn[*FleetUser]("role").
					Label("Access Role").
					Colors(map[string]string{
						"admin":      "danger",
						"operator":   "primary",
						"technician": "warning",
					}),
				tables.TextColumn[*FleetUser]("department").
					Label("Department"),
			)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Grid(2).Schema(
				forms.TextInput("name").Label("Full Name").Required(),
				forms.TextInput("email").Email().Label("Work Email").Required(),
			),
			forms.Grid(2).Schema(
				forms.Select("role").
					Label("Role").
					Options(map[string]string{
						"admin":      "Fleet Administrator",
						"operator":   "Operations Controller",
						"technician": "Field Hardware Engineer",
					}),
				forms.TextInput("department").Label("Assigned Department"),
			),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*FleetUser, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		searchLower := strings.ToLower(strings.TrimSpace(search))
		var matched []*FleetUser

		for _, u := range fleetUsers {
			if searchLower != "" {
				if !strings.Contains(strings.ToLower(u.Name), searchLower) &&
					!strings.Contains(strings.ToLower(u.Email), searchLower) &&
					!strings.Contains(strings.ToLower(u.Department), searchLower) {
					continue
				}
			}
			matched = append(matched, u)
		}

		sort.Slice(matched, func(i, j int) bool {
			return matched[i].ID < matched[j].ID
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*FleetUser{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*FleetUser, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if u, ok := fleetUsers[id]; ok {
			return u, nil
		}
		return nil, fmt.Errorf("user %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(fleetUsers)+1)
		u := &FleetUser{
			ID:         newID,
			Name:       fmt.Sprintf("%v", data["name"]),
			Email:      fmt.Sprintf("%v", data["email"]),
			Role:       fmt.Sprintf("%v", data["role"]),
			Department: fmt.Sprintf("%v", data["department"]),
			IsActive:   true,
			CreatedAt:  time.Now(),
		}
		fleetUsers[newID] = u
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		u, ok := fleetUsers[id]
		if !ok {
			return fmt.Errorf("user %s not found", id)
		}

		if v, exists := data["name"]; exists && v != nil {
			u.Name = fmt.Sprintf("%v", v)
		}
		if v, exists := data["email"]; exists && v != nil {
			u.Email = fmt.Sprintf("%v", v)
		}
		if v, exists := data["role"]; exists && v != nil {
			u.Role = fmt.Sprintf("%v", v)
		}
		if v, exists := data["department"]; exists && v != nil {
			u.Department = fmt.Sprintf("%v", v)
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(fleetUsers, id)
		return nil
	})

	return res
}

// ----------------------------------------------------
// 3. Telemetry Logs Resource
// ----------------------------------------------------

func buildTelemetryResource() *filament.BaseResource[*TelemetryLog] {
	res := filament.NewResource[*TelemetryLog]("telemetry").
		ModelName("Telemetry").
		PluralModelName("Telemetry Logs").
		NavigationGroup("IoT Fleet Operations").
		RecordID(func(t *TelemetryLog) string {
			if t == nil {
				return ""
			}
			return t.ID
		})

	res.Table(func(t *tables.Table[*TelemetryLog]) *tables.Table[*TelemetryLog] {
		return t.
			Columns(
				tables.TextColumn[*TelemetryLog]("device_name").
					Label("Device Node").
					Weight("bold").
					Searchable().
					Sortable(),
				tables.TextColumn[*TelemetryLog]("metric").
					Label("Sensor Metric").
					Searchable(),
				tables.TextColumn[*TelemetryLog]("value").
					Label("Reading Value").
					Weight("bold"),
				tables.BadgeColumn[*TelemetryLog]("severity").
					Label("Severity").
					Colors(map[string]string{
						"normal":   "success",
						"warning":  "warning",
						"critical": "danger",
					}),
			)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("device_name").Label("Device Name").Required(),
			forms.TextInput("metric").Label("Sensor Metric").Required(),
			forms.TextInput("value").Label("Reading Value").Required(),
			forms.Select("severity").
				Label("Severity").
				Options(map[string]string{
					"normal":   "Normal",
					"warning":  "Warning",
					"critical": "Critical Alert",
				}),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*TelemetryLog, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		searchLower := strings.ToLower(strings.TrimSpace(search))
		var matched []*TelemetryLog

		for _, item := range telemetryLogs {
			if searchLower != "" {
				if !strings.Contains(strings.ToLower(item.DeviceName), searchLower) &&
					!strings.Contains(strings.ToLower(item.Metric), searchLower) &&
					!strings.Contains(strings.ToLower(item.Value), searchLower) {
					continue
				}
			}
			matched = append(matched, item)
		}

		sort.Slice(matched, func(i, j int) bool {
			return matched[i].ID < matched[j].ID
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*TelemetryLog{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*TelemetryLog, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if t, ok := telemetryLogs[id]; ok {
			return t, nil
		}
		return nil, fmt.Errorf("telemetry log %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(telemetryLogs)+1)
		item := &TelemetryLog{
			ID:         newID,
			DeviceName: fmt.Sprintf("%v", data["device_name"]),
			Metric:     fmt.Sprintf("%v", data["metric"]),
			Value:      fmt.Sprintf("%v", data["value"]),
			Severity:   fmt.Sprintf("%v", data["severity"]),
			RecordedAt: time.Now(),
		}
		if item.DeviceName == "" || item.DeviceName == "<nil>" {
			item.DeviceName = fmt.Sprintf("%v", data["name"])
		}
		if item.Severity == "" || item.Severity == "<nil>" {
			item.Severity = "normal"
		}
		telemetryLogs[newID] = item
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		t, ok := telemetryLogs[id]
		if !ok {
			return fmt.Errorf("telemetry log %s not found", id)
		}
		if v, exists := data["value"]; exists && v != nil {
			t.Value = fmt.Sprintf("%v", v)
		}
		if v, exists := data["severity"]; exists && v != nil {
			t.Severity = fmt.Sprintf("%v", v)
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(telemetryLogs, id)
		return nil
	})

	return res
}

// ----------------------------------------------------
// 4. Maintenance Tickets Resource
// ----------------------------------------------------

func buildTicketResource() *filament.BaseResource[*MaintenanceTicket] {
	res := filament.NewResource[*MaintenanceTicket]("tickets").
		ModelName("Maintenance Ticket").
		PluralModelName("Maintenance Tickets").
		NavigationGroup("IoT Fleet Operations").
		RecordID(func(t *MaintenanceTicket) string {
			if t == nil {
				return ""
			}
			return t.ID
		})

	res.Table(func(t *tables.Table[*MaintenanceTicket]) *tables.Table[*MaintenanceTicket] {
		return t.
			Columns(
				tables.TextColumn[*MaintenanceTicket]("title").
					Label("Work Order Title").
					Weight("bold").
					Searchable(),
				tables.BadgeColumn[*MaintenanceTicket]("priority").
					Label("Priority").
					Colors(map[string]string{
						"critical": "danger",
						"high":     "warning",
						"medium":   "info",
						"low":      "secondary",
					}),
				tables.BadgeColumn[*MaintenanceTicket]("status").
					Label("Status").
					Colors(map[string]string{
						"open":        "danger",
						"in_progress": "warning",
						"resolved":    "success",
						"closed":      "secondary",
					}),
				tables.TextColumn[*MaintenanceTicket]("assigned_to").
					Label("Technician").
					Searchable(),
				tables.TextColumn[*MaintenanceTicket]("department").
					Label("Department"),
			)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("title").Label("Ticket Title").Required(),
			forms.Select("priority").
				Label("Priority").
				Options(map[string]string{
					"critical": "Critical",
					"high":     "High",
					"medium":   "Medium",
					"low":      "Low",
				}),
			forms.Select("status").
				Label("Status").
				Options(map[string]string{
					"open":        "Open",
					"in_progress": "In Progress",
					"resolved":    "Resolved",
					"closed":      "Closed",
				}),
			forms.TextInput("assigned_to").Label("Assigned Technician"),
			forms.TextInput("department").Label("Department"),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*MaintenanceTicket, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		searchLower := strings.ToLower(strings.TrimSpace(search))
		var matched []*MaintenanceTicket

		for _, item := range tickets {
			if searchLower != "" {
				if !strings.Contains(strings.ToLower(item.Title), searchLower) &&
					!strings.Contains(strings.ToLower(item.AssignedTo), searchLower) &&
					!strings.Contains(strings.ToLower(item.Department), searchLower) {
					continue
				}
			}
			matched = append(matched, item)
		}

		sort.Slice(matched, func(i, j int) bool {
			return matched[i].ID < matched[j].ID
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*MaintenanceTicket{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*MaintenanceTicket, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if t, ok := tickets[id]; ok {
			return t, nil
		}
		return nil, fmt.Errorf("ticket %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(tickets)+1)
		title := fmt.Sprintf("%v", data["title"])
		if title == "" || title == "<nil>" {
			title = fmt.Sprintf("%v", data["name"])
		}

		tk := &MaintenanceTicket{
			ID:         newID,
			Title:      title,
			Priority:   fmt.Sprintf("%v", data["priority"]),
			Status:     fmt.Sprintf("%v", data["status"]),
			AssignedTo: fmt.Sprintf("%v", data["assigned_to"]),
			Department: fmt.Sprintf("%v", data["department"]),
			CreatedAt:  time.Now(),
		}
		if tk.Priority == "" || tk.Priority == "<nil>" {
			tk.Priority = "medium"
		}
		if tk.Status == "" || tk.Status == "<nil>" {
			tk.Status = "open"
		}
		if tk.AssignedTo == "" || tk.AssignedTo == "<nil>" {
			tk.AssignedTo = "Field Engineer"
		}
		if tk.Department == "" || tk.Department == "<nil>" {
			tk.Department = "Hardware Support"
		}
		tickets[newID] = tk
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		tk, ok := tickets[id]
		if !ok {
			return fmt.Errorf("ticket %s not found", id)
		}
		if v, exists := data["title"]; exists && v != nil {
			tk.Title = fmt.Sprintf("%v", v)
		}
		if v, exists := data["status"]; exists && v != nil {
			tk.Status = fmt.Sprintf("%v", v)
		}
		if v, exists := data["priority"]; exists && v != nil {
			tk.Priority = fmt.Sprintf("%v", v)
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(tickets, id)
		return nil
	})

	return res
}

// ----------------------------------------------------
// 5. Firmware Releases Resource
// ----------------------------------------------------

func buildFirmwareResource() *filament.BaseResource[*FirmwareRelease] {
	res := filament.NewResource[*FirmwareRelease]("firmware").
		ModelName("Firmware Release").
		PluralModelName("Firmware Releases").
		NavigationGroup("Operations & Engineers").
		RecordID(func(f *FirmwareRelease) string {
			if f == nil {
				return ""
			}
			return f.ID
		})

	res.Table(func(t *tables.Table[*FirmwareRelease]) *tables.Table[*FirmwareRelease] {
		return t.
			Columns(
				tables.TextColumn[*FirmwareRelease]("version").
					Label("Version").
					Weight("bold").
					Searchable(),
				tables.TextColumn[*FirmwareRelease]("hardware_target").
					Label("Target Hardware").
					Searchable(),
				tables.BadgeColumn[*FirmwareRelease]("channel").
					Label("Channel").
					Colors(map[string]string{
						"stable": "success",
						"beta":   "warning",
						"edge":   "danger",
					}),
				tables.TextColumn[*FirmwareRelease]("rollout_pct").
					Label("Rollout (%)"),
			)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("version").Label("Release Version").Required(),
			forms.TextInput("hardware_target").Label("Target Hardware Model").Required(),
			forms.Select("channel").
				Label("Release Channel").
				Options(map[string]string{
					"stable": "Stable (Production)",
					"beta":   "Beta Testing",
					"edge":   "Edge / Nightly",
				}),
			forms.TextInput("rollout_pct").Numeric().Label("Rollout Percentage (0-100)"),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*FirmwareRelease, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		searchLower := strings.ToLower(strings.TrimSpace(search))
		var matched []*FirmwareRelease

		for _, item := range firmwareReleases {
			if searchLower != "" {
				if !strings.Contains(strings.ToLower(item.Version), searchLower) &&
					!strings.Contains(strings.ToLower(item.HardwareTarget), searchLower) &&
					!strings.Contains(strings.ToLower(item.Channel), searchLower) {
					continue
				}
			}
			matched = append(matched, item)
		}

		sort.Slice(matched, func(i, j int) bool {
			return matched[i].ID < matched[j].ID
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*FirmwareRelease{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*FirmwareRelease, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if f, ok := firmwareReleases[id]; ok {
			return f, nil
		}
		return nil, fmt.Errorf("firmware release %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(firmwareReleases)+1)
		ver := fmt.Sprintf("%v", data["version"])
		if ver == "" || ver == "<nil>" {
			ver = fmt.Sprintf("%v", data["name"])
		}
		rollout, _ := strconv.Atoi(fmt.Sprintf("%v", data["rollout_pct"]))

		fw := &FirmwareRelease{
			ID:             newID,
			Version:        ver,
			HardwareTarget: fmt.Sprintf("%v", data["hardware_target"]),
			Channel:        fmt.Sprintf("%v", data["channel"]),
			RolloutPct:     rollout,
			IsActive:       true,
			ReleasedAt:     time.Now(),
		}
		if fw.Channel == "" || fw.Channel == "<nil>" {
			fw.Channel = "stable"
		}
		if fw.HardwareTarget == "" || fw.HardwareTarget == "<nil>" {
			fw.HardwareTarget = "EdgeCore-Pro"
		}
		if fw.RolloutPct <= 0 {
			fw.RolloutPct = 100
		}
		firmwareReleases[newID] = fw
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		fw, ok := firmwareReleases[id]
		if !ok {
			return fmt.Errorf("firmware release %s not found", id)
		}
		if v, exists := data["version"]; exists && v != nil {
			fw.Version = fmt.Sprintf("%v", v)
		}
		if v, exists := data["channel"]; exists && v != nil {
			fw.Channel = fmt.Sprintf("%v", v)
		}
		if v, exists := data["rollout_pct"]; exists && v != nil {
			fw.RolloutPct, _ = strconv.Atoi(fmt.Sprintf("%v", v))
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(firmwareReleases, id)
		return nil
	})

	return res
}
