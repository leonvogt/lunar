package cmd

import (
	"fmt"
	"time"

	"github.com/leonvogt/lunar/internal"
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
			ageStr := formatAge(snapshot.Age)
			if snapshot.Age == 0 {
				fmt.Println(snapshot.Name)
			} else {
				fmt.Printf("%s (%s)\n", snapshot.Name, ageStr)
			}
		}

		return nil
	})
}

func formatAge(age time.Duration) string {
	if age == 0 {
		return "unknown"
	}

	days := int(age.Hours() / 24)
	hours := int(age.Hours()) % 24
	minutes := int(age.Minutes()) % 60

	if days > 0 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	if hours > 0 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	if minutes > 0 {
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}

	return "just now"
}
