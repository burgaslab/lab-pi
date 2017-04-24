# Raspberry PI web GPIO controller [![Build Status](https://api.travis-ci.org/krasi-georgiev/rpi-web-control.svg?branch=master)](https://api.travis-ci.org/krasi-georgiev/rpi-web-control)
*a very minimalistic app to control the RPI outputs using a web browser*
  * garage or front door
  * heating appliances
  
  the app uses go routines and it is non blocking so you can control different pins with different delays and trigger options 
  
### Download and run the [latest release](/releases)
 it is single executable binary so just download and run - quick and efective :)
 
### Usage
   ```go
   rpi-web-control -pp password
   // -h  - help
   // -pp - required - the password that each client should use to authenticate
   // -p  - optional - the port for the server - default is 80
   ```
**open the home page:** http://raspberrypi.local

  *the RPI support avahi/bonjour so you can access it by its hostname: `raspberrypi.local`*

  ```go
  // pass  - required - the password set when you started the server using -pp
  // pin   - optional - default is 21(next to gnd), the pin you want to control
  // type  - optional - default is `timer`. timer(set 1 wait and set 0) or toggle(toggle between 1 and 0)
  // delay - optional - default is `3s`. The delay for the timer.
  ```
  the home page sends AJAX requests to 
```
http://raspberrypi.local/control?pass=password&pin=21&type=timer&delay=3s
```

![RPI pinout](/pizeropinout.jpg)

### Build from Source (fun and educational)

  **[Install `go` on the RPI..](https://golang.org/doc/install)**
  ```go
  ssh pi@raspberrypi.local
  wget https://storage.googleapis.com/golang/go1.8.1.linux-armv6l.tar.gz
  tar -C /usr/local -xzf go1.8.1.linux-armv6l.tar.gz
  go get github.com/krasi-georgiev/rpi-web-control
  ~/go/bin/rpi-web-control -pp password
  ```
  *I have only test RPI-Zero but I think PI-3 should install the amd64 version*

  
## Build on any system and copy it to the PI
  ```go
  GOOS=linux GOARCH=arm GOARM=6 go build -o rpi-web-control -v *.go
  // GOOS,GOARCH - sets the target architecture. This example is for RPI Zero

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
  ExecStart=/usr/local/bin/rpi-web-control
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

**TODO**

- [x] install golang on the PI so we can compile directly to save time  
- [x] control the PI outputs to connect a relay which will control the door
- [x] build the home page
- [x] add some simple authentication
- [ ] implement healthcheck - maybe using curl ? and restart the service if failed
- [x] setup with travis CI to build executable on every push
