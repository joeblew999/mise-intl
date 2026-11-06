package main

import (
	"fmt"
	"os"
	"os/exec"
)

// verify is go tool intl verify: what the repo's GitHub workflow runs once it has checked out, the repo's own setup
// and test tasks when it has them (with mise), then check. Run here, it does what GitHub does.
func verify(args []string) {
	if _, err := exec.LookPath("mise"); err == nil {
		runTask("setup")
		runTask("test")
	} else {
		fmt.Println("· no mise here: the repo's own tasks are left to it (go test ./...)")
	}
	check(nil)
}

// runTask runs the repo's own task when it has one.
func runTask(task string) {
	if exec.Command("mise", "tasks", "info", task).Run() != nil {
		return
	}
	fmt.Println("$ mise run " + task)
	cmd := exec.Command("mise", "run", task)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fail(fmt.Errorf("mise run %s: %w", task, err))
	}
}
