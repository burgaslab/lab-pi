package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/urfave/cli"
)

var (
	// TO DO maybe bind to phisical pin locations on the board
	gpioPins      = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 18, 22, 23, 24, 25, 26, 27}
	hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}
	httpC         = newHTTPConfig()
	app           = cli.NewApp()
)

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"
const defaultDelay = 2
const defaultPin = "18"

func main() {

	app.Name = "Raspberry PI GPIO web controller"
	app.Version = "17.04"
	app.Compiled = time.Now()
	app.Usage = "Raspberry Pi based golang app to control raspberry PI GPIO ports using a web interface"

	app.Flags = []cli.Flag{
		cli.UintFlag{
			Name:  "p,port",
			Value: 80,
			Usage: "port for the webserver",
		},
		cli.StringFlag{
			Name:  "pp,password",
			Usage: "required password for the web server",
		},
	}

	app.Action = func(c *cli.Context) error {
		// start the signal handler as soon as we can to make sure that
		// we don't miss any signals during boot
		quit := make(chan os.Signal, 128)
		signal.Notify(quit, hanldeSignals...)

		var err error

		if err = httpC.setPort(c); err != nil {
			fmt.Println("Incorrect Usage!")
			cli.ShowCommandHelp(c, "")
			return err
		}
		if err = httpC.setPass(c); err != nil {
			fmt.Println("Incorrect Usage!")
			cli.ShowCommandHelp(c, "")
			return err
		}

		srv := &http.Server{Addr: ":" + httpC.port}

		http.HandleFunc("/control", control)
		http.HandleFunc("/", home)

		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		})

		go func() {
			log.Print("Started web server on port ", httpC.port)
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				log.Printf("Httpserver: ListenAndServe() error: %s", err)
				os.Exit(1)
			}
		}()

		// send heartbeat signals to systemd otherwise it will restart the process
		go func() {
			interval, err := daemon.SdWatchdogEnabled(false)
			if err != nil || interval == 0 {
				log.Print("Watchdog not enabled for this service!")
				return
			}
			for {
				_, err := http.Get("http://127.0.0.1:" + httpC.port)
				if err == nil {
					daemon.SdNotify(false, "WATCHDOG=1")
				} else {
					log.Printf("RPI Controller watchdog error: %v", err)
				}
				time.Sleep(interval / 3)
			}
		}()

		return shutdown(quit, srv)
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func control(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse(r.RequestURI)
	v := u.Query()

	if err := httpC.authenticate(v); err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	t := newRpiControl()

	if err := t.setType(v); err != nil {
		log.Printf(err.Error())
		fmt.Fprint(w, err)
		return
	}
	if err := t.setDelay(v); err != nil {
		log.Printf(err.Error())
		fmt.Fprint(w, err)
		return
	}
	if err := t.setPin(v); err != nil {
		log.Printf(err.Error())
		fmt.Fprint(w, err)
		return
	}

	ch := make(chan string)
	go func() {
		switch a := t.Type; a {
		case "timer":
			if err := t.startTimer(ch); err != nil {
				r := fmt.Sprintf("Huston we have a problem with the timer: %v", err)
				log.Printf(r)
				ch <- r
			}
		case "toggle":
			if err := t.toggle(ch); err != nil {
				r := fmt.Sprintf("Huston we have a problem with the toggle: %v", err)
				log.Printf(r)
				ch <- r
			}
		}
	}()
	fmt.Fprint(w, <-ch)
}

func shutdown(quit chan os.Signal, srv *http.Server) error {
	log.Print("Received signal: ", <-quit)

	if err := srv.Shutdown(context.Background()); err != nil {
		return err
	}
	log.Print("gracefull shutdown!")
	return nil
}

func newHTTPConfig() *httpConfig {
	return &httpConfig{}
}

type httpConfig struct {
	port string
	pass string
}

func (c *httpConfig) setPort(cli *cli.Context) error {
	p := cli.Uint("port")
	if p > 0 && p < 65536 {
		c.port = cli.String("port")
		return nil
	}
	return errors.New("Invalid port number:" + fmt.Sprint(p) + ", select a port between 1 and 65535")

}
func (c *httpConfig) setPass(cli *cli.Context) error {
	if cli.String("password") == "" {
		return errors.New("Password can't be empty")
	}
	c.pass = cli.String("password")
	return nil
}
func (c *httpConfig) authenticate(url url.Values) error {
	if d, ok := url["pass"]; ok && httpC.pass == d[0] {
		return nil
	}
	return errors.New("No accesso amiho")
}

