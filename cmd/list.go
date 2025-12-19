package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
		Run: func(_ *cobra.Command, args []string) {
			listSnapshots()
		},
	}
)

func listSnapshots() {
	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first.")
		return
	}

	config, err := internal.ReadConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	snapshotManager, err := internal.SnapshotManager(config)
	if err != nil {
		fmt.Printf("Error initializing snapshot manager: %v\n", err)
		return
	}
	defer snapshotManager.Close()

	snapshots, err := snapshotManager.ListSnapshots()
	if err != nil {
		fmt.Printf("Error listing snapshots: %v\n", err)
		return
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found.")
		return
	}

	for _, snapshot := range snapshots {
		fmt.Println(snapshot)
	}
}
