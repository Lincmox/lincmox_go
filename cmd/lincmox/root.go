package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/lincmox/lincmox/internal/lincstation"
	"github.com/spf13/cobra"
)

const socketPath = "/run/lincmoxd/lincmoxd.sock"

// Context keys — unexported to avoid collisions.
type deviceKey struct{}
type apiClientKey struct{}

func newRootCmd() *cobra.Command {
	var simulate bool
	var verbose bool
	var busID int

	cmd := &cobra.Command{
		Use:   "lincmox",
		Short: "LincMox hardware controller CLI",
		Long: `
██╗     ██╗███╗   ██╗ ██████╗███╗   ███╗ ██████╗ ██╗  ██╗
██║     ██║████╗  ██║██╔════╝████╗ ████║██╔═══██╗╚██╗██╔╝
██║     ██║██╔██╗ ██║██║     ██╔████╔██║██║   ██║ ╚███╔╝ 
██║     ██║██║╚██╗██║██║     ██║╚██╔╝██║██║   ██║ ██╔██╗ 
███████╗██║██║ ╚████║╚██████╗██║ ╚═╝ ██║╚██████╔╝██╔╝ ██╗
╚══════╝╚═╝╚═╝  ╚═══╝ ╚═════╝╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═╝

LincMox hardware controller — controls LEDs and LED strip.

When /run/lincmoxd/lincmoxd.sock is present the CLI acts as an HTTP client
talking to the lincmoxd daemon over the Unix socket.
Otherwise it drives the hardware directly.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			if _, err := os.Stat(socketPath); err == nil {
				// Socket present → API client mode.
				transport := &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						return (&net.Dialer{
							Timeout: 5 * time.Second,
						}).DialContext(ctx, "unix", socketPath)
					},
				}
				client := &http.Client{
					Transport: transport,
					Timeout:   30 * time.Second,
				}
				ctx = context.WithValue(ctx, apiClientKey{}, client)
				cmd.SetContext(ctx)
				return nil
			}

			// No socket → direct device mode.
			var opts []lincstation.DeviceOption
			if simulate {
				opts = append(opts, lincstation.WithSimulate(true))
			}
			if verbose {
				opts = append(opts, lincstation.WithVerbose())
			}
			if busID >= 0 {
				opts = append(opts, lincstation.WithBusID(busID))
			}
			dev, err := lincstation.NewDevice(opts...)
			if err != nil {
				return fmt.Errorf("failed to open device: %w", err)
			}
			ctx = context.WithValue(ctx, deviceKey{}, dev)
			cmd.SetContext(ctx)
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&simulate, "simulate", false, "simulate hardware commands (no real I/O)")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose I2C logging")
	cmd.PersistentFlags().IntVar(&busID, "bus", -1, "force the I2C bus number to use (default: auto-detect)")

	cmd.AddCommand(
		newLEDCmd(),
		newStripCmd(),
		newResetCmd(),
		newStatusCmd(),
	)

	return cmd
}

// deviceFromCtx retrieves the *lincstation.Device stored in the command context.
func deviceFromCtx(cmd *cobra.Command) *lincstation.Device {
	v := cmd.Context().Value(deviceKey{})
	if v == nil {
		return nil
	}
	return v.(*lincstation.Device)
}

// apiClientFromCtx retrieves the *http.Client stored in the command context.
func apiClientFromCtx(cmd *cobra.Command) *http.Client {
	v := cmd.Context().Value(apiClientKey{})
	if v == nil {
		return nil
	}
	return v.(*http.Client)
}

// isAPIMode returns true when the CLI is running in HTTP-over-Unix-socket mode.
func isAPIMode(cmd *cobra.Command) bool {
	return apiClientFromCtx(cmd) != nil
}
