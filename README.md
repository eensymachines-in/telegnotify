# telegnotify
----

Is a `systemctl unit` running on the device that'd collect device, log, working details and _post it to an API_ on the cloud. 

- We then have periodic notifications of the most recent device & service status running on the device. 
  - Status of specific services - configured for the device
  - Status of GPIO pins - configured on a particular device
  - Logs for the device - again from a particular file on the device
  
- No longer do we have to remote login using the ssh to diagnose / check if the device is doing what its intended to do, or browse thru logs to trace the events. Either this service sends it a database or an API can route relevant messages to Telegram. 
  
- Re-usable /  configurable service such that it can be installed on any device and in context of any program run on the device. _We though are developing this in the context of the patio program_

```
-----telegnotify ----notifications---- webapi ---- telegram
            |
            |
|-----------|-------------|
vitals  gpio status     logs 
|           |             |
----------------------------
            |
    shell scripts
```

Under the hood the program runs shell scripts that can extract real time device information periodically. Shell scripts are better since it has a clear separation of concerns when it comes to editing the program, or extending the functionality as we move ahead.

### Push to the device using GitHub actions:
-----

When it comes to updates, a mechanism that can push new code as it is committed to the repo to be pushed onto remote devices. 

### systemctl unit:
---

A systemctl unit that has no dependency with other services will help to keep the program running. We dont need any cron jobs (though it seems like it needs one), the Go program can have coroutines that a sense of periodics. 

### Referring to `/etc/aquapone.reg.json` or devicereg API?
----

Device registration details - not yet decided if we want to refer such a device registration from the cloud, or from the device on the ground.
While the device on the ground is more susceptible to attacks and breaches, referring to the API may not always be the fastest way to get the device details. Although referring to the API would mean that if the device isnt registered atall it would deny running the device in the first place, Having a source of truth on the server makes more sense for the device. Though getting the MacID still is better from the hardware underneath.

In a way we are also decoupling the service from the context of `patio program`. Now the service can be used with any of the other porograms, it'd then mean the serivce is a generic and not only in the context of the `patio program`

1. Device name 
2. Telegram group that the notifications needed to be posted on 

### Checking for vitals, service status :
----
It's beneficial to periodically check the vital statistics of the chip. For remote devices, receiving regular diagnostic updates can reassure the owners.

Key metrics to report for remotely operating devices include:

CPU uptime
Systemctl service status
CPU load
Online status

To achieve this, shell scripts will be executed at regular intervals to gather these metrics, and a GoLang loop will send the collected data to the notification API. This approach ensures a clear separation of concerns.

### Checking for GPIO digital status:
------

For designated configurable pins, you can monitor the digital status of the GPIO and report it to the API, which will then generate a notification. This process can be executed periodically and serves as a general notification within the context of aquaponics.

This notification is specifically for digital pins and not for analog pins, as it is typically the actuators that require verification to ensure they are operating as expected.
