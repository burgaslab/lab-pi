# Raspberry PI web GPIO controller 
*a very minimalistic app to control the RPI outputs using a web api* 

## Usage

  start the server... 
  ```go
  go get github.com/krasi-georgiev/rpi-gpio-web-api
  
  rpi-gpio-web-api -pp password // or ~/go/bin/rpi-gpio-web-api
  
  // -h  - help
  // -pp - required - the password that each client should use to authenticate
  // -p  - optional - default is 80 , the port for the server
  ```

  control the RPI from a web browser...
  
  http://raspberrypi.local/control?pass=password&pin=21&type=timer&delay=3s
  ```go
  // raspberrypi.local - the RPI support avahi/bonjour so you can access it by its hostname
  // pass  - required , the password set when you started the server using -pp
  // pin   - optional, the GPIO pin you want to control, if not provided it will default to 21 (it is next to GND and easy to mesure)
  // type`  - optional, default is `timer`. timer(set 1 wait and set 0) or toggle(toggle between 1 and 0 for every request)
  // delay` - optional, default is `3s`. The delay for the timer function.
  ```
  

  
## Build on any system and copy it to the PI
  ```go
  GOOS=linux GOARCH=arm GOARM=6 go build -o rpi-gpio-web-api -v *.go
  // GOOS=linux GOARCH=arm - sets the target executable architecture

  scp ./rpi-gpio-web-api pi@raspberrypi.local:/usr/local/bin/rpi-gpio-web-api
  ssh pi@rpi-gpio-web-api
  rpi-gpio-web-api -pp password
  ```


## Create systemd service so it runs it at boot and restarts if killed.

*/lib/systemd/system/door-opener.service*

```
  [Unit]
  Description=Lab Door Opener

  [Service]
  ExecStart=/usr/local/bin/door-opener
  Restart=always

  [Install]
  WantedBy=multi-user.target
```

3.Enable and start the service

 ```
 systemctl daemon-reload
 systemctl enable door-opener.service
 systemctl start door-opener.service
 ```

**TODO**

- [x] install golang on the PI so we can compile directly to save time  
- [x] control the PI outputs to connect a relay which will control the door
- [ ] build the home page
- [x] add some simple authentication
- [ ] implement healthcheck - maybe using curl ? and restart the service if failed
- [ ] run using docker
