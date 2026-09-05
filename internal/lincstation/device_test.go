package lincstation

import (
	"testing"
)

func TestSetLED(t *testing.T) {
	device, err := NewDevice(WithSimulation(), WithVerbose())
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	if err := device.SetLED(PowerLED, true, White); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if err := device.SetLED(PowerLED, false, Red); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if err := device.SetLED(SATA1LED, true, Orange); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBlinkLED(t *testing.T) {
	device, err := NewDevice(WithSimulation())
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	if err := device.BlinkLED(PowerLED, true); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSetStripAnimation(t *testing.T) {
	device, err := NewDevice(WithSimulation())
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	if err := device.SetStripAnimation(AnimBreath); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestReset(t *testing.T) {
	device, err := NewDevice(WithSimulation())
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	if err := device.Reset(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
