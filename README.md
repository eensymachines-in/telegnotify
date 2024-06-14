# telegnotify
----

Is a `systemctl unit` running on the device that'd collect device, log, working details and post it to an API on the cloud. 

- We then have periodic notifications of the most recent device & service status running on the device. 
  - Status of specific services - configured for the device
  - Status of GPIO pins - configured on a particular device
  - Logs for the device - again from a particular file on the device
  
- No longer do we have to remote login using the ssh to diagnose / check if the device is doing what its intended to do, or browse thru logs to trace the events. Either this service sends it a database or an API can route relevant messages to Telegram. 
  
- Re-usable /  configurable service such that it can be installed on any device and in context of any program run on the device. _We though are developing this in the context of the patio program_

```
    telegnotify ----notifications---- webapi ---- telegram
    |
    |
|---------|-------------|
vitals  gpio status   logs 
```