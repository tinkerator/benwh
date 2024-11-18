// Program status queries the status of a specific Site Device.
package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
	"zappem.net/pub/net/benwh"
)

var (
	email    = flag.String("email", "", "account email address")
	devices  = flag.String("devices", "", "comma separated device list")
	config   = flag.String("config", "./benwh.config", "config file location")
	newLogin = flag.Bool("newlogin", false, "create new login config file")
	debug    = flag.Bool("debug", false, "show all status data")
	poll     = flag.Int("poll", 1, "number of samples to take before exit")
	delay    = flag.Duration("delay", 0, "time to wait between service calls")
)

// createConfig reads information for the device config.
func createConfig() (conf benwh.Config, err error) {
	conf.Email = *email
	conf.Device = strings.Split(*devices, ",")
	r := bufio.NewReader(os.Stdin)
	if *email == "" {
		fmt.Print("Email: ")
		conf.Email, err = r.ReadString('\n')
		if err != nil {
			err = fmt.Errorf("failed to read username: %v", err)
			return
		}
		conf.Email = strings.Trim(conf.Email, " \n\r")
	}
	if len(conf.Device) == 0 || conf.Device[0] == "" {
		conf.Device = nil
		fmt.Print("Side Device ID: ")
		dev, err2 := r.ReadString('\n')
		if err2 != nil {
			err = fmt.Errorf("failed to read site device: %v", err2)
			return
		}
		conf.Device = append(conf.Device, strings.Trim(dev, " \n\r"))
	}
	fmt.Print("Password: ")
	pass, err2 := term.ReadPassword(syscall.Stdin)
	if err2 != nil {
		err = fmt.Errorf("failed to read password: %v", err2)
		return
	}
	h := md5.Sum([]byte(pass))
	conf.Password = hex.EncodeToString(h[:])
	return
}

func main() {
	flag.Parse()

	var conf benwh.Config
	var err error

	if *newLogin {
		conf, err = createConfig()
		if err != nil {
			log.Fatalf("config creation failed: %v", err)
		}
	} else {
		d, err := os.ReadFile(*config)
		if err != nil {
			log.Fatalf("unable to read --config=%q: %v", *config, err)
		}
		if err := json.Unmarshal(d, &conf); err != nil {
			log.Fatalf("unable to decode --config=%q: %v", *config, err)
		}
	}

	conn, err := benwh.NewConn(conf)
	if err != nil {
		log.Fatalf("unable to authenticate a connection: %v", err)
	}

	if *newLogin {
		d, err := json.Marshal(conf)
		if err != nil {
			log.Fatalf("unable to marshal --config=%q: %v", *config, err)
		}
		if err := os.WriteFile(*config, d, 0600); err != nil {
			log.Fatalf("failed to write --config=%q: %v", *config, err)
		}
	}

	samples := 0
	backoff := 5 * time.Second
	for first := true; ; first = false {
		resp, err := conn.Status()
		switch err {
		case nil:
			backoff = 5 * time.Second
		case benwh.ErrRetryLater:
			backoff = backoff * 2
			log.Printf("no data received (waiting %v): %v", backoff, err)
			time.Sleep(backoff)
			continue
		default:
			log.Fatalf("failed to obtain status: %v", err)
		}
		if *debug {
			log.Printf("resp %#v", resp)
		} else {
			if first {
				log.Print("(kW) Utility    Solar     Gen  A-Gate   House  %Charge")
			}
			log.Printf("      %6.3f   %6.3f  %6.3f  %6.3f  %6.3f   %6.3f", resp.PUti, resp.PSun, resp.PGen, resp.PFhp, resp.PLoad, resp.Soc)
		}
		if *delay == 0 {
			break
		}
		if samples >= 0 {
			samples++
		}
		if samples == *poll {
			break
		}
		time.Sleep(*delay)
	}
}
