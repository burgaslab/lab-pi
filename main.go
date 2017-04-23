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

	"github.com/urfave/cli"
)

var (
	// TO DO maybe bind to phisical pin locations on the board
	gpioPins      = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27}
	hanldeSignals = []os.Signal{syscall.SIGINT, syscall.SIGKILL}
	httpC         = newHTTPConfig()
	app           = cli.NewApp()
)

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"

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
		fmt.Fprint(w, err)
	}
	if err := t.setDelay(v); err != nil {
		fmt.Fprint(w, err)
	}
	if err := t.setPin(v); err != nil {
		fmt.Fprint(w, err)
	}

	go func() {
		switch a := t.Type; a {
		case "timer":
			if err := t.startTimer(); err != nil {
				log.Printf("Huston we have a problem with the pin timer %v", err)
			}
		case "toggle":
			if err := t.toggle(); err != nil {
				log.Printf("Huston we have a problem with the pin toggle %v", err)
			}
		}
	}()
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
		Pin:   "21",
		Delay: 3 * time.Second,
	}
}

type rpiControl struct {
	Type  string
	Pin   string
	Delay time.Duration
}

func (c *rpiControl) setType(url url.Values) error {
	if d, ok := url["type"]; ok {
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
	if p, ok := url["Pin"]; ok {
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
	if d, ok := url["delay"]; ok {
		if t, err := time.ParseDuration(d[0]); err == nil {
			c.Delay = t
			return nil
		}
		e := fmt.Sprintf("Invalid time delay format :%v (use 1ms, 1s1, 1m1, 1h)", d[0])
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
func (c *rpiControl) startTimer() error {
	if err := c.enablePin(); err != nil {
		log.Printf("I couldn't enable pin %v, because %v", c.Pin, err)
	}
	if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("1"), 0644); err != nil {
		return err
	}
	log.Printf("Pin %v set to 1 for %v seconds", c.Pin, c.Delay)
	time.Sleep(c.Delay)
	if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("0"), 0644); err != nil {
		return err
	}
	log.Printf("Pin %v set to 0", c.Pin)
	return nil
}

func (c *rpiControl) toggle() error {
	if err := c.enablePin(); err != nil {
		log.Printf("I couldn't enable pin %v, because %v", c.Pin, err)
	}

	d, err := ioutil.ReadFile(sysfs + "gpio" + c.Pin + "/value")
	if err != nil {
		log.Printf("Oh boy can't read the status of pin  %v becasue I don't have my glasses and %v", c.Pin, err)
	}

	if string(d) == "1\n" {
		if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("0"), 0644); err != nil {
			return err
		}
		log.Printf("Congrats pin  %v is set to level 0", c.Pin)
		return nil
	}
	if err := ioutil.WriteFile(sysfs+"gpio"+c.Pin+"/value", []byte("1"), 0644); err != nil {
		return err
	}
	log.Printf("Congrats pin  %v is set to level 1", c.Pin)
	return nil
}
