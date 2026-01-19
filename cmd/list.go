package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/ui"
	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
		Run: func(_ *cobra.Command, args []string) {
			if err := listSnapshots(); err != nil {
				fmt.Println(err)
			}
		},
	}
)

func listSnapshots() error {
	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		snapshots, err := manager.ListSnapshots()
		if err != nil {
			return fmt.Errorf("error listing snapshots: %v", err)
		}

		if len(snapshots) == 0 {
			fmt.Println("No snapshots found.")
			return nil
		}

		for _, snapshot := range snapshots {
			ageStr := ui.FormatAge(snapshot.Age)
			if snapshot.Age == 0 {
				fmt.Println(snapshot.Name)
			} else {
				fmt.Printf("%s (%s)\n", snapshot.Name, ageStr)
			}
		}

		return nil
	})
}
