# Raspberry PI web GPIO controller 
*a very minimalistic app to control the RPI outputs using a web api* 

## Usage

  start the server... 
  ```
  rpi-controller -pp password
  ```
  control a pin from a web browser...
  
  ```http://ip?pass=password&pin=21&type=timer```
  

1. build the executable binary

**- Build on any system and copy it to the PI**
  ```
  GOOS=linux GOARCH=arm GOARM=6 go build -o rpi-controller -v *.go
  ```
  * GOOS=linux GOARCH=arm - sets the executable architecture so the executable can be build on any system and then run it on the PI
  ```
    scp ./rpi-controller pi@192.168.1.11:/usr/local/bin/rpi-controller
  ```
  * ssh to the PI and run the binary `/usr/local/bin/rpi-controller [-h for all cli options]`

**- Build directly on the PI**
  ```
  cd /root/go/src/github.com/krasi-georgiev/rpi-gpio-web-api
  go run -pp password
  ```


2. create systemd service so it runs it at boot and restarts if killed.

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
