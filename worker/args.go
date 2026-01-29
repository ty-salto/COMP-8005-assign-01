package main

import (
	"flag"
	"fmt"
)

func parseArgs() (host string, port int, err error) {
	flag.StringVar(&host, "c", "", "controller host")
	flag.IntVar(&port, "p", 0, "controller port")
	flag.Parse()

	if host == "" || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("missing required argument")
	}
	return host, port, nil
}