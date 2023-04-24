package daemon

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/ubuntu/adsys/internal/policies/mount"
)

func (a *App) installMount() {
	cmd := &cobra.Command{
		Use:    "mount MOUNTS_FILE",
		Short:  "Mount the locations listed in the specified file for the current user",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE:   func(cmd *cobra.Command, args []string) error { return runMounts(args[0]) },
	}
	a.rootCmd.AddCommand(cmd)
}

func runMounts(filepath string) error {
	if err := mount.RunMountForCurrentUser(context.Background(), filepath); err != nil {
		return err
	}
	return nil
}
