package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

var gpioPins = []int{}

var hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}

func main() {

	app := cli.NewApp()
	app.Name = "Burgas Lab Door Opener"
	app.Version = "17.04"
	app.Compiled = time.Now()
	app.Usage = "Raspberry Pi based golang app to control a door relay via a web interface"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "p,port",
			Value: "8008",
			Usage: "port for the webserver",
		},
	}

	app.Action = func(c *cli.Context) error {

		// start the signal handler as soon as we can to make sure that
		// we don't miss any signals during boot
		signals := make(chan os.Signal, 2048)
		signal.Notify(signals, hanldeSignals...)

		srv := &http.Server{Addr: ":" + c.String("port")}
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "hello world\n")
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

func cleanup(signals chan os.Signal, srv *http.Server) error {
	for s := range signals {
		log.Print("Received signal: ", s)

		switch s {
		default:
			if err := srv.Shutdown(nil); err != nil {
				panic(err) // failure/timeout shutting down the server gracefully
			}
			// []]byte("hello\ngo\n")
			// echo 21 > /sys/class/gpio/unexport
			log.Print("cleaning up the mess I created")
		}
		return nil
	}
	return nil
}
