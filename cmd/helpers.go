package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/erikgeiser/promptkit/selection"
	"github.com/leonvogt/lunar/internal"
)

// Handles the common pattern of checking config, creating a manager,
// and ensuring cleanup. It calls the provided function with the manager and config.
func withSnapshotManager(operation func(manager *internal.Manager, config *internal.Config) error) error {
	if !internal.DoesConfigExist() {
		return fmt.Errorf("there seems to be no configuration file. Please run 'lunar init' first")
	}

	config, err := internal.ReadConfig()
	if err != nil {
		return fmt.Errorf("error reading config: %v", err)
	}

	snapshotManager, err := internal.NewSnapshotManager(config)
	if err != nil {
		return fmt.Errorf("error initializing snapshot manager: %v", err)
	}
	defer snapshotManager.Close()

	return operation(snapshotManager, config)
}

func selectSnapshot(manager *internal.Manager, promptMessage string) (string, error) {
	snapshots, err := manager.ListSnapshots()
	if err != nil {
		return "", fmt.Errorf("error listing snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		return "", fmt.Errorf("no snapshots found")
	}

	snapshotNames := make([]string, len(snapshots))
	for i, snapshot := range snapshots {
		snapshotNames[i] = snapshot.Name
	}

	prompt := selection.New(promptMessage, snapshotNames)
	prompt.PageSize = 50

	selectedSnapshot, err := prompt.RunPrompt()
	if err != nil {
		return "", err
	}

	return selectedSnapshot, nil
}

// Returns the snapshot name from args if provided,
// otherwise prompts the user to select one.
func getSnapshotNameFromArgsOrPrompt(args []string, manager *internal.Manager, promptMessage string) (string, error) {
	if len(args) >= 1 {
		snapshotName := args[0]
		if err := manager.CheckIfSnapshotExists(snapshotName); err != nil {
			return "", err
		}
		return snapshotName, nil
	}

	return selectSnapshot(manager, promptMessage)
}

// spawnBackgroundCommand starts a background process with the given arguments.
// The process runs independently and survives after the parent exits.
func spawnBackgroundCommand(args ...string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find executable: %v", err)
	}

	command := exec.Command(executable, args...)
	command.Stdout = nil
	command.Stderr = nil
	command.Stdin = nil

	if err := command.Start(); err != nil {
		return fmt.Errorf("could not start background process: %v", err)
	}

	return nil
}
