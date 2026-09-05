package main

import (
	"sync"
	"time"

	"github.com/lincmox/lincmox/internal/lincstation"
)

// LEDController manages access to LEDs with priority between automatic monitors
// and manual commands (from API or CLI).
//
// Manual commands win for a configurable TTL. Once the TTL expires, monitors
// regain control automatically.
type LEDController struct {
	device *lincstation.Device
	mu     sync.Mutex
	states map[lincstation.LED]ledState
}

type ledState struct {
	// manualUntil is the time until which manual override is active.
	// Zero value means the LED is in automatic mode.
	manualUntil time.Time
}

// NewLEDController creates a new LEDController wrapping the given device.
func NewLEDController(device *lincstation.Device) *LEDController {
	return &LEDController{
		device: device,
		states: make(map[lincstation.LED]ledState),
	}
}

// SetManual overrides a LED manually for the given TTL duration.
// After TTL expires, monitors can resume control via SetAuto.
func (c *LEDController) SetManual(led lincstation.LED, on bool, color lincstation.Color, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.states[led] = ledState{manualUntil: time.Now().Add(ttl)}
	return c.device.SetLED(led, on, color)
}

// SetManualBlink overrides LED blink state manually for the given TTL.
func (c *LEDController) SetManualBlink(led lincstation.LED, blink bool, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.states[led] = ledState{manualUntil: time.Now().Add(ttl)}
	return c.device.BlinkLED(led, blink)
}

// SetAuto is called by monitors. It writes to the device only if no manual
// override is currently active for this LED.
func (c *LEDController) SetAuto(led lincstation.LED, blink bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state, ok := c.states[led]; ok && time.Now().Before(state.manualUntil) {
		return nil // manual override still active, skip
	}
	return c.device.BlinkLED(led, blink)
}

// IsManual reports whether the LED is currently under manual override.
func (c *LEDController) IsManual(led lincstation.LED) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.states[led]
	return ok && time.Now().Before(state.manualUntil)
}
