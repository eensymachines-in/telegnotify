package main

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/eensymachines-in/utilities"
	log "github.com/sirupsen/logrus"
)

func init() {
	close := utilities.SetUpLog()
	defer close()
}

// runShellScript pre written shell scripts on sh files can be run
func runShellScript(scriptPath string) (string, error) {
	// Create the command to run the shell script
	cmd := exec.Command("/bin/bash", scriptPath)

	// Create a buffer to capture the standard output
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// Return the output as a string
	return out.String(), nil
}

// runCommand Command with arguments run to get the output
func runCommand(cmd string, args ...string) (string, error) {
	// Create the command with its arguments
	command := exec.Command(cmd, args...)

	// Create a buffer to capture the standard output
	var out bytes.Buffer
	command.Stdout = &out

	// Run the command
	err := command.Run()
	if err != nil {
		return "", err
	}

	// Return the output as a string
	return out.String(), nil
}

func main() {

	log.Info("Starting service for telegram notifications")
	defer log.Warn("now closing service for telegram notifications")

	output, err := runCommand("echo", "-n", "Hello, world!")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Echo output:", output)

	runShellScript("./scripts/service_status.sh")

}
