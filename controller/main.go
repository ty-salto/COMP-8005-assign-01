package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"assign1/internal/messages"
)

const PasswordLen = 3

// Put your exact required 79-char charset here.
const LegalCharset79 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./0123456789:;<=>?@[\\]^_`{|}~"

func main() {
	startTotal := time.Now()

	shadowPath, username, port, err := parseArgs()
	if err != nil {
		usage(err)
		os.Exit(2)
	}

	startParse := time.Now()
	fullHash, err := loadShadowHash(shadowPath, username)
	if err != nil {
		usage(err)
		os.Exit(2)
	}
	alg, err := detectAlg(fullHash)
	if err != nil {
		usage(err)
		os.Exit(2)
	}
	parseDur := time.Since(startParse)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		usage(fmt.Errorf("listen failed: %w", err))
		os.Exit(2)
	}
	defer ln.Close()

	fmt.Printf("[controller] listening on :%d\n", port)

	conn, err := ln.Accept()
	if err != nil {
		usage(fmt.Errorf("accept failed: %w", err))
		os.Exit(2)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)

	// Expect REGISTER
	var reg messages.RegisterMsg
	if err := messages.RecvLine(r, &reg); err != nil {
		usage(fmt.Errorf("read REGISTER failed: %w", err))
		os.Exit(2)
	}
	if reg.Type != messages.REGISTER {
		_ = messages.Send(conn, messages.AckMsg{Type: messages.ACK, Status: "ERROR", Error: "expected REGISTER"})
		usage(fmt.Errorf("protocol error: expected REGISTER"))
		os.Exit(2)
	}
	_ = messages.Send(conn, messages.AckMsg{Type: messages.ACK, Status: "OK"})

	// Send JOB
	if LegalCharset79 == "TODO_REPLACE_WITH_EXACT_79_CHARSET" {
		usage(fmt.Errorf("set LegalCharset79 in controller/main.go and internal/messages/messages.go (job charset must match)"))
		os.Exit(2)
	}

	job := messages.JobMsg{
		Type:        messages.JOB,
		Username:    username,
		FullHash:    fullHash,
		Alg:         alg,
		Charset:     LegalCharset79,
		PasswordLen: PasswordLen,
	}

	startDispatch := time.Now()
	if err := messages.Send(conn, job); err != nil {
		usage(fmt.Errorf("send JOB failed: %w", err))
		os.Exit(2)
	}
	dispatchDur := time.Since(startDispatch)

	// Wait RESULT
	startReturn := time.Now()
	var res messages.ResultMsg
	if err := messages.RecvLine(r, &res); err != nil {
		usage(fmt.Errorf("read RESULT failed: %w", err))
		os.Exit(2)
	}
	returnDur := time.Since(startReturn)

	if res.Type != messages.RESULT {
		usage(fmt.Errorf("protocol error: expected RESULT"))
		os.Exit(2)
	}

	// Report
	fmt.Println("----- FINAL RESULT -----")
	fmt.Printf("status: %s\n", res.Status)
	if res.Status == "FOUND" {
		fmt.Printf("password: %s\n", res.Password)
	}
	if res.Status == "ERROR" {
		fmt.Printf("error: %s\n", res.Error)
	}

	fmt.Println("----- TIMING -----")
	fmt.Printf("controller_parse_ms: %.3f\n", float64(parseDur.Microseconds())/1000.0)
	fmt.Printf("job_dispatch_ms: %.3f\n", float64(dispatchDur.Microseconds())/1000.0)
	fmt.Printf("worker_compute_ms: %.3f\n", float64(res.WorkerComputeNs)/1e6)
	fmt.Printf("result_return_ms: %.3f\n", float64(returnDur.Microseconds())/1000.0)
	fmt.Printf("total_end_to_end_ms: %.3f\n", float64(time.Since(startTotal).Microseconds())/1000.0)
}

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

func loadShadowHash(path, username string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot open shadow file: %w", err)
	}
	lines := strings.Split(string(b), "\n")
	prefix := username + ":"
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			parts := strings.Split(line, ":")
			if len(parts) < 2 || parts[1] == "" {
				return "", fmt.Errorf("malformed or unsupported hash entry")
			}
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("username not in shadow file")
}

func detectAlg(fullHash string) (string, error) {
	switch {
	case strings.HasPrefix(fullHash, "$1$"):
		return "md5", nil
	case strings.HasPrefix(fullHash, "$5$"):
		return "sha256", nil
	case strings.HasPrefix(fullHash, "$6$"):
		return "sha512", nil
	case strings.HasPrefix(fullHash, "$2a$") || strings.HasPrefix(fullHash, "$2b$") || strings.HasPrefix(fullHash, "$2y$"):
		return "bcrypt", nil
	case strings.HasPrefix(fullHash, "$y$") || strings.HasPrefix(fullHash, "$7$"):
		return "yescrypt", nil
	default:
		return "", fmt.Errorf("malformed or unsupported hash entry")
	}
}

func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	fmt.Fprintln(os.Stderr, "usage: controller -f <shadow file> -u <username> -p <port>")
}
