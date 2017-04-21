package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

// TO DO maybe bind to phisical pin locations on the board
var gpioPins = []int{4, 17, 27, 22, 5, 6, 13, 19, 26, 18, 23, 24, 25, 12, 16, 20, 21}

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"

var hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}

var config struct {
	port  string
	pin   string
	delay time.Duration
}

func main() {

	app := cli.NewApp()
	app.Name = "Burgas Lab Door Opener"
	app.Version = "17.04"
	app.Compiled = time.Now()
	app.Usage = "Raspberry Pi based golang app to control a door relay via a web interface"

	app.Flags = []cli.Flag{
		cli.UintFlag{
			Name:  "p,port",
			Value: 80,
			Usage: "port for the webserver",
		},
		cli.UintFlag{
			Name:  "pp,pin",
			Value: 21,
			Usage: "GPIO pin to control",
		},
		cli.UintFlag{
			Name:  "d,delay",
			Value: 3,
			Usage: "the total time in seconds that this GPIO pin will be set to high level",
		},
	}

	app.Action = func(c *cli.Context) error {
		var err error
		config.pin, err = parsePin(c.Int("pin"))
		if err != nil {
			return err
		}
		config.port, err = parsePort(c.Uint("port"))
		if err != nil {
			return err
		}
		config.delay = time.Duration(c.Uint("delay")) * time.Second

		if err != nil {
			return err
		}
		// start the signal handler as soon as we can to make sure that
		// we don't miss any signals during boot
		signals := make(chan os.Signal, 128)
		signal.Notify(signals, hanldeSignals...)

		// enable if not already enabled
		if _, err := os.Stat(sysfs + "gpio" + config.pin); os.IsNotExist(err) {
			// enable the GPIO interface and set gpion pin as an output
			if _, err := os.Stat(sysfsGPIOenable); os.IsNotExist(err) {
				log.Fatal(err)
			}

			if err := ioutil.WriteFile(sysfsGPIOenable, []byte(config.pin), 0644); err != nil {
				log.Fatal(err)
			}
		}
		if err := ioutil.WriteFile(sysfs+"gpio"+config.pin+"/direction", []byte("out"), 0644); err != nil {
			log.Fatal(err)
		}
		log.Printf("Trigger enabled for pin %v", config.pin)

		srv := &http.Server{Addr: ":" + config.port}

		http.HandleFunc("/trigger", trigger)

		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		})

		go func() {
			log.Print("Started web server on port ", config.port)
			if err := srv.ListenAndServe(); err != nil {
				log.Printf("Httpserver: ListenAndServe() error: %s", err)
				os.Exit(1)
			}
		}()

		return cleanup(signals, srv)
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "door opener: %s\n", err)
		os.Exit(1)
	}

}

func trigger(w http.ResponseWriter, r *http.Request) {
	go func() {
		// toggle the pin output using a set delay
		if err := ioutil.WriteFile(sysfs+"gpio"+config.pin+"/value", []byte("1"), 0644); err != nil {
			log.Fatal(err)
		}
		log.Printf("Pin %v enabled for %v seconds", config.pin, config.delay)
		time.Sleep(config.delay)
		if err := ioutil.WriteFile(sysfs+"gpio"+config.pin+"/value", []byte("0"), 0644); err != nil {
			log.Fatal(err)
		}
		log.Printf("Delay expired : pin %v set to disabled ", config.pin)
	}()
	fmt.Fprintf(w, "ok")
}

func cleanup(signals chan os.Signal, srv *http.Server) error {
	for s := range signals {
		log.Print("Received signal: ", s)

		switch s {
		default:
			if err := srv.Shutdown(nil); err != nil {
				panic(err) // failure/timeout shutting down the server gracefully
			}
			err := ioutil.WriteFile(sysfsGPIOdisable, []byte(config.pin), 0644)
			if err != nil {
				log.Fatal(err)
			}
			log.Print("cleaning up the mess I created")
		}
		return nil
	}
	return nil
}

func parsePort(p uint) (string, error) {
	if p > 0 && p < 65536 {
		return fmt.Sprint(p), nil
	}
	return "", errors.New("Invalid port number:" + fmt.Sprint(p) + ", select a port between 1 and 65535")

}

func parsePin(p int) (string, error) {
	sort.Ints(gpioPins)

	for _, v := range gpioPins {
		if v == p {
			return fmt.Sprint(p), nil
		}
	}
	e := fmt.Sprintf("Invalid GPIO pin number:%v, choose one of :%v", p, gpioPins)
	return "", errors.New(e)
}
