package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/lincmox/lincmox/internal/lincstation"
	"github.com/spf13/cobra"
)

func newLEDCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "led",
		Short: "Control individual LEDs",
	}
	cmd.AddCommand(
		newLEDPowerCmd(),
		newLEDSATACmd(),
		newLEDNVMeCmd(),
		newLEDNetworkCmd(),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// led power
// ---------------------------------------------------------------------------

func newLEDPowerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "power",
		Short: "Control the power LED",
	}
	cmd.AddCommand(
		newLEDOnCmd("power", lincstation.PowerLED, "/api/v1/led/power"),
		newLEDOffCmd("power", lincstation.PowerLED, "/api/v1/led/power"),
		newLEDBlinkCmd("power", lincstation.PowerLED, "/api/v1/led/power"),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// led sata <1|2>
// ---------------------------------------------------------------------------

func newLEDSATACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sata <1|2>",
		Short: "Control a SATA drive LED",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("specify a sub-command: on, off, blink")
		},
	}

	for _, n := range []int{1, 2} {
		n := n
		apiBase := fmt.Sprintf("/api/v1/led/sata/%d", n)
		led, _ := lincstation.SATALEDByNumber(n)

		sub := &cobra.Command{
			Use:   strconv.Itoa(n),
			Short: fmt.Sprintf("Control SATA %d LED", n),
		}
		sub.AddCommand(
			newLEDOnCmd(fmt.Sprintf("sata%d", n), led, apiBase),
			newLEDOffCmd(fmt.Sprintf("sata%d", n), led, apiBase),
			newLEDBlinkCmd(fmt.Sprintf("sata%d", n), led, apiBase),
		)
		cmd.AddCommand(sub)
	}
	return cmd
}

// ---------------------------------------------------------------------------
// led nvme <1|2|3|4>
// ---------------------------------------------------------------------------

func newLEDNVMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nvme <1|2|3|4>",
		Short: "Control an NVMe drive LED",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("specify a sub-command: on, off, blink")
		},
	}

	for _, n := range []int{1, 2, 3, 4} {
		n := n
		apiBase := fmt.Sprintf("/api/v1/led/nvme/%d", n)
		led, _ := lincstation.NVMELEDByNumber(n)

		sub := &cobra.Command{
			Use:   strconv.Itoa(n),
			Short: fmt.Sprintf("Control NVMe %d LED", n),
		}
		sub.AddCommand(
			newLEDOnCmd(fmt.Sprintf("nvme%d", n), led, apiBase),
			newLEDOffCmd(fmt.Sprintf("nvme%d", n), led, apiBase),
			newLEDBlinkCmd(fmt.Sprintf("nvme%d", n), led, apiBase),
		)
		cmd.AddCommand(sub)
	}
	return cmd
}

// ---------------------------------------------------------------------------
// led network
// ---------------------------------------------------------------------------

func newLEDNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Control the network LED",
	}
	cmd.AddCommand(
		newLEDOnCmd("network", lincstation.NetworkLED, "/api/v1/led/network"),
		newLEDOffCmd("network", lincstation.NetworkLED, "/api/v1/led/network"),
		newLEDBlinkCmd("network", lincstation.NetworkLED, "/api/v1/led/network"),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// Generic on/off/blink command builders
// ---------------------------------------------------------------------------

func newLEDOnCmd(_ string, led lincstation.LED, apiBase string) *cobra.Command {
	return &cobra.Command{
		Use:   "on <color>",
		Short: "Turn LED on",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			colorStr := args[0]
			color, err := lincstation.ColorFromString(colorStr)
			if err != nil {
				return err
			}

			if isAPIMode(cmd) {
				return ledAPIRequest(cmd, "POST", apiBase+"/on", map[string]string{"color": colorStr})
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			return dev.SetLED(led, true, color)
		},
	}
}

func newLEDOffCmd(_ string, led lincstation.LED, apiBase string) *cobra.Command {
	return &cobra.Command{
		Use:   "off <color>",
		Short: "Turn LED off",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			colorStr := args[0]
			color, err := lincstation.ColorFromString(colorStr)
			if err != nil {
				return err
			}

			if isAPIMode(cmd) {
				return ledAPIRequest(cmd, "POST", apiBase+"/off", map[string]string{"color": colorStr})
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			return dev.SetLED(led, false, color)
		},
	}
}

func newLEDBlinkCmd(_ string, led lincstation.LED, apiBase string) *cobra.Command {
	return &cobra.Command{
		Use:   "blink <on|off>",
		Short: "Set LED blink state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blinkStr := args[0]
			var blink bool
			switch blinkStr {
			case "on":
				blink = true
			case "off":
				blink = false
			default:
				return fmt.Errorf("invalid blink value %q (valid: on, off)", blinkStr)
			}

			if isAPIMode(cmd) {
				return ledAPIRequest(cmd, "POST", apiBase+"/blink", map[string]string{"blink": blinkStr})
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			return dev.BlinkLED(led, blink)
		},
	}
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func ledAPIRequest(cmd *cobra.Command, method, path string, body interface{}) error {
	client := apiClientFromCtx(cmd)
	return doAPIRequest(client, method, "http://lincmoxd"+path, body)
}

func doAPIRequest(client *http.Client, method, url string, body interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}
