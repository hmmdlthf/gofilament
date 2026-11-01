package main

import (
	"fmt"
	"sync"
	"time"
)

// Device represents an IoT connected hardware device in the fleet.
type Device struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SerialNo     string    `json:"serial_no"`
	Model        string    `json:"model"`
	OS           string    `json:"os"`
	BatteryLevel int       `json:"battery_level"`
	Status       string    `json:"status"` // online, offline, maintenance
	Location     string    `json:"location"`
	Firmware     string    `json:"firmware"`
	LatencyMs    int       `json:"latency_ms"`
	LastPing     time.Time `json:"last_ping"`
	CreatedAt    time.Time `json:"created_at"`
}

// FleetUser represents an operations engineer or operator.
type FleetUser struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Role       string    `json:"role"` // admin, operator, technician
	Department string    `json:"department"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// TelemetryLog represents a real-time sensor measurement from fleet hardware.
type TelemetryLog struct {
	ID         string    `json:"id"`
	DeviceName string    `json:"device_name"`
	Metric     string    `json:"metric"`   // Temperature, CPU Load, Battery Voltage, Signal RSSI
	Value      string    `json:"value"`    // "42.8 °C", "18%", "12.4 V", "-62 dBm"
	Severity   string    `json:"severity"` // normal, warning, critical
	RecordedAt time.Time `json:"recorded_at"`
}

// MaintenanceTicket represents a work order or hardware service request.
type MaintenanceTicket struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	DeviceID   string    `json:"device_id"`
	Priority   string    `json:"priority"` // critical, high, medium, low
	Status     string    `json:"status"`   // open, in_progress, resolved, closed
	AssignedTo string    `json:"assigned_to"`
	Department string    `json:"department"`
	CreatedAt  time.Time `json:"created_at"`
}

// FirmwareRelease represents an OTA firmware binary distributed to the fleet.
type FirmwareRelease struct {
	ID             string    `json:"id"`
	Version        string    `json:"version"` // e.g. "v3.2.0"
	HardwareTarget string    `json:"hardware_target"`
	Channel        string    `json:"channel"` // stable, beta, edge
	RolloutPct     int       `json:"rollout_pct"`
	IsActive       bool      `json:"is_active"`
	ReleasedAt     time.Time `json:"released_at"`
}

var (
	dataMu           sync.RWMutex
	devices          = make(map[string]*Device)
	fleetUsers       = make(map[string]*FleetUser)
	telemetryLogs    = make(map[string]*TelemetryLog)
	tickets          = make(map[string]*MaintenanceTicket)
	firmwareReleases = make(map[string]*FirmwareRelease)
)

func seedDatabase() {
	dataMu.Lock()
	defer dataMu.Unlock()

	devices = make(map[string]*Device)
	fleetUsers = make(map[string]*FleetUser)
	telemetryLogs = make(map[string]*TelemetryLog)
	tickets = make(map[string]*MaintenanceTicket)
	firmwareReleases = make(map[string]*FirmwareRelease)

	now := time.Now()

	devList := []*Device{
		{
			ID:           "1",
			Name:         "Gateway Alpha-101",
			SerialNo:     "GW-9921-USA",
			Model:        "PulseEdge v4",
			OS:           "Linux Embedded 6.1",
			BatteryLevel: 98,
			Status:       "online",
			Location:     "Austin Tech Hub, TX",
			Firmware:     "v2.4.1",
			LatencyMs:    18,
			LastPing:     now.Add(-2 * time.Minute),
			CreatedAt:    now.Add(-30 * 24 * time.Hour),
		},
		{
			ID:           "2",
			Name:         "Drone Rover X-2",
			SerialNo:     "ROV-4412-CA",
			Model:        "AeroPulse 300",
			OS:           "RTOS Pulse 2.0",
			BatteryLevel: 42,
			Status:       "online",
			Location:     "Fremont Facility, CA",
			Firmware:     "v2.3.9",
			LatencyMs:    32,
			LastPing:     now.Add(-5 * time.Minute),
			CreatedAt:    now.Add(-20 * 24 * time.Hour),
		},
		{
			ID:           "3",
			Name:         "Telemetry Sensor Node 09",
			SerialNo:     "SN-0092-DE",
			Model:        "MicroPulse I2C",
			OS:           "Zephyr RTOS",
			BatteryLevel: 14,
			Status:       "maintenance",
			Location:     "Munich Logistics Hub",
			Firmware:     "v1.9.0",
			LatencyMs:    85,
			LastPing:     now.Add(-45 * time.Minute),
			CreatedAt:    now.Add(-60 * 24 * time.Hour),
		},
		{
			ID:           "4",
			Name:         "Cargo Tracker Pod #88",
			SerialNo:     "TRK-8810-SG",
			Model:        "PulseTrack GPS",
			OS:           "FreeRTOS 10.4",
			BatteryLevel: 0,
			Status:       "offline",
			Location:     "Singapore Port Pier 4",
			Firmware:     "v2.1.0",
			LatencyMs:    0,
			LastPing:     now.Add(-26 * time.Hour),
			CreatedAt:    now.Add(-90 * 24 * time.Hour),
		},
		{
			ID:           "5",
			Name:         "Smart Gateway Central",
			SerialNo:     "GW-1100-UK",
			Model:        "PulseEdge v4 Pro",
			OS:           "Linux Embedded 6.1",
			BatteryLevel: 100,
			Status:       "online",
			Location:     "London Data Center, UK",
			Firmware:     "v2.4.2",
			LatencyMs:    12,
			LastPing:     now.Add(-1 * time.Minute),
			CreatedAt:    now.Add(-10 * 24 * time.Hour),
		},
	}

	for _, d := range devList {
		devices[d.ID] = d
	}

	userList := []*FleetUser{
		{
			ID:         "1",
			Name:       "Sarah Connor",
			Email:      "sarah.c@pulsefleet.io",
			Role:       "admin",
			Department: "Operations & Safety",
			IsActive:   true,
			CreatedAt:  now.Add(-60 * 24 * time.Hour),
		},
		{
			ID:         "2",
			Name:       "Marcus Vance",
			Email:      "marcus.v@pulsefleet.io",
			Role:       "technician",
			Department: "Hardware Engineering",
			IsActive:   true,
			CreatedAt:  now.Add(-40 * 24 * time.Hour),
		},
		{
			ID:         "3",
			Name:       "Elena Rostova",
			Email:      "elena.r@pulsefleet.io",
			Role:       "operator",
			Department: "Flight Control & Tracking",
			IsActive:   true,
			CreatedAt:  now.Add(-25 * 24 * time.Hour),
		},
	}

	for _, u := range userList {
		fleetUsers[u.ID] = u
	}

	telemList := []*TelemetryLog{
		{ID: "1", DeviceName: "Gateway Alpha-101", Metric: "Core Temperature", Value: "42.8 °C", Severity: "normal", RecordedAt: now.Add(-2 * time.Minute)},
		{ID: "2", DeviceName: "Gateway Alpha-101", Metric: "CPU Load", Value: "18%", Severity: "normal", RecordedAt: now.Add(-3 * time.Minute)},
		{ID: "3", DeviceName: "Drone Rover X-2", Metric: "Battery Voltage", Value: "11.2 V", Severity: "warning", RecordedAt: now.Add(-5 * time.Minute)},
		{ID: "4", DeviceName: "Telemetry Sensor Node 09", Metric: "Signal RSSI", Value: "-89 dBm", Severity: "critical", RecordedAt: now.Add(-45 * time.Minute)},
		{ID: "5", DeviceName: "Smart Gateway Central", Metric: "Throughput", Value: "1.4 Gbps", Severity: "normal", RecordedAt: now.Add(-1 * time.Minute)},
	}
	for _, t := range telemList {
		telemetryLogs[t.ID] = t
	}

	ticketList := []*MaintenanceTicket{
		{ID: "1", Title: "Replace solar backup cell on Node 09", DeviceID: "3", Priority: "critical", Status: "in_progress", AssignedTo: "Marcus Vance", Department: "Hardware Engineering", CreatedAt: now.Add(-2 * 24 * time.Hour)},
		{ID: "2", Title: "Antenna realignment at Tower 7", DeviceID: "1", Priority: "high", Status: "open", AssignedTo: "Marcus Vance", Department: "Hardware Engineering", CreatedAt: now.Add(-1 * 24 * time.Hour)},
		{ID: "3", Title: "Recover offline cargo pod GPS beacon", DeviceID: "4", Priority: "critical", Status: "open", AssignedTo: "Sarah Connor", Department: "Operations & Safety", CreatedAt: now.Add(-12 * time.Hour)},
		{ID: "4", Title: "Routine thermal audit & firmware check", DeviceID: "5", Priority: "low", Status: "resolved", AssignedTo: "Elena Rostova", Department: "Flight Control & Tracking", CreatedAt: now.Add(-5 * 24 * time.Hour)},
	}
	for _, tk := range ticketList {
		tickets[tk.ID] = tk
	}

	fwList := []*FirmwareRelease{
		{ID: "1", Version: "v3.2.0-rc1", HardwareTarget: "PulseEdge v4 Pro", Channel: "beta", RolloutPct: 25, IsActive: true, ReleasedAt: now.Add(-3 * 24 * time.Hour)},
		{ID: "2", Version: "v2.4.2-LTS", HardwareTarget: "PulseEdge v4", Channel: "stable", RolloutPct: 100, IsActive: true, ReleasedAt: now.Add(-14 * 24 * time.Hour)},
		{ID: "3", Version: "v2.3.9", HardwareTarget: "AeroPulse 300", Channel: "stable", RolloutPct: 80, IsActive: true, ReleasedAt: now.Add(-30 * 24 * time.Hour)},
		{ID: "4", Version: "v1.9.0", HardwareTarget: "MicroPulse I2C", Channel: "edge", RolloutPct: 10, IsActive: false, ReleasedAt: now.Add(-60 * 24 * time.Hour)},
	}
	for _, f := range fwList {
		firmwareReleases[f.ID] = f
	}

	fmt.Printf("📡 Pulse Fleet seed data loaded: %d devices, %d logs, %d tickets, %d releases, %d operators.\n",
		len(devices), len(telemetryLogs), len(tickets), len(firmwareReleases), len(fleetUsers))
}
