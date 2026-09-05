package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lincmox/lincmox/internal/lincstation"
)

// Monitor is the interface all background monitors must implement.
type Monitor interface {
	Run(ctx context.Context, controller *LEDController)
	Name() string
}

// MonitorManager manages lifecycle of all monitors.
type MonitorManager struct {
	controller *LEDController
	mu         sync.Mutex
	running    map[string]context.CancelFunc
}

// NewMonitorManager creates a new MonitorManager.
func NewMonitorManager(controller *LEDController) *MonitorManager {
	return &MonitorManager{
		controller: controller,
		running:    make(map[string]context.CancelFunc),
	}
}

// Start starts a monitor in background. If a monitor with the same name is
// already running, it is stopped first.
func (m *MonitorManager) Start(mon Monitor) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing monitor with same name
	if cancel, ok := m.running[mon.Name()]; ok {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.running[mon.Name()] = cancel
	go mon.Run(ctx, m.controller)
}

// Stop stops a monitor by name.
func (m *MonitorManager) Stop(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.running[name]; ok {
		cancel()
		delete(m.running, name)
	}
}

// Status returns the list of currently running monitor names.
func (m *MonitorManager) Status() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.running))
	for name := range m.running {
		names = append(names, name)
	}
	return map[string]any{"running": names}
}

// StartNetwork starts the NetworkMonitor.
func (m *MonitorManager) StartNetwork(iface string, interval time.Duration, threshold uint64) {
	m.Start(&NetworkMonitor{
		iface:     iface,
		interval:  interval,
		threshold: threshold,
	})
}

// StopNetwork stops the NetworkMonitor.
func (m *MonitorManager) StopNetwork() {
	m.Stop("network")
}

// --- NetworkMonitor ---

// NetworkMonitor watches network interface statistics and blinks the Network LED
// when traffic exceeds the configured threshold.
type NetworkMonitor struct {
	iface     string
	interval  time.Duration
	threshold uint64 // bytes per interval that count as "active"
}

func (n *NetworkMonitor) Name() string { return "network" }

func (n *NetworkMonitor) Run(ctx context.Context, ctrl *LEDController) {
	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	var prevRx, prevTx uint64

	for {
		select {
		case <-ctx.Done():
			// Release the LED back to neutral state
			ctrl.SetAuto(lincstation.NetworkLED, false)
			return
		case <-ticker.C:
			rx, tx, err := readNetStats(n.iface)
			if err != nil {
				continue
			}
			var deltaRx, deltaTx uint64
			if rx >= prevRx {
				deltaRx = rx - prevRx
			}
			if tx >= prevTx {
				deltaTx = tx - prevTx
			}
			prevRx, prevTx = rx, tx

			active := (deltaRx + deltaTx) > n.threshold
			ctrl.SetAuto(lincstation.NetworkLED, active)
		}
	}
}

// readNetStats reads rx_bytes and tx_bytes for the given network interface
// from /sys/class/net/<iface>/statistics/.
func readNetStats(iface string) (rx, tx uint64, err error) {
	rx, err = readUint64File(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", iface))
	if err != nil {
		return 0, 0, err
	}
	tx, err = readUint64File(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", iface))
	if err != nil {
		return 0, 0, err
	}
	return rx, tx, nil
}

func readUint64File(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return v, nil
}

// --- DiskMonitor (stub — future implementation) ---

// DiskMonitor watches disk I/O statistics and blinks a LED when activity
// exceeds the configured threshold. Not yet implemented.
type DiskMonitor struct {
	device    string // e.g. "sda"
	led       lincstation.LED
	interval  time.Duration
	threshold uint64
}

func (d *DiskMonitor) Name() string { return "disk-" + d.device }

func (d *DiskMonitor) Run(ctx context.Context, ctrl *LEDController) {
	// TODO: read /sys/block/<device>/stat, fields 3 and 7 (sectors read/written)
	// and drive d.led accordingly.
	<-ctx.Done()
}
