# Raspberry PI web GPIO controller 
*a very minimalistic app to control the RPI outputs using a web api* 


## Usage

  start the server... 
  ```go
  ssh pi@raspberrypi.local
  go get github.com/krasi-georgiev/rpi-web-control
  ~/go/bin/rpi-web-control -pp password
  
  // -h  - help
  // -pp - required - the password that each client should use to authenticate
  // -p  - optional - default is 80 , the port for the server
  ```

  control the RPI from a web browser...
  
  http://raspberrypi.local/control?pass=password&pin=21&type=timer&delay=3s
  ```go
  
  // pass  - required - the password set when you started the server using -pp
  // pin   - optional - default is 21(next to gnd), the pin you want to control
  // type  - optional - default is `timer`. timer(set 1 wait and set 0) or toggle(toggle between 1 and 0)
  // delay - optional - default is `3s`. The delay for the timer.
  ```
* the RPI support avahi/bonjour so you can access it by its hostname: *raspberrypi.local*  

![RPI pinout](/pizeropinout.jpg)

  
## Build on any system and copy it to the PI
  ```go
  GOOS=linux GOARCH=arm GOARM=6 go build -o rpi-web-control -v *.go
  // GOOS=linux GOARCH=arm - sets the target executable architecture

  scp ./rpi-web-control pi@raspberrypi.local:/usr/local/bin/rpi-web-control
  ssh pi@raspberrypi.local
  rpi-web-control -pp password
  ```


## Create systemd service so it runs at boot and restarts if killed.

*/lib/systemd/system/rpi-web-control.service*

```
  [Unit]
  Description=Rpi Web Controller

  [Service]
  ExecStart=/usr/local/bin/rpi-web-control
  Restart=always

  [Install]
  WantedBy=multi-user.target
```

3.Enable and start the service

 ```
 systemctl daemon-reload
 systemctl enable rpi-web-control.service
 systemctl start rpi-web-control.service
 ```

**TODO**

- [x] install golang on the PI so we can compile directly to save time  
- [x] control the PI outputs to connect a relay which will control the door
- [ ] build the home page
- [x] add some simple authentication
- [ ] implement healthcheck - maybe using curl ? and restart the service if failed
- [ ] run using docker
