package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

var gpioPins = []int{}

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"

var hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}

var config struct {
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
		// TODO validate the input using parse functions
		config.pin = c.String("pin")
		config.delay = time.Duration(c.Int("delay")) * time.Second

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

		srv := &http.Server{Addr: ":" + c.String("port")}

		http.HandleFunc("/toggle", toggle)

		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "status ok")
		})
		go func() {
			log.Print("started web server on port :", c.String("port"))
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

func toggle(w http.ResponseWriter, r *http.Request) {
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

// ParseLevel takes a string level and returns the Logrus log level constant.
// func ParseDelay(lvl string) (Level, error) {
// 	switch strings.ToLower(lvl) {
// 	case "panic":
// 		return PanicLevel, nil
// 	case "fatal":
// 		return FatalLevel, nil
// 	case "error":
// 		return ErrorLevel, nil
// 	case "warn", "warning":
// 		return WarnLevel, nil
// 	case "info":
// 		return InfoLevel, nil
// 	case "debug":
// 		return DebugLevel, nil
// 	}
//
// 	var l Level
// 	return l, fmt.Errorf("not a valid logrus Level: %q", lvl)
// }
