# Raspberry PI door opener

*A simple golang app with a build in http server as a web front that controls the GPIO of the raspberry pi*



1. build the executable binary
  ```
  GOOS=linux GOARCH=arm GOARM=6 go build -v *.go
  ```
* GOOS=linux GOARCH=arm - sets the executable architecture so the executable can be build on any system and then run it on the PI

2. copy over to the PI

```
  scp ./door-opener pi@192.168.1.11:/usr/local/bin/door-opener
```
* ssh to the PI and run the binary /usr/local/bin/door-opener [-h for all cli options]


3. create systemd service so it runs it at boot and restarts if killed.

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

4.Enable and start the service

 ```
 systemctl daemon-reload
 systemctl enable door-opener.service
 systemctl start door-opener.service
 ```

**TODO**

- [x] install golang on the PI so we can compile directly to save time  
- [x] control the PI outputs to connect a relay which will control the door
- [ ] build the home page
- [ ] add some simple authentication
- [ ] implement healthcheck - maybe using curl ? and restart the service if failed


 ## NEXT Project
- [ ] stream music to the bluetooth player
