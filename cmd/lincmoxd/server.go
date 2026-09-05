package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/lincmox/lincmox/internal/lincstation"
)

// Server is the lincmoxd HTTP server serving the REST API and the embedded Web UI.
type Server struct {
	device     *lincstation.Device
	controller *LEDController
	mux        *http.ServeMux
	monitors   *MonitorManager
}

// NewServer creates a new Server with the given Device.
func NewServer(device *lincstation.Device) *Server {
	controller := NewLEDController(device)
	s := &Server{
		device:     device,
		controller: controller,
		mux:        http.NewServeMux(),
		monitors:   NewMonitorManager(controller),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// LED routes
	s.mux.HandleFunc("/api/v1/led/power/on", s.handleLEDAction("power", "on"))
	s.mux.HandleFunc("/api/v1/led/power/off", s.handleLEDAction("power", "off"))
	s.mux.HandleFunc("/api/v1/led/power/blink", s.handleLEDBlink("power"))
	s.mux.HandleFunc("/api/v1/led/sata/1/on", s.handleLEDAction("sata1", "on"))
	s.mux.HandleFunc("/api/v1/led/sata/1/off", s.handleLEDAction("sata1", "off"))
	s.mux.HandleFunc("/api/v1/led/sata/1/blink", s.handleLEDBlink("sata1"))
	s.mux.HandleFunc("/api/v1/led/sata/2/on", s.handleLEDAction("sata2", "on"))
	s.mux.HandleFunc("/api/v1/led/sata/2/off", s.handleLEDAction("sata2", "off"))
	s.mux.HandleFunc("/api/v1/led/sata/2/blink", s.handleLEDBlink("sata2"))
	s.mux.HandleFunc("/api/v1/led/nvme/1/on", s.handleLEDAction("nvme1", "on"))
	s.mux.HandleFunc("/api/v1/led/nvme/1/off", s.handleLEDAction("nvme1", "off"))
	s.mux.HandleFunc("/api/v1/led/nvme/1/blink", s.handleLEDBlink("nvme1"))
	s.mux.HandleFunc("/api/v1/led/nvme/2/on", s.handleLEDAction("nvme2", "on"))
	s.mux.HandleFunc("/api/v1/led/nvme/2/off", s.handleLEDAction("nvme2", "off"))
	s.mux.HandleFunc("/api/v1/led/nvme/2/blink", s.handleLEDBlink("nvme2"))
	s.mux.HandleFunc("/api/v1/led/nvme/3/on", s.handleLEDAction("nvme3", "on"))
	s.mux.HandleFunc("/api/v1/led/nvme/3/off", s.handleLEDAction("nvme3", "off"))
	s.mux.HandleFunc("/api/v1/led/nvme/3/blink", s.handleLEDBlink("nvme3"))
	s.mux.HandleFunc("/api/v1/led/nvme/4/on", s.handleLEDAction("nvme4", "on"))
	s.mux.HandleFunc("/api/v1/led/nvme/4/off", s.handleLEDAction("nvme4", "off"))
	s.mux.HandleFunc("/api/v1/led/nvme/4/blink", s.handleLEDBlink("nvme4"))
	s.mux.HandleFunc("/api/v1/led/network/on", s.handleLEDAction("network", "on"))
	s.mux.HandleFunc("/api/v1/led/network/off", s.handleLEDAction("network", "off"))
	s.mux.HandleFunc("/api/v1/led/network/blink", s.handleLEDBlink("network"))

	// Strip routes
	s.mux.HandleFunc("/api/v1/strip/animation", s.handleStripAnimation)
	s.mux.HandleFunc("/api/v1/strip/brightness", s.handleStripBrightness)
	s.mux.HandleFunc("/api/v1/strip/color", s.handleStripColor)
	s.mux.HandleFunc("/api/v1/strip/loop/1/color", s.handleStripLoopColor(1))
	s.mux.HandleFunc("/api/v1/strip/loop/2/color", s.handleStripLoopColor(2))

	// Reset
	s.mux.HandleFunc("/api/v1/reset", s.handleReset)

	// Status
	s.mux.HandleFunc("/api/v1/status", s.handleStatus)

	// Monitors
	s.mux.HandleFunc("/api/v1/monitors", s.handleMonitors)
	s.mux.HandleFunc("/api/v1/monitors/network/enable", s.handleNetworkMonitorEnable)
	s.mux.HandleFunc("/api/v1/monitors/network/disable", s.handleNetworkMonitorDisable)

	// Web UI
	s.mux.Handle("/", http.FileServer(webUI()))
}

// ListenAndServe starts the server on both a Unix socket and a TCP address.
func (s *Server) ListenAndServe(ctx context.Context, sockPath, tcpAddr string) error {
	// Remove stale socket
	os.Remove(sockPath)

	unixLn, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("unix socket: %w", err)
	}
	// Allow group access so CLI can connect
	os.Chmod(sockPath, 0660)

	tcpLn, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		unixLn.Close()
		return fmt.Errorf("tcp: %w", err)
	}

	httpServer := &http.Server{Handler: s.mux}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		httpServer.Serve(unixLn)
	}()
	go func() {
		defer wg.Done()
		httpServer.Serve(tcpLn)
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(shutCtx)
	unixLn.Close()
	tcpLn.Close()
	wg.Wait()
	return nil
}

