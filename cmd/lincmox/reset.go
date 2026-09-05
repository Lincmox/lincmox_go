package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <full|leds|strip>",
		Short: "Reset the device or specific subsystems",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := args[0]

			if isAPIMode(cmd) {
				return doAPIRequest(
					apiClientFromCtx(cmd),
					"POST",
					"http://lincmoxd/api/v1/reset",
					map[string]string{"mode": mode},
				)
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}

			switch mode {
			case "full":
				return dev.Reset()
			case "leds":
				return dev.ResetLEDs()
			case "strip":
				return dev.ResetStrip()
			default:
				return fmt.Errorf("unknown reset mode %q (valid: full, leds, strip)", mode)
			}
		},
	}
}
