package main

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/eensymachines-in/utilities"
	"github.com/sirupsen/logrus"
)

func init() {

}

func runShellScript(scriptPath string, args ...string) (string, error) {
	// Create the command to run the shell script with arguments
	cmd := exec.Command("/bin/bash", append([]string{scriptPath}, args...)...)

	// Create buffers to capture the standard output and standard error
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("cmd.Run() failed with %s: %s", err, stderr.String())
	}

	// Return the output as a string
	return out.String(), nil
}

func main() {
	// Path to the shell script
	logrus.Info("Now starting program")
	close := utilities.SetUpLog()
	defer close() // incase its a file output log, this shall close the same

	scriptPath := "./scripts/service_status.sh"

	// Arguments to pass to the shell script
	args := []string{"aquapone.service"}

	// Run the shell script with arguments
	output, err := runShellScript(scriptPath, args...)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print the output from the shell script
	// mind the linefeed in the  print statement itself
	fmt.Printf("Shell script output:%s\n", output)
}
