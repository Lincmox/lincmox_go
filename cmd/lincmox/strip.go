package main

import (
	"fmt"
	"strconv"

	"github.com/lincmox/lincmox/internal/lincstation"
	"github.com/spf13/cobra"
)

func newStripCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "strip",
		Short: "Control the LED strip",
	}
	cmd.AddCommand(
		newStripAnimationCmd(),
		newStripBrightnessCmd(),
		newStripColorCmd(),
		newStripLoopCmd(),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// strip animation <off|breath|loop>
// ---------------------------------------------------------------------------

func newStripAnimationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "animation <off|breath|loop>",
		Short: "Set the strip animation mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			animStr := args[0]
			anim, err := lincstation.AnimationFromString(animStr)
			if err != nil {
				return err
			}

			if isAPIMode(cmd) {
				return doAPIRequest(
					apiClientFromCtx(cmd),
					"POST",
					"http://lincmoxd/api/v1/strip/animation",
					map[string]string{"animation": animStr},
				)
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			return dev.SetStripAnimation(anim)
		},
	}
}

// ---------------------------------------------------------------------------
// strip brightness <0-255>
// ---------------------------------------------------------------------------

func newStripBrightnessCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "brightness <0-255>",
		Short: "Set the strip brightness",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := strconv.ParseUint(args[0], 10, 8)
			if err != nil {
				return fmt.Errorf("invalid brightness %q: must be 0-255", args[0])
			}
			brightness := uint8(val)

			if isAPIMode(cmd) {
				return doAPIRequest(
					apiClientFromCtx(cmd),
					"POST",
					"http://lincmoxd/api/v1/strip/brightness",
					map[string]uint8{"brightness": brightness},
				)
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			return dev.SetStripBrightness(brightness)
		},
	}
}

// ---------------------------------------------------------------------------
// strip color <r> <g> <b>
// ---------------------------------------------------------------------------

func newStripColorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "color <r> <g> <b>",
		Short: "Set the strip solid color (each channel 0-255)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, g, b, err := parseRGB(args)
			if err != nil {
				return err
			}

			if isAPIMode(cmd) {
				return doAPIRequest(
					apiClientFromCtx(cmd),
					"POST",
					"http://lincmoxd/api/v1/strip/color",
					map[string]uint8{"r": r, "g": g, "b": b},
				)
			}

			dev := deviceFromCtx(cmd)
			if dev == nil {
				return fmt.Errorf("no device available")
			}
			return dev.SetStripColor(r, g, b)
		},
	}
}

// ---------------------------------------------------------------------------
// strip loop <1|2> color <r> <g> <b>
// ---------------------------------------------------------------------------

func newStripLoopCmd() *cobra.Command {
	loopCmd := &cobra.Command{
		Use:   "loop",
		Short: "Control individual loop colors",
	}

	for _, n := range []int{1, 2} {
		n := n
		sub := &cobra.Command{
			Use:   strconv.Itoa(n),
			Short: fmt.Sprintf("Control loop %d", n),
		}
		colorSub := &cobra.Command{
			Use:   "color <r> <g> <b>",
			Short: fmt.Sprintf("Set loop %d color (each channel 0-255)", n),
			Args:  cobra.ExactArgs(3),
			RunE: func(cmd *cobra.Command, args []string) error {
				r, g, b, err := parseRGB(args)
				if err != nil {
					return err
				}

				if isAPIMode(cmd) {
					return doAPIRequest(
						apiClientFromCtx(cmd),
						"POST",
						fmt.Sprintf("http://lincmoxd/api/v1/strip/loop/%d/color", n),
						map[string]uint8{"r": r, "g": g, "b": b},
					)
				}

				dev := deviceFromCtx(cmd)
				if dev == nil {
					return fmt.Errorf("no device available")
				}
				return dev.SetStripLoopColor(n, r, g, b)
			},
		}
		sub.AddCommand(colorSub)
		loopCmd.AddCommand(sub)
	}
	return loopCmd
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseRGB parses three string arguments as r, g, b uint8 values.
func parseRGB(args []string) (r, g, b uint8, err error) {
	parse := func(s, name string) (uint8, error) {
		v, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return 0, fmt.Errorf("invalid %s value %q: must be 0-255", name, s)
		}
		return uint8(v), nil
	}
	if r, err = parse(args[0], "r"); err != nil {
		return
	}
	if g, err = parse(args[1], "g"); err != nil {
		return
	}
	b, err = parse(args[2], "b")
	return
}
