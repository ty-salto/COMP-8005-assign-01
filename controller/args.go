package main

import (
	"flag"
	"fmt"
)

func parseArgs() (shadowPath, username string, port int, err error) {
	flag.StringVar(&shadowPath, "f", "", "path to shadow file")
	flag.StringVar(&username, "u", "", "username")
	flag.IntVar(&port, "p", 0, "port")
	flag.Parse()

	if shadowPath == "" || username == "" || port <= 0 || port > 65535 {
		return "", "", 0, fmt.Errorf("missing required argument")
	}
	return shadowPath, username, port, nil
}
