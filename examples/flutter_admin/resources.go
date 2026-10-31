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
