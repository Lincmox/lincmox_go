package lincstation

import (
	"fmt"
	"strings"
)

// LED identifie une LED sur le LincStation.
type LED int

const (
	PowerLED   LED = iota // LED Power
	SATA1LED              // LED SATA 1
	SATA2LED              // LED SATA 2
	NetworkLED            // LED Network
	NVME1LED              // LED NVMe 1
	NVME2LED              // LED NVMe 2
	NVME3LED              // LED NVMe 3
	NVME4LED              // LED NVMe 4
)

// Color identifie une couleur de LED.
type Color int

const (
	White  Color = iota // Blanc
	Red                 // Rouge
	Orange              // Orange
)

// Animation identifie le mode d'animation du strip LED.
type Animation int

const (
	AnimOff    Animation = 0x00 // Éteint
	AnimBreath Animation = 0x01 // Respiration
	AnimLoop   Animation = 0x02 // Boucle de couleurs
)

// i2cAddress est l'adresse I2C du contrôleur LincStation.
const i2cAddress = byte(0x26)

// colorMasks contient les masques de registre pour chaque couleur d'une LED.
type colorMasks struct {
	White  byte
	Red    byte
	Orange byte
}

// ledConfig décrit la configuration de registre d'une LED.
type ledConfig struct {
	label   string
	regOn   byte
	regOff  byte
	regBlink byte
	colors  colorMasks
}

// ledRegistry associe chaque LED à sa configuration de registres.
var ledRegistry = map[LED]ledConfig{
	PowerLED: {
		label:    "Power",
		regOn:    0xA0,
		regOff:   0xB0,
		regBlink: 0x50,
		colors:   colorMasks{White: 0x01, Red: 0x02, Orange: 0x03},
	},
	SATA1LED: {
		label:    "SATA 1",
		regOn:    0xA0,
		regOff:   0xB0,
		regBlink: 0x52,
		colors:   colorMasks{White: 0x04, Red: 0x08, Orange: 0x0C},
	},
	SATA2LED: {
		label:    "SATA 2",
		regOn:    0xA0,
		regOff:   0xB0,
		regBlink: 0x54,
		colors:   colorMasks{White: 0x10, Red: 0x20, Orange: 0x30},
	},
	NetworkLED: {
		label:    "Network",
		regOn:    0xA0,
		regOff:   0xB0,
		regBlink: 0x56,
		colors:   colorMasks{White: 0x40, Red: 0x80, Orange: 0xC0},
	},
	NVME1LED: {
		label:    "NVMe 1",
		regOn:    0xA1,
		regOff:   0xB1,
		regBlink: 0x58,
		colors:   colorMasks{White: 0x01, Red: 0x02, Orange: 0x03},
	},
	NVME2LED: {
		label:    "NVMe 2",
		regOn:    0xA1,
		regOff:   0xB1,
		regBlink: 0x5A,
		colors:   colorMasks{White: 0x04, Red: 0x08, Orange: 0x0C},
	},
	NVME3LED: {
		label:    "NVMe 3",
		regOn:    0xA1,
		regOff:   0xB1,
		regBlink: 0x5C,
		colors:   colorMasks{White: 0x10, Red: 0x20, Orange: 0x30},
	},
	NVME4LED: {
		label:    "NVMe 4",
		regOn:    0xA1,
		regOff:   0xB1,
		regBlink: 0x5E,
		colors:   colorMasks{White: 0x40, Red: 0x80, Orange: 0xC0},
	},
}

// Registres du strip LED.
const (
	regStripAnimation  = byte(0x90)
	regStripBrightness = byte(0x91)
	regStripRed        = byte(0x92)
	regStripGreen      = byte(0x93)
	regStripBlue       = byte(0x94)
	regStripLoop1Red   = byte(0x95)
	regStripLoop1Green = byte(0x96)
	regStripLoop1Blue  = byte(0x97)
	regStripLoop2Red   = byte(0x98)
	regStripLoop2Green = byte(0x99)
	regStripLoop2Blue  = byte(0x9A)
)

// animationValues mappe chaque Animation à sa valeur de registre.
var animationValues = map[Animation]byte{
	AnimOff:    0x00,
	AnimBreath: 0x01,
	AnimLoop:   0x02,
}

// SATALEDByNumber retourne la LED SATA correspondant au numéro (1 ou 2).
func SATALEDByNumber(n int) (LED, error) {
	switch n {
	case 1:
		return SATA1LED, nil
	case 2:
		return SATA2LED, nil
	default:
		return 0, fmt.Errorf("%w: SATA LED %d (valid: 1-2)", ErrInvalidNumber, n)
	}
}

// NVMELEDByNumber retourne la LED NVMe correspondant au numéro (1 à 4).
func NVMELEDByNumber(n int) (LED, error) {
	switch n {
	case 1:
		return NVME1LED, nil
	case 2:
		return NVME2LED, nil
	case 3:
		return NVME3LED, nil
	case 4:
		return NVME4LED, nil
	default:
		return 0, fmt.Errorf("%w: NVMe LED %d (valid: 1-4)", ErrInvalidNumber, n)
	}
}

// ColorFromString convertit une chaîne en Color.
// Valeurs acceptées : "white", "red", "orange" (insensible à la casse).
func ColorFromString(s string) (Color, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "white":
		return White, nil
	case "red":
		return Red, nil
	case "orange":
		return Orange, nil
	default:
		return 0, fmt.Errorf("%w: %q (valid: white, red, orange)", ErrInvalidColor, s)
	}
}

// AnimationFromString convertit une chaîne en Animation.
// Valeurs acceptées : "off", "breath", "loop" (insensible à la casse).
func AnimationFromString(s string) (Animation, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off":
		return AnimOff, nil
	case "breath":
		return AnimBreath, nil
	case "loop":
		return AnimLoop, nil
	default:
		return 0, fmt.Errorf("%w: %q (valid: off, breath, loop)", ErrInvalidAnimation, s)
	}
}

// String retourne le label lisible de la LED.
func (l LED) String() string {
	if cfg, ok := ledRegistry[l]; ok {
		return cfg.label
	}
	return fmt.Sprintf("LED(%d)", int(l))
}

// String retourne le nom de la couleur.
func (c Color) String() string {
	switch c {
	case White:
		return "white"
	case Red:
		return "red"
	case Orange:
		return "orange"
	default:
		return fmt.Sprintf("Color(%d)", int(c))
	}
}

// String retourne le nom de l'animation.
func (a Animation) String() string {
	switch a {
	case AnimOff:
		return "off"
	case AnimBreath:
		return "breath"
	case AnimLoop:
		return "loop"
	default:
		return fmt.Sprintf("Animation(%d)", int(a))
	}
}