func newRpiControl() *rpiControl {
	return &rpiControl{
		Type:  "timer",
		Pin:   defaultPin,
		Delay: defaultDelay * time.Second,
	}
}

type rpiControl struct {
	Type  string
	Pin   string
	Delay time.Duration
}

func (c *rpiControl) setType(url url.Values) error {
	if d, ok := url["type"]; ok && d[0] != "" {
		switch v := d[0]; v {
		case "timer":
			return nil
		case "toggle":
			c.Type = v
		default:
			return errors.New("Invalid control type:" + v)
		}
	}
	return nil
}
func (c *rpiControl) setPin(url url.Values) error {
	if p, ok := url["pin"]; ok && p[0] != "" {
		for _, v := range gpioPins {
			if strconv.Itoa(v) == p[0] {
				c.Pin = p[0]
				return nil
			}
		}
		sort.Ints(gpioPins)
		e := fmt.Sprintf("Invalid GPIO pin number:%v, choose one of :%v", p, gpioPins)
		return errors.New(e)
	}
	return nil
}

func (c *rpiControl) setDelay(url url.Values) error {
	if d, ok := url["delay"]; ok && d[0] != "" {
		if t, err := time.ParseDuration(d[0]); err == nil {
			c.Delay = t
			return nil
		}
		e := fmt.Sprintf("Invalid time delay format :%v (use 1ms, 1s, 1m, 1h)", d[0])
		return errors.New(e)
	}
	return nil
}

func (c *rpiControl) enablePin() error {
	// enable if not already enabled
	if _, err := os.Stat(sysfs + "gpio" + c.Pin); os.IsNotExist(err) {
		if _, err := os.Stat(sysfsGPIOenable); os.IsNotExist(err) {
			return err
		}

		if err := ioutil.WriteFile(sysfsGPIOenable, []byte(c.Pin), 0644); err != nil {
			return err
		}
		if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/direction", []byte("out"), 0644); err != nil {
			return err
		}
		log.Printf("Output %v ready for work!", c.Pin)
	}
	return nil
}

func (c *rpiControl) disablePin() {
	if _, err := os.Stat(sysfs + "gpio" + c.Pin); os.IsNotExist(err) {
		// it is already disabled so nothing else to do, bail out
		return
	}

	err := ioutil.WriteFile(sysfsGPIOdisable, []byte(c.Pin), 0644)
	if err != nil {
		log.Printf("Oops can't disable pin %v because %v", c.Pin, err)
	}
	log.Printf("Disabled pin %v", c.Pin)
}

// enable and then disable a pin output using a set delay
func (c *rpiControl) startTimer(ch chan string) error {
	if err := c.enablePin(); err != nil {
		log.Printf("I couldn't enable pin %v, because %v", c.Pin, err)
		return err
	}
	if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("1"), 0644); err != nil {
		return err
	}
	r := fmt.Sprintf("Pin %v got 'HIGH' on drugs for %v seconds", c.Pin, c.Delay)
	log.Printf(r)
	ch <- r
	time.Sleep(c.Delay)
	if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("0"), 0644); err != nil {
		return err
	}
	log.Printf("pin %v is laid 'LOW'", c.Pin)
	return nil
}

