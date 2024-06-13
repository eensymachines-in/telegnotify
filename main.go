package main

/*
We encountered challenges in accessing the remote device for health and running parameters monitoring. Although retrieving the device's IP address is a potential solution, exposing the IP address of an IoT device poses a security risk. In the event of a breach, this exposure could compromise actuators, making the system vulnerable. To mitigate this risk, we need to schedule the device to periodically send status updates using a systemctl service. These updates can be sent to a server that logs the information to a database or routes it to a Telegram bot for notifications.

Golang app that fires a bunch of native ash scripts to get the data and then posts it to the api server.
Scripts make it a better separation of concern and more editable in the future
*/
import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/eensymachines-in/patio/interrupt"
	"github.com/eensymachines-in/utilities"
	"github.com/eensymachines-in/webpi-telegnotify/models"
	"github.com/sirupsen/logrus"
)

type ScriptOutput interface {
	ToNotification() (models.DeviceNotifcn, error)
}

/* implementations for ScriptOutput */
type VitalStatsOutput string

func (vso VitalStatsOutput) ToNotification() (models.DeviceNotifcn, error) {
	if vso == "" {
		return nil, fmt.Errorf("Empty VitalStatsOutput, cannot convert to Notification")
	}
	values := strings.Split(string(vso), ",")
	if len(values) < 5 {
		// unexpected number of vital stats check again
		logrus.Errorf("Unexpected number of vital stats %v", values)
		return nil, fmt.Errorf("Unexpeted count of stats VitalStatsOutput: %d", len(values))
	}
	return models.Notification("Rpi0wdev, test device", DeviceMac, time.Now(), models.VitalStats(values[0], values[1], values[2], values[3], values[4])), nil
}

type ShellScript struct {
	Path     string
	Args     []string
	ToOutput func(string) ScriptOutput
}

func (ss *ShellScript) Run() (ScriptOutput, error) {
	// Create the command to run the shell script with arguments
	cmd := exec.Command("/bin/bash", append([]string{ss.Path}, ss.Args...)...)

	// Create buffers to capture the standard output and standard error
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		return ss.ToOutput(""), fmt.Errorf("cmd.Run() failed with %s: %s", err, stderr.String())
	}

	// Return the output as a specialized string
	return ss.ToOutput(out.String()), nil
}

var (
	NewShellScript = func(path string, args []string, scrpop func(string) ScriptOutput) *ShellScript {
		return &ShellScript{
			Path:     path,
			Args:     args,
			ToOutput: scrpop,
		}
	}
	NewVitalStatsOutput = func(op string) ScriptOutput {
		return VitalStatsOutput(op)
	}
	DeviceMac string
)

func init() {
	/* Getting the mac address of the device once for the entire program duration
	Other device details as name can also be extracted here */
	cmd := exec.Command("/bin/bash", "./scripts/mac_id.sh")

	// Create buffers to capture the standard output and standard error
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		logrus.Fatalf("failed to get mac address of the device: %s", err)
	}
	DeviceMac = out.String()
	logrus.WithFields(logrus.Fields{
		"mac address": DeviceMac,
	}).Debug("device mac id read")
}

func main() {
	// Path to the shell script
	logrus.Info("Now starting program")
	defer logrus.Warn("Now exiting the service")
	cleanup := utilities.SetUpLog()
	defer cleanup() // incase its a file output log, this shall close the same
	/* Setting up scripts to run, each script is a task that sends output to a single channel that can be sent to the api server */
	Scripts := []*ShellScript{
		NewShellScript("./scripts/vital_stats.sh", []string{}, NewVitalStatsOutput),
	}
	notfcnChn := make(chan models.DeviceNotifcn, 10) // single channel over which all task send their script output periodically

	// Main thread context
	ctx, cancel := context.WithCancel(context.Background()) // use this context in all the task loops
	var wg sync.WaitGroup

	/* Signal watcher task - this spins out first since we want the system interruptions to get the fist priority when exiting */
	wg.Add(1)
	go func() {
		/* System signal watcher group - when receives the interrupt signal will cancel the context here in main */
		defer wg.Done()
		for intr := range interrupt.SysSignalWatch(ctx, &wg) {
			logrus.WithFields(logrus.Fields{
				"time": intr.Format(time.RFC822),
			}).Warn("Interrupted...")
			cancel()
		}
	}()

	/* Starting the reading tasks before we start the writing ones */
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		for {
			select {
			case not, ok := <-notfcnChn:
				if ok {
					fmt.Println(not.ToMessageTxt())
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	/* Spin out each tasks for the scripts, these are the worker threads that spin out
	they usually write on the notification channels */
	for _, scrp := range Scripts {
		wg.Add(1)
		go func(script *ShellScript, notify chan models.DeviceNotifcn, ctx context.Context, wg *sync.WaitGroup) {
			defer wg.Done()
			for {
				select {
				case <-time.After(10 * time.Second):
					output, err := script.Run()
					if err != nil {
						logrus.Errorf("failed to run script %s: %s", script.Path, err)
						return
					}
					not, _ := output.ToNotification()
					notify <- not
				case <-ctx.Done():
					return
				}
			}
		}(scrp, notfcnChn, ctx, &wg)
	}
	wg.Wait()
}
