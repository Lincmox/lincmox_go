package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lincmox/lincmox/internal/lincstation"
	"github.com/spf13/cobra"
)

const (
	defaultSockPath = "/run/lincmoxd/lincmoxd.sock"
	defaultTCPAddr  = ":8080"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		sockPath string
		tcpAddr  string
		simulate bool
		verbose  bool
		busID    int
	)

	root := &cobra.Command{
		Use:   "lincmoxd",
		Short: "LincMox daemon — REST API and Web UI for LincStation LED control",
		Long: `lincmoxd is the LincMox background daemon.

It exposes a REST API and an embedded Web UI to control LincStation LEDs.
When lincmoxd is running, the lincmox CLI automatically routes its commands
through the daemon's Unix socket instead of accessing the I2C bus directly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(sockPath, tcpAddr, simulate, verbose, busID)
		},
	}

	root.Flags().StringVar(&sockPath, "sock", defaultSockPath, "Unix socket path")
	root.Flags().StringVar(&tcpAddr, "addr", defaultTCPAddr, "TCP address to listen on")
	root.Flags().BoolVar(&simulate, "simulate", false, "Use mock I2C backend (no hardware required)")
	root.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose I2C logging")
	root.Flags().IntVar(&busID, "bus", -1, "force the I2C bus number to use (default: auto-detect)")

	return root
}

func run(sockPath, tcpAddr string, simulate, verbose bool, busID int) error {
	// Build device options
	var opts []lincstation.Option
	if simulate || os.Getenv("LINCMOX_ENV") == "dev" {
		opts = append(opts, lincstation.WithSimulation())
	}
	if verbose {
		opts = append(opts, lincstation.WithVerbose())
	}
	if busID >= 0 {
		opts = append(opts, lincstation.WithBusID(busID))
	}

	device, err := lincstation.NewDevice(opts...)
	if err != nil {
		return fmt.Errorf("open I2C device: %w", err)
	}
	defer device.Close()

	server := NewServer(device)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("lincmoxd listening on %s (unix) and %s (tcp)\n", sockPath, tcpAddr)
	return server.ListenAndServe(ctx, sockPath, tcpAddr)
}
