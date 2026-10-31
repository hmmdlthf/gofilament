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

var (
	dataMu     sync.RWMutex
	devices    = make(map[string]*Device)
	fleetUsers = make(map[string]*FleetUser)
)

func seedDatabase() {
	dataMu.Lock()
	defer dataMu.Unlock()

	devices = make(map[string]*Device)
	fleetUsers = make(map[string]*FleetUser)

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

	fmt.Printf("📡 Pulse Fleet seed data loaded: %d devices, %d fleet operators.\n", len(devices), len(fleetUsers))
}
