package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"assign1/internal/messages"
)

const PasswordLen = 3
const LegalCharset79 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./0123456789:;<=>?@[\\]^_`{|}~"

func main() {
	host, port, err := parseArgs()
	if err != nil {
		usage(err)
		os.Exit(2)
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		usage(fmt.Errorf("controller unreachable: %w", err))
		os.Exit(2)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)

	// REGISTER
	reg := messages.RegisterMsg{
		Type:    messages.REGISTER,
		Worker:  hostnameOr("worker"),
		Version: "v1",
	}
	if err := messages.Send(conn, reg); err != nil {
		usage(fmt.Errorf("send REGISTER failed: %w", err))
		os.Exit(2)
	}

	var ack messages.AckMsg
	if err := messages.RecvLine(r, &ack); err != nil {
		usage(fmt.Errorf("read ACK failed: %w", err))
		os.Exit(2)
	}
	if ack.Type != messages.ACK || ack.Status != "OK" {
		usage(fmt.Errorf("registration rejected: %s", ack.Error))
		os.Exit(2)
	}

	// Heartbeat later (not now):
	// go heartbeatLoop(conn)

	// JOB
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

	// CRACK (single-threaded)
	startCompute := time.Now()
	res := crack(&job)
	res.WorkerComputeNs = time.Since(startCompute).Nanoseconds()

	// RESULT (exactly one final result)
	if err := messages.Send(conn, res); err != nil {
		fmt.Fprintf(os.Stderr, "send RESULT failed: %v\n", err)
		os.Exit(2)
	}
}

func parseArgs() (host string, port int, err error) {
	flag.StringVar(&host, "c", "", "controller host")
	flag.IntVar(&port, "p", 0, "controller port")
	flag.Parse()

	if host == "" || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("missing required argument")
	}
	return host, port, nil
}

func validateJob(job *messages.JobMsg) error {
	if job.PasswordLen != PasswordLen {
		return fmt.Errorf("invalid password length")
	}
	if job.Charset != LegalCharset79 {
		return fmt.Errorf("invalid charset")
	}
	switch job.Alg {
	case "yescrypt", "bcrypt", "sha256", "sha512", "md5":
	default:
		return fmt.Errorf("unsupported hash algorithm")
	}
	if job.FullHash == "" {
		return fmt.Errorf("empty hash field")
	}
	return nil
}

func crack(job *messages.JobMsg) *messages.ResultMsg {
	// Deterministic sequential enumeration over full search space.
	charset := job.Charset
	base := len(charset)
	space := 1
	for i := 0; i < job.PasswordLen; i++ {
		space *= base
	}

	for idx := 0; idx < space; idx++ {
		cand := indexToCandidate(idx, charset, job.PasswordLen)

		// 🔍 print current candidate
		fmt.Printf("testing candidate: %s\n", cand)

		ok, err := verifyCandidate(job.Alg, cand, job.FullHash)
		if err != nil {
			return &messages.ResultMsg{Type: messages.RESULT, Status: "ERROR", Error: err.Error()}
		}
		if ok {
			return &messages.ResultMsg{Type: messages.RESULT, Status: "FOUND", Password: cand}
		}
	}
	return &messages.ResultMsg{Type: messages.RESULT, Status: "NOT_FOUND"}
}

func indexToCandidate(idx int, charset string, length int) string {
	base := len(charset)
	out := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		out[i] = charset[idx%base]
		idx /= base
	}
	return string(out)
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
