package main

import (
	"fmt"
	"sync"
)

type LEDState struct {
	Color string `json:"color"`
	State string `json:"state"` // "on", "off", "blink"
}

type StripState struct {
	Color      string `json:"color"`
	Anim       string `json:"anim"`
	Brightness int    `json:"brightness"`
	Loop1      string `json:"loop1"`
	Loop2      string `json:"loop2"`
}

type DeviceState struct {
	sync.RWMutex
	LEDs  map[string]*LEDState `json:"leds"`
	Strip *StripState          `json:"strip"`
}

func NewDeviceState() *DeviceState {
	return &DeviceState{
		LEDs: map[string]*LEDState{
			"power":   {Color: "white", State: "off"},
			"sata1":   {Color: "white", State: "off"},
			"sata2":   {Color: "white", State: "off"},
			"nvme1":   {Color: "white", State: "off"},
			"nvme2":   {Color: "white", State: "off"},
			"nvme3":   {Color: "white", State: "off"},
			"nvme4":   {Color: "white", State: "off"},
			"network": {Color: "white", State: "off"},
		},
		Strip: &StripState{
			Color:      "#ff0000",
			Anim:       "off",
			Brightness: 128,
			Loop1:      "#00ff00",
			Loop2:      "#0000ff",
		},
	}
}

func (s *DeviceState) UpdateLED(name, color, state string) {
	s.Lock()
	defer s.Unlock()
	if l, ok := s.LEDs[name]; ok {
		if color != "" {
			l.Color = color
		}
		if state != "" {
			l.State = state
		}
	}
}

func (s *DeviceState) UpdateStripAnim(anim string) {
	s.Lock()
	defer s.Unlock()
	s.Strip.Anim = anim
}

func (s *DeviceState) UpdateStripBrightness(b int) {
	s.Lock()
	defer s.Unlock()
	s.Strip.Brightness = b
}

func (s *DeviceState) UpdateStripColor(r, g, b int) {
	s.Lock()
	defer s.Unlock()
	s.Strip.Color = fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func (s *DeviceState) UpdateStripLoop(loop int, r, g, b int) {
	s.Lock()
	defer s.Unlock()
	colorHex := fmt.Sprintf("#%02x%02x%02x", r, g, b)
	if loop == 1 {
		s.Strip.Loop1 = colorHex
	} else if loop == 2 {
		s.Strip.Loop2 = colorHex
	}
}
