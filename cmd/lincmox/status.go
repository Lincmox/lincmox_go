package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display current device status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if isAPIMode(cmd) {
				return statusAPIMode(cmd)
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			fmt.Println(dev.String())
			return nil
		},
	}
}

func statusAPIMode(cmd *cobra.Command) error {
	client := apiClientFromCtx(cmd)
	resp, err := client.Get("http://lincmoxd/api/v1/status")
	if err != nil {
		return fmt.Errorf("GET /api/v1/status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	// Pretty-print JSON if possible, otherwise print raw.
	var pretty interface{}
	if err := json.Unmarshal(body, &pretty); err == nil {
		formatted, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			fmt.Println(string(formatted))
			return nil
		}
	}
	// Fallback: raw response.
	fmt.Println(string(body))
	return nil
}