// --- Helpers ---

func (s *Server) ledFromName(name string) (lincstation.LED, bool) {
	m := map[string]lincstation.LED{
		"power":   lincstation.PowerLED,
		"sata1":   lincstation.SATA1LED,
		"sata2":   lincstation.SATA2LED,
		"nvme1":   lincstation.NVME1LED,
		"nvme2":   lincstation.NVME2LED,
		"nvme3":   lincstation.NVME3LED,
		"nvme4":   lincstation.NVME4LED,
		"network": lincstation.NetworkLED,
	}
	led, ok := m[name]
	return led, ok
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- LED Handlers ---

type ledOnOffRequest struct {
	Color string `json:"color"`
}

func (s *Server) handleLEDAction(ledName, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		led, ok := s.ledFromName(ledName)
		if !ok {
			writeError(w, http.StatusBadRequest, "unknown LED: "+ledName)
			return
		}
		var req ledOnOffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		color, err := lincstation.ColorFromString(req.Color)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		on := action == "on"
		const manualTTL = 30 * time.Second
		if err := s.controller.SetManual(led, on, color, manualTTL); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

type ledBlinkRequest struct {
	Blink bool `json:"blink"`
}

func (s *Server) handleLEDBlink(ledName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		led, ok := s.ledFromName(ledName)
		if !ok {
			writeError(w, http.StatusBadRequest, "unknown LED: "+ledName)
			return
		}
		var req ledBlinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := s.device.BlinkLED(led, req.Blink); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// --- Strip Handlers ---

type animationRequest struct {
	Mode string `json:"mode"`
}

func (s *Server) handleStripAnimation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req animationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	anim, err := lincstation.AnimationFromString(req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.device.SetStripAnimation(anim); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type brightnessRequest struct {
	Value int `json:"value"`
}

func (s *Server) handleStripBrightness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req brightnessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Value < 0 || req.Value > 255 {
		writeError(w, http.StatusBadRequest, "brightness must be 0-255")
		return
	}
	if err := s.device.SetStripBrightness(byte(req.Value)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type colorRequest struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

func (s *Server) handleStripColor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req colorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := s.device.SetStripColor(byte(req.R), byte(req.G), byte(req.B)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStripLoopColor(loop int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req colorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := s.device.SetStripLoopColor(loop, byte(req.R), byte(req.G), byte(req.B)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// --- Reset Handler ---

type resetRequest struct {
	Mode string `json:"mode"`
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	var err error
	switch req.Mode {
	case "full":
		err = s.device.Reset()
	case "leds":
		err = s.device.ResetLEDs()
	case "strip":
		err = s.device.ResetStrip()
	default:
		writeError(w, http.StatusBadRequest, "mode must be full, leds or strip")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Status Handler ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"device":   s.device.String(),
		"monitors": s.monitors.Status(),
	})
}

// --- Monitor Handlers ---

func (s *Server) handleMonitors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	writeJSON(w, http.StatusOK, s.monitors.Status())
}

type networkMonitorRequest struct {
	Iface       string `json:"iface"`
	IntervalMs  int    `json:"interval_ms"`
	ThresholdBs uint64 `json:"threshold_bytes"`
}

func (s *Server) handleNetworkMonitorEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req networkMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Iface == "" {
		req.Iface = "eth0"
	}
	if req.IntervalMs <= 0 {
		req.IntervalMs = 500
	}
	if req.ThresholdBs == 0 {
		req.ThresholdBs = 1024
	}
	s.monitors.StartNetwork(req.Iface, time.Duration(req.IntervalMs)*time.Millisecond, req.ThresholdBs)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleNetworkMonitorDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	s.monitors.StopNetwork()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