func (c *rpiControl) toggle(ch chan string) error {
	if err := c.enablePin(); err != nil {
		log.Printf("I couldn't enable pin %v, because %v", c.Pin, err)
	}

	d, err := ioutil.ReadFile(sysfs + "gpio" + c.Pin + "/value")
	if err != nil {
		log.Printf("Oh boy can't read the status of pin	%v becasue I don't have my glasses and %v", c.Pin, err)
	}

	if string(d) == "1\n" {
		if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("0"), 0644); err != nil {
			return err
		}
		r := fmt.Sprintf("pin %v just got 'LOW' on selfesteam", c.Pin)
		log.Printf(r)
		ch <- r
		return nil
	}
	if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("1"), 0644); err != nil {
		return err
	}
	r := fmt.Sprintf("pin %v just got 'HIGH' on drugs", c.Pin)
	log.Printf(r)
	ch <- r
	return nil
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
		<html lang='en'>
		<head>
				<meta name='viewport' content='width=device-width, initial-scale=1, maximum-scale=1'>
				<title>RPI Web controller</title>

				<style>
				form {
					width: 80%%;
					margin: 0 auto;
					max-width: 400px;
					}
				body {font-size: 20px;font-family: Arial;}
				input,select {padding: 10px;font-size: 14px;width:100%%; margin:10px 0px}

				input[type=submit] {
						cursor: pointer;
						display: inline-block;
						color: #fff;
						border: 0px solid #6b963c;
						padding: 5px 10px;
						margin: 5px 0px;
						background-color:#5c9fcd;
						font-size: 30px;
				}
				#result {
						font-weight:bold;
						text-align:center;
				}
				#loaderWrapper {
					width:30px;
					margin:0 auto;
				}

				.loader {
						border: 16px solid #f3f3f3; /* Light grey */
						border-top: 16px solid #3498db; /* Blue */
						border-radius: 50%%;
						height: 30px;
						animation: spin 1s linear infinite;
				}

				@keyframes spin {
						0%% { transform: rotate(0deg); }
						100%% { transform: rotate(360deg); }
				}
				</style>
		</head>

		<body>
		<form id="controllerForm">
		<fieldset>
			<legend>Control Options</legend>
			<select id="type">
				<option value="timer">timer</option>
				<option value="toggle">toggle</option>
			</select>
			<input type="password" id="pass" placeholder="password" />
			<input type="text" id="pin" placeholder="Pin (optional, default is %v)" >
			<input type="text" id="delay" placeholder="Delay (optional, default is %v)">
			<input type="submit" value="GO">
		</fieldset>
		</form>
		<div id="loaderWrapper"></div>
		<div id="result"></div>

		<script type="text/javascript">

		var pass = getCookie("pass");
		if (pass != "") {
				document.getElementById("pass").value = pass;
		}
		var pin = getCookie("pin");
		if (pin != "") {
				document.getElementById("pin").value = pin;
		}
		var delay = getCookie("delay");
		if (delay != "") {
				document.getElementById("delay").value = delay;
		}

		var controllerForm = document.forms["controllerForm"];

		controllerForm.onsubmit = function(event){
			event.preventDefault();

			var today = new Date();
			today.setMonth(today.getMonth()+12);
			document.cookie = "pass="+document.getElementById("pass").value + ';expires=' + today.toGMTString();
			document.cookie = "pin="+document.getElementById("pin").value + ';expires=' + today.toGMTString();
			document.cookie = "delay="+document.getElementById("delay").value + ';expires=' + today.toGMTString();

			var pass="pass="+document.getElementById("pass").value;
			var type="&type="+document.getElementById("type").value;
			var pin="&pin="+document.getElementById("pin").value;
			var delay="&delay="+document.getElementById("delay").value;

			var xhttp = new XMLHttpRequest();
			xhttp.open("GET","/control?"+pass+type+pin+delay,true);

			document.getElementById("result").innerHTML = "";
			document.getElementById("loaderWrapper").classList.add('loader');

			xhttp.onload = function() {
				document.getElementById("loaderWrapper").classList.remove('loader');

				if (xhttp.status == 200) {
						document.getElementById("result").innerHTML = this.responseText;
				}
				else{
						document.getElementById("result").innerHTML = "request error";
				}
			}
			xhttp.onreadystatechange = function() {
				document.getElementById("loaderWrapper").classList.remove('loader');
				if (xhttp.readyState == 4 && xhttp.status == 0) {
					document.getElementById("result").innerHTML = "server error";
				}
			};

			xhttp.timeout = 4000;
			xhttp.ontimeout = function () {
				document.getElementById("loaderWrapper").classList.remove('loader');
				document.getElementById("result").innerHTML = "timeout";
			}
			xhttp.send();
		}


		function getCookie(cname) {
			var name = cname + "=";
			var decodedCookie = decodeURIComponent(document.cookie);
			var ca = decodedCookie.split(';');
			for(var i = 0; i <ca.length; i++) {
					var c = ca[i];
					while (c.charAt(0) == ' ') {
							c = c.substring(1);
					}
					if (c.indexOf(name) == 0) {
							return c.substring(name.length, c.length);
					}
			}
			return "";
	}
		</script>

		</body>
		</html>
		`, defaultPin, defaultDelay)
}
