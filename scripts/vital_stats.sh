#!/bin/bash
# run commands to get vital statistics for each of the parameters of the device
aqpstatus=$(echo -n $(systemctl is-active aquapone.service))
cfgstatus=$(echo -n $(systemctl is-active cfgwatch.service))
is_online=$(echo -n $(curl -Is https://www.google.com | awk 'NR==1 {print $1" "$2; exit}'))
usage=$(echo -n $(vmstat | awk '{print $12" "$13}' | tail -1))
# uptime=$(uptime | awk '{print $3" "$4" "$5}'| rev | cut -c 2- | rev | tr -d '\n')
upsince=$(echo -n $(uptime -s))

echo -n "$aqpstatus,$cfgstatus,$is_online,$usage,$upsince" 