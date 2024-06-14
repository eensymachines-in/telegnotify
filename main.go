package main

/*
We encountered challenges in accessing the remote device for health and running parameters monitoring. Although retrieving the device's IP address is a potential solution, exposing the IP address of an IoT device poses a security risk. In the event of a breach, this exposure could compromise actuators, making the system vulnerable. To mitigate this risk, we need to schedule the device to periodically send status updates using a systemctl service. These updates can be sent to a server that logs the information to a database or routes it to a Telegram bot for notifications.

Golang app that fires a bunch of native ash scripts to get the data and then posts it to the api server.
Scripts make it a better separation of concern and more editable in the future
*/
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/eensymachines-in/patio/interrupt"
	"github.com/eensymachines-in/utilities"
	"github.com/eensymachines-in/webpi-telegnotify/models"
	log "github.com/sirupsen/logrus"
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
		log.Errorf("Unexpected number of vital stats %v", values)
		return nil, fmt.Errorf("Unexpeted count of stats VitalStatsOutput: %d", len(values))
	}
	// TODO: device name to be extracted from the reg file
	return models.Notification("Rpi0wdev test device @ Tejaura", DeviceMac, time.Now(), models.VitalStats(values[0], values[1], values[2], values[3], values[4])), nil
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
	once      sync.Once
)

func init() {
	/* Getting the mac address of the device once for the entire program duration
	Other device details as name can also be extracted here */
	// TODO: when getting the mac id of the device, first the program shall refer to the reg file
	// If not found then the fallback method is to execute shell to get mac id
	// this shall also check to see if the device with the same macid is already registered,
	// if such device is not registered then the service shall stop since sending device notifications for unregistered devices is not recommended.

	cmd := exec.Command("/bin/bash", "./scripts/mac_id.sh")

	// Create buffers to capture the standard output and standard error
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to get mac address of the device: %s", err)
	}
	DeviceMac = out.String()
	log.WithFields(log.Fields{
		"mac address": DeviceMac,
	}).Debug("device mac id read")

	// TODO: need to check withh devicereg service to know if the device is registered .
	// incase the device ISNT registered this service shall exit prematurely
	// No point in hhaving a iunregistered device send any notifications 
	// from devicereg service we would get  
	// - confirmation of the device registration 
	// - device name 
	checkEnvVars := []string{
		"TELEGNOTIFY_BASEURL",
		"CHECK_INTERVAL",
	}
	var missingEnvVar bool
	for _, v := range checkEnvVars {
		if v == "" {
			once.Do(func() {
				missingEnvVar = true
			})
			log.WithFields(log.Fields{
				"name": v,
			}).Error("Missing environment var")
		}
	}
	if missingEnvVar {
		panic("One or more environment variables is missing, aborting")
	} else {
		log.Info("All environment variables in place..")
	}
}

// upsendNotification : route notificaiton to a north bound api -  which then knows how to deal with the notification
// not		: device notification that needs to send, error incase notification could not be send
func upsendNotification(not models.DeviceNotifcn) error {
	/* Url and the client */
	cl := &http.Client{Timeout: 5 * time.Second}
	// url := "http://aqua.eensymachines.in:30004/api/devices/b8:27:eb:43:59:f8/notifications?typ=vitals"
	url := fmt.Sprintf("%s/%s/notifications?typ=vitals", os.Getenv("TELEGNOTIFY_BASEURL"), DeviceMac)

	/* request payload */
	byt, err := json.Marshal(not)
	if err != nil {
		return fmt.Errorf("failed to marshal notification %s", err)
	}
	buff := bytes.NewBuffer(byt)

	/* request ready */
	req, err := http.NewRequest("POST", url, buff)
	if err != nil {
		return fmt.Errorf("failed to form request %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	/* Send request */

	resp, err := cl.Do(req)
	if err != nil {
		// perhaps the device is not online
		// server is not reachable
		return fmt.Errorf("server not reachable %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected server response: %d", resp.StatusCode)
	}
	return nil
}

func main() {
	// Path to the shell script
	log.Info("Now starting program")
	defer log.Warn("Now exiting the service")
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
			log.WithFields(log.Fields{
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
					// NOTE: ToMessageTxt - returns msg and the error alongside, so you would get to see <nil> being printed alongside a message
					fmt.Println(not.ToMessageTxt())
					// Make arrangements to send the message to the api endpoint
					if err := upsendNotification(not); err != nil {
						// Case when the notificaiton could not be sent to the api
						// There could be several reasons for this
						log.WithFields(log.Fields{
							"err": err,
						}).Error("failed to upsend the notification..")
						continue
					}
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
						log.Errorf("failed to run script %s: %s", script.Path, err)
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
