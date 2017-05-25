package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krasi-georgiev/rpi-web-control/gpio"
	"github.com/krasi-georgiev/rpi-web-control/server"

	"github.com/coreos/go-systemd/daemon"
	"github.com/urfave/cli"
)

var (
	hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}
	srvConfig     = server.NewConfig()
	app           = cli.NewApp()
)

func main() {

	app.Name = "Raspberry Pi GPIO web controller"
	app.Version = "17.04"
	app.Compiled = time.Now()
	app.Usage = "Raspberry Pi based golang app to control raspberry Pi GPIO ports using a web interface"

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

		if err = srvConfig.SetPort(c); err != nil {
			fmt.Println("Incorrect Usage!")
			cli.ShowCommandHelp(c, "")
			return err
		}
		if err = srvConfig.SetPass(c); err != nil {
			fmt.Println("Incorrect Usage!")
			cli.ShowCommandHelp(c, "")
			return err
		}

		srv := &http.Server{Addr: ":" + srvConfig.Port}

		http.HandleFunc("/control", control)
		http.HandleFunc("/", home)

		go func() {
			log.Print("Started web server on port ", srvConfig.Port)
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
				time.Sleep(interval / 3)
				res, err := http.Get("http://127.0.0.1:" + srvConfig.Port)
				if err == nil {
					daemon.SdNotify(false, "WATCHDOG=1")
				} else {
					log.Printf("RPi Controller watchdog error: %v", err)
				}
				res.Body.Close()
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

	if err := srvConfig.Authenticate(v); err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	t := gpio.NewControl()

	if err := t.SetType(v); err != nil {
		log.Printf(err.Error())
		fmt.Fprint(w, err)
		return
	}
	if err := t.SetDelay(v); err != nil {
		log.Printf(err.Error())
		fmt.Fprint(w, err)
		return
	}
	if err := t.SetPin(v); err != nil {
		log.Printf(err.Error())
		fmt.Fprint(w, err)
		return
	}

	ch := make(chan string)
	go func() {
		switch a := t.Type; a {
		case "timer":
			if err := t.StartTimer(ch); err != nil {
				r := fmt.Sprintf("Huston we have a problem with the timer: %v", err)
				log.Printf(r)
				ch <- r
			}
		case "toggle":
			if err := t.Toggle(ch); err != nil {
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

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
		<html lang='en'>
		<head>
				<meta name='viewport' content='width=device-width, initial-scale=1, maximum-scale=1'>
				<title>RPi Web controller</title>

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
		`, gpio.DefaultPin, gpio.DefaultDelay)
}
