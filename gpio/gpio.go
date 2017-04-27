package gpio

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"
)

var (
	gpioPins = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 18, 22, 23, 24, 25, 26, 27}
)

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"

// DefaultDelay  used as a default if not explicitly set
const DefaultDelay = 2

// DefaultPin used as a default if not explicitly set
const DefaultPin = "18"

//NewControl the constructor with some defaults
func NewControl() *Control {
	return &Control{
		Type:  "timer",
		Pin:   DefaultPin,
		Delay: DefaultDelay * time.Second,
	}
}

// Control holds all configuration
type Control struct {
	Type  string
	Pin   string
	Delay time.Duration
}

// SetType is the controller type setter
func (c *Control) SetType(url url.Values) error {
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

//SetPin the pin on gpio that willbe controlled
func (c *Control) SetPin(url url.Values) error {
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

// SetDelay delay between enable and disable timer
func (c *Control) SetDelay(url url.Values) error {
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

func (c *Control) enablePin() error {
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

func (c *Control) disablePin() {
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

// StartTimer enable and then disable a pin output using a set delay
func (c *Control) StartTimer(ch chan string) error {
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

// Toggle between high and low state
func (c *Control) Toggle(ch chan string) error {
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
