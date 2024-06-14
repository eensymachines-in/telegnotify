export TELEGNOTIFY_BASEURL=http://aqua.eensymachines.in:30004/api/devices
export DEVICEREG_BASEURL=http://aqua.eensymachines.in:30001/api/devices
export CHECK_INTERVAL=600 #check and send notificaitons in how many seconds
build:
	go mod tidy
	go build -o /usr/local/bin/telegnotify
	# also we can have some systemctl unit built and set 

run: 
	go run .
