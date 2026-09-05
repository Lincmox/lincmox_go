package lincstation

import (
	"fmt"
	"sync"
)

// i2cBus is the interface implemented by real and mock I2C backends.
type i2cBus interface {
	ReadByte(reg byte) (byte, error)
	WriteByte(reg, value byte) error
	Close() error
}

// Option is an alias for DeviceOption for backward compatibility.
type Option = DeviceOption

// DeviceOption is a functional option for configuring Device.
type DeviceOption func(*DeviceOptions)

type DeviceOptions struct {
	simulate bool
	busID    int
	verbose  bool
}

// WithSimulation enables mock I2C backend (no hardware required).
func WithSimulation() DeviceOption {
	return func(o *DeviceOptions) { o.simulate = true }
}

// WithSimulate enables mock I2C backend (no hardware required) with explicit bool.
func WithSimulate(simulate bool) DeviceOption {
	return func(o *DeviceOptions) { o.simulate = simulate }
}

// WithBusID forces a specific I2C bus number (auto-detected by default).
func WithBusID(id int) DeviceOption {
	return func(o *DeviceOptions) { o.busID = id }
}

// WithVerbose enables verbose I2C logging.
func WithVerbose() DeviceOption {
	return func(o *DeviceOptions) { o.verbose = true }
}

// Device is the LincStation hardware controller.
type Device struct {
	bus   i2cBus
	mu    sync.Mutex
	verbose bool
}

// NewDevice creates a new Device with the given options.
func NewDevice(opts ...DeviceOption) (*Device, error) {
	var options DeviceOptions
	for _, opt := range opts {
		opt(&options)
	}

	var bus i2cBus
	var err error

	if options.simulate {
		bus = newMockBus(options.verbose)
	} else {
		bus, err = newSMBusDevice(options.busID, options.verbose)
		if err != nil {
			return nil, err
		}
	}

	return &Device{
		bus:     bus,
		verbose: options.verbose,
	}, nil
}

// SetLED turns a specific LED on or off with the given color.
func (d *Device) SetLED(led LED, on bool, color Color) error {
	cfg, ok := ledRegistry[led]
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidLED, led)
	}

	colorMask, ok := colorMaskForColor(cfg, color)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidColor, color)
	}

	reg := cfg.regOn
	if !on {
		reg = cfg.regOff
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		action := "on"
		if !on {
			action = "off"
		}
		fmt.Printf("[I2C] WRITE 0x%02X ← 0x%02X (LED %s %s %s)\n", reg, colorMask, cfg.label, action, color.String())
	}

	return d.bus.WriteByte(reg, colorMask)
}

// BlinkLED enables or disables blinking for a specific LED.
func (d *Device) BlinkLED(led LED, blink bool) error {
	cfg, ok := ledRegistry[led]
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidLED, led)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		action := "enable"
		if !blink {
			action = "disable"
		}
		fmt.Printf("[I2C] WRITE 0x%02X ← 0x%02X (LED %s blink %s)\n", cfg.regBlink, byteFromBool(blink), cfg.label, action)
	}

	return d.bus.WriteByte(cfg.regBlink, byteFromBool(blink))
}

// SetStripAnimation sets the LED strip animation mode.
func (d *Device) SetStripAnimation(anim Animation) error {
	val, ok := animationValues[anim]
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidAnimation, anim)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		fmt.Printf("[I2C] WRITE 0x%02X ← 0x%02X (Strip animation: %s)\n", regStripAnimation, val, anim.String())
	}

	return d.bus.WriteByte(regStripAnimation, val)
}

// SetStripBrightness sets the LED strip brightness (0-255).
func (d *Device) SetStripBrightness(brightness byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		fmt.Printf("[I2C] WRITE 0x%02X ← 0x%02X (Strip brightness: %d)\n", regStripBrightness, brightness, brightness)
	}

	return d.bus.WriteByte(regStripBrightness, brightness)
}

// SetStripColor sets the LED strip solid color.
func (d *Device) SetStripColor(r, g, b byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		fmt.Printf("[I2C] WRITE 0x%02X ← 0x%02X, 0x%02X ← 0x%02X, 0x%02X ← 0x%02X (Strip color: RGB(%d,%d,%d))\n",
			regStripRed, r, regStripGreen, g, regStripBlue, b, r, g, b)
	}

	if err := d.bus.WriteByte(regStripRed, r); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripGreen, g); err != nil {
		return err
	}
	return d.bus.WriteByte(regStripBlue, b)
}

// SetStripLoopColor sets the color for a specific loop (1 or 2).
func (d *Device) SetStripLoopColor(loop int, r, g, b byte) error {
	var regR, regG, regB byte
	switch loop {
	case 1:
		regR, regG, regB = regStripLoop1Red, regStripLoop1Green, regStripLoop1Blue
	case 2:
		regR, regG, regB = regStripLoop2Red, regStripLoop2Green, regStripLoop2Blue
	default:
		return fmt.Errorf("%w: %d", ErrInvalidLoop, loop)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		fmt.Printf("[I2C] WRITE 0x%02X ← 0x%02X, 0x%02X ← 0x%02X, 0x%02X ← 0x%02X (Strip loop %d: RGB(%d,%d,%d))\n",
			regR, r, regG, g, regB, b, loop, r, g, b)
	}

	if err := d.bus.WriteByte(regR, r); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regG, g); err != nil {
		return err
	}
	return d.bus.WriteByte(regB, b)
}

// ResetLEDs resets all individual LEDs to off.
func (d *Device) ResetLEDs() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cfg := range ledRegistry {
		if d.verbose {
			fmt.Printf("[I2C] WRITE 0x%02X ← 0x00 (LED %s off)\n", cfg.regOff, cfg.label)
		}
		if err := d.bus.WriteByte(cfg.regOff, 0x00); err != nil {
			return err
		}
		if d.verbose {
			fmt.Printf("[I2C] WRITE 0x%02X ← 0x00 (LED %s blink off)\n", cfg.regBlink, cfg.label)
		}
		if err := d.bus.WriteByte(cfg.regBlink, 0x00); err != nil {
			return err
		}
	}
	return nil
}

// ResetStrip resets the LED strip to off.
func (d *Device) ResetStrip() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.verbose {
		fmt.Println("[I2C] Resetting strip (animation=off, brightness=0, color=black)")
	}
	if err := d.bus.WriteByte(regStripAnimation, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripBrightness, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripRed, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripGreen, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripBlue, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripLoop1Red, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripLoop1Green, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripLoop1Blue, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripLoop2Red, 0x00); err != nil {
		return err
	}
	if err := d.bus.WriteByte(regStripLoop2Green, 0x00); err != nil {
		return err
	}
	return d.bus.WriteByte(regStripLoop2Blue, 0x00)
}

// Reset resets both LEDs and strip.
func (d *Device) Reset() error {
	if err := d.ResetLEDs(); err != nil {
		return err
	}
	return d.ResetStrip()
}

// Close closes the I2C bus connection.
func (d *Device) Close() error {
	return d.bus.Close()
}

// String returns a string representation of the device state.
func (d *Device) String() string {
	return "LincStation Device (I2C 0x26)"
}

// --- Helpers ---

func colorMaskForColor(cfg ledConfig, color Color) (byte, bool) {
	switch color {
	case White:
		return cfg.colors.White, true
	case Red:
		return cfg.colors.Red, true
	case Orange:
		return cfg.colors.Orange, true
	default:
		return 0, false
	}
}

func byteFromBool(b bool) byte {
	if b {
		return 0x01
	}
	return 0x00
}