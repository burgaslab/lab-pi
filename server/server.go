package server

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/urfave/cli"
)

// NewConfig boostraps the server port number and authentication token
func NewConfig() *Config {
	return &Config{}
}

// Config holdes the port numver and password
type Config struct {
	Port string
	pass string
}

// SetPort is the port setter
func (c *Config) SetPort(cli *cli.Context) error {
	p := cli.Uint("port")
	if p > 0 && p < 65536 {
		c.Port = cli.String("port")
		return nil
	}
	return errors.New("Invalid port number:" + fmt.Sprint(p) + ", select a port between 1 and 65535")

}

// SetPass is the port setter
func (c *Config) SetPass(cli *cli.Context) error {
	if cli.String("password") == "" {
		return errors.New("Password can't be empty")
	}
	c.pass = cli.String("password")
	return nil
}

// Authenticate is for protected pages
func (c *Config) Authenticate(url url.Values) error {
	if d, ok := url["pass"]; ok && c.pass == d[0] {
		return nil
	}
	return errors.New("No accesso amiho")
}
