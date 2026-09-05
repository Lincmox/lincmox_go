package lincstation

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// mockBus est un backend I2C simulé, utilisé pour les tests et le mode dev.
// Il persiste les valeurs de registres dans un fichier JSON.
type mockBus struct {
	mu        sync.Mutex
	registers map[byte]byte
	path      string
	verbose   bool
}

// newMockBus crée un mockBus et charge l'état depuis le fichier JSON si existant.
func newMockBus(verbose bool) *mockBus {
	m := &mockBus{
		registers: make(map[byte]byte),
		path:      "/tmp/lincstation_mock_registers.json",
		verbose:   verbose,
	}
	m._load()
	return m
}

// _load charge les registres depuis le fichier JSON.
// Les clés sont des chaînes décimales représentant les adresses de registre.
func (m *mockBus) _load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		// Pas de fichier : état vierge, pas d'erreur.
		return
	}

	var raw map[string]uint8
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	for k, v := range raw {
		var reg byte
		if _, err := fmt.Sscanf(k, "%d", &reg); err == nil {
			m.registers[reg] = v
		}
	}
}

// _save sauvegarde atomiquement les registres dans le fichier JSON.
func (m *mockBus) _save() {
	out := make(map[string]string, len(m.registers))
	for k, v := range m.registers {
		out[fmt.Sprintf("%d", k)] = fmt.Sprintf("0x%02X", v)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return
	}

	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, m.path)
}

// ReadByte lit la valeur du registre reg (ou 0x00 si non défini).
func (m *mockBus) ReadByte(reg byte) (byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val := m.registers[reg] // zero value = 0x00 si absent
	if m.verbose {
		fmt.Printf("[MOCK] READ  0x%02X → 0x%02X\n", reg, val)
	}
	return val, nil
}

// WriteByte écrit value dans le registre reg.
func (m *mockBus) WriteByte(reg, value byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	value &= 0xFF
	m.registers[reg] = value
	if m.verbose {
		fmt.Printf("[MOCK] WRITE 0x%02X ← 0x%02X\n", reg, value)
	}
	m._save()
	return nil
}

// Close est un no-op pour le mockBus.
func (m *mockBus) Close() error {
	return nil
}
