package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/leonvogt/lunar/internal"
)

// withSnapshotManager handles the common pattern of checking config, creating a manager,
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

// printWaitingStatus prints appropriate waiting messages based on operation status
func printWaitingStatus(status internal.OperationStatus) {
	if status.WaitingForOperation {
		fmt.Println("Waiting for ongoing operation to complete...")
	}
	if status.WaitingForSnapshot {
		fmt.Println("Waiting for ongoing snapshot to complete...")
	}
}
