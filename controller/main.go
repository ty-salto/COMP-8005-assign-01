package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"assign1/internal/constants"
	"assign1/internal/messages"
	"assign1/internal/waiting"
)

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
	fmt.Println("[controller] worker connected")

	defer fmt.Println("[controller] Closing Connection")
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
	job := messages.JobMsg{
		Type:        messages.JOB,
		Username:    username,
		FullHash:    fullHash,
		Alg:         alg,
		Charset:     constants.LegalCharset79,
		PasswordLen: constants.PasswordLen,
	}

	startDispatch := time.Now()
	if err := messages.Send(conn, job); err != nil {
		usage(fmt.Errorf("send JOB failed: %w", err))
		os.Exit(2)
	}
	dispatchDur := time.Since(startDispatch)

	// Wait RESULT
	done := make(chan struct{}) 
	waiting.StartDots(done, "[controller] waiting result")
	
	startReturn := time.Now()
	var res messages.ResultMsg
	if err := messages.RecvLine(r, &res); err != nil {
		usage(fmt.Errorf("read RESULT failed: %w", err))
		os.Exit(2)
	}

	close(done)
	fmt.Println("\n[controller] Recieved result")
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

func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	fmt.Fprintln(os.Stderr, "usage: controller -f <shadow file> -u <username> -p <port>")
}
