package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"assign1/internal/messages"
	"assign1/internal/waiting"
)

func main() {
	host, port, err := parseArgs()
	if err != nil {
		usage(err)
		os.Exit(2)
	}


	fmt.Println("[worker] Connecting...")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		usage(fmt.Errorf("controller unreachable: %w", err))
		os.Exit(2)
	}
	fmt.Println("[worker] Connected")
	defer conn.Close()

	r := bufio.NewReader(conn)

	// REGISTER
	reg := messages.RegisterMsg{
		Type:    messages.REGISTER,
		Worker:  hostnameOr("worker"),
	}


	fmt.Println("[worker] Registering...")
	if err := messages.Send(conn, reg); err != nil {
		usage(fmt.Errorf("send REGISTER failed: %w", err))
		os.Exit(2)
	}
	fmt.Println("[worker] Closing Listener")

	var ack messages.AckMsg
	if err := messages.RecvLine(r, &ack); err != nil {
		usage(fmt.Errorf("read ACK failed: %w", err))
		os.Exit(2)
	}
	if ack.Type != messages.ACK || ack.Status != "OK" {
		usage(fmt.Errorf("registration rejected: %s", ack.Error))
		os.Exit(2)
	}

	fmt.Println("[worker] Register Successful")

	// Heartbeat later (around here)

	// JOB
	fmt.Println("[worker] Receiving Job...")
	var job messages.JobMsg
	if err := messages.RecvLine(r, &job); err != nil {
		sendError(conn, fmt.Errorf("read JOB failed: %w", err))
		os.Exit(2)
	}
	if job.Type != messages.JOB {
		sendError(conn, fmt.Errorf("protocol error: expected JOB"))
		os.Exit(2)
	}

	if err := validateJob(&job); err != nil {
		sendError(conn, err)
		os.Exit(2)
	}
	fmt.Println("[worker] Received Job")

	// CRACK (single-threaded)
	fmt.Println("[worker] Cracking password")
	done := make(chan struct{}) 
	waiting.StartDots(done, "[worker] cracking")
	
	startCompute := time.Now()
	res := crack(&job)
	res.WorkerComputeNs = time.Since(startCompute).Nanoseconds()

	// RESULT (exactly one final result)
	fmt.Println("\n[worker] Sending result...")
	if err := messages.Send(conn, res); err != nil {
		close(done)
		fmt.Fprintf(os.Stderr, "send RESULT failed: %v\n", err)
		os.Exit(2)
	}
	close(done)
	fmt.Println("[worker] Sent result")
}

func sendError(conn net.Conn, err error) {
	_ = messages.Send(conn, &messages.ResultMsg{
		Type:   messages.RESULT,
		Status: "ERROR",
		Error:  err.Error(),
	})
}

func usage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	fmt.Fprintln(os.Stderr, "usage: worker -c <controller_host> -p <port>")
}

func hostnameOr(fallback string) string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return fallback
	}
	return h
}
