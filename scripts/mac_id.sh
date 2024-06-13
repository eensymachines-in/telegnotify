#!/bin/bash
macid=$(cat /sys/class/net/wlan0/address)
echo -n "$macid" # this -n is important since echo adds extra '\n' to the output by default 