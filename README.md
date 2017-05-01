# Raspberry Pi web GPIO controller [![Build Status](https://api.travis-ci.org/krasi-georgiev/rpi-web-control.svg?branch=master)](https://travis-ci.org/krasi-georgiev/rpi-web-control)
*a very minimalistic app to control the RPi outputs using a web browser*
  * garage or front door
  * heating appliances

  the app uses go routines and it is non blocking so you can control different pins with different delays and trigger options

### Download and run the [latest release](../../releases)
 it is single executable binary so **no dependancies** just download and run - quick and efective :thumbsup:

### Usage
   ```go
   rpi-web-control -pp password
   // -h  - help
   // -pp - required - the password that each client should use to authenticate
   // -p  - optional - the port for the server - default is 80
   ```
**open the home page:** http://raspberrypi.local  
*the RPi support avahi/bonjour so you can access it by its hostname: `raspberrypi.local`*

![Web Gui Preview](/preview.png)

```
pass  (required) the password set when you started the server using -pp  
type  (optional) timer(set 1 wait and set 0) or toggle(toggle between 1 and 0)
pin   (optional) the pin to control,    default is 18
delay (optional) delay for the timer,   default is `2s`
```
the home page sends AJAX requests to  
```
http://raspberrypi.local/control?pass=password&pin=18&type=timer&delay=2s
```

![RPi pinout](/pizeropinout.jpg)

### Build from Source (fun and educational):neckbeard:

  **[Install `go` on the RPi..](https://golang.org/doc/install)**
  ```go
  ssh pi@raspberrypi.local
  wget https://storage.googleapis.com/golang/go1.8.1.linux-armv6l.tar.gz
  tar -C /usr/local -xzf go1.8.1.linux-armv6l.tar.gz
  go get github.com/krasi-georgiev/rpi-web-control
  ~/go/bin/rpi-web-control -pp password
  ```
  *I have only test RPi-Zero but I think Pi-3 should install the amd64 version*


## Build on any system and copy it to the Pi
  ```go
  GOOS=linux GOARCH=arm GOARM=6 go build -o rpi-web-control -v *.go
  // GOOS,GOARCH - sets the target architecture. This example is for Pi Zero

  scp ./rpi-web-control pi@raspberrypi.local:/usr/local/bin/rpi-web-control
  ssh pi@raspberrypi.local
  rpi-web-control -pp password
  ```


## Create systemd service so it runs at boot and restarts if killed.

  Create the service file
  `nano /lib/systemd/system/rpi-web-control.service`

```
  [Unit]
  Description=Rpi Web Controller

  [Service]
  ExecStart=/usr/local/bin/rpi-web-control -pp password
  WatchdogSec=10s
  Restart=always

  [Install]
  WantedBy=multi-user.target
```

 Enable and start the service...

 ```
 systemctl daemon-reload
 systemctl enable rpi-web-control.service
 systemctl start rpi-web-control.service
 ```
