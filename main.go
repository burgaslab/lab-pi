package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

// TO DO maybe bind to phisical pin locations on the board
var gpioPins = []int{4, 17, 27, 22, 5, 6, 13, 19, 26, 18, 23, 24, 25, 12, 16, 20, 21}

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"

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

func newRpiTrigger() *rpiTrigger {
	return &rpiTrigger{
		action: "timer",
		pin:    "21",
		delay:  3 * time.Second,
	}
}

type rpiTrigger struct {
	action string
	pin    string
	delay  time.Duration
}

func (c *rpiTrigger) setAction(url url.Values) error {
	if d, ok := url["action"]; ok {
		switch v := d[0]; v {
		case "timer":
			return nil
		case "toggle":
			c.action = v
		default:
			return errors.New("Invalid action name:" + v)
		}
	}
	return nil
}
func (c *rpiTrigger) setPin(url url.Values) error {
	if p, ok := url["pin"]; ok {
		for _, v := range gpioPins {
			if strconv.Itoa(v) == p[0] {
				c.pin = p[0]
				return nil
			}
		}
		sort.Ints(gpioPins)
		e := fmt.Sprintf("Invalid GPIO pin number:%v, choose one of :%v", p, gpioPins)
		return errors.New(e)
	}
	return nil
}

func (c *rpiTrigger) setDelay(url url.Values) error {
	if d, ok := url["delay"]; ok {
		if t, err := time.ParseDuration(d[0]); err == nil {
			c.delay = t
			return nil
		}
		e := fmt.Sprintf("Invalid time delay format :%v (use 1ms, 1s1, 1m1, 1h)", d[0])
		return errors.New(e)
	}
	return nil
}

func (c *rpiTrigger) prepareGPIO() {
	// // enable if not already enabled
	// if _, err := os.Stat(sysfs + "gpio" + httpConfig.pin); os.IsNotExist(err) {
	// 	// enable the GPIO interface and set gpion pin as an output
	// 	if _, err := os.Stat(sysfsGPIOenable); os.IsNotExist(err) {
	// 		log.Fatal(err)
	// 	}
	//
	// 	if err := ioutil.WriteFile(sysfsGPIOenable, []byte(httpConfig.pin), 0644); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	// if err := ioutil.WriteFile(sysfs+"gpio"+httpConfig.pin+"/direction", []byte("out"), 0644); err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("Trigger enabled for pin %v", httpConfig.pin)
}

var hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}

func main() {
	app := cli.NewApp()
	app.Name = "Raspberry PI GPIO web controller"
	app.Version = "17.04"
	app.Compiled = time.Now()
	app.Usage = "Raspberry Pi based golang app to trigger raspberry PI GPIO ports using a web interface"

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

		httpConfig := newHTTPConfig()
		if err = httpConfig.setPort(c); err != nil {
			return err
		}
		if err = httpConfig.setPass(c); err != nil {
			return err
		}

		srv := &http.Server{Addr: ":" + httpConfig.port}

		http.HandleFunc("/trigger", trigger)

		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		})

		go func() {
			log.Print("Started web server on port ", httpConfig.port)
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				log.Printf("Httpserver: ListenAndServe() error: %s", err)
				os.Exit(1)
			}
		}()

		return cleanup(quit, srv)
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

}

func trigger(w http.ResponseWriter, r *http.Request) {
	// // TODO set as a midleware for all requests
	// if err := authenticate(string(d["pass"])); err != nil {
	// 	fmt.Fprint(w, err.Error())
	// }

	u, _ := url.Parse(r.RequestURI)
	v := u.Query()
	t := newRpiTrigger()

	if err := t.setAction(v); err != nil {
		fmt.Fprint(w, err)
	}
	if err := t.setDelay(v); err != nil {
		fmt.Fprint(w, err)
	}
	if err := t.setPin(v); err != nil {
		fmt.Fprint(w, err)
	}

	fmt.Fprint(w, t)

	go func() {

		// pin, err = parsePin()
		// if err != nil {
		// 	return err
		// }
		// action, err = parseAction()
		//
		// if err != nil {
		// 	return err
		// }
		// delay, err = parseTime(time.Duration(c.Uint("delay")) * time.Second)
		//
		// if err != nil {
		// 	return err
		// }

	}()
}

func authenticate(pass string) error {
	// if httpConfig.passw != pass {
	// 	return errors.New("No accesso amiho")
	// }
	return nil
}

func triggerTimer() {
	// // toggle the pin output using a set delay
	// if err := ioutil.WriteFile(sysfs+"gpio"+httpConfig.pin+"/value", []byte("1"), 0644); err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("Pin %v enabled for %v seconds", httpConfig.pin, httpConfig.delay)
	// time.Sleep(httpConfig.delay)
	// if err := ioutil.WriteFile(sysfs+"gpio"+httpConfig.pin+"/value", []byte("0"), 0644); err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("Delay expired : pin %v set to disabled ", httpConfig.pin)
}

func triggerToggle() {

}

func cleanup(quit chan os.Signal, srv *http.Server) error {
	log.Print("Received signal: ", <-quit)

	if err := srv.Shutdown(context.Background()); err != nil {
		return err
	}
	// err := ioutil.WriteFile(sysfsGPIOdisable, []byte(httpConfig.pin), 0644)
	// if err != nil {
	// 	return err
	// }
	log.Print("gracefull shutdown!")
	return nil
}
