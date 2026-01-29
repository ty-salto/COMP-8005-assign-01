package messages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type Type string

const (
	REGISTER Type = "REGISTER"
	ACK      Type = "ACK"
	JOB      Type = "JOB"
	RESULT   Type = "RESULT"

)

type RegisterMsg struct {
	Type    Type   `json:"type"`
	Worker  string `json:"worker"`
}

type AckMsg struct {
	Type   Type   `json:"type"`
	Status string `json:"status"` // "OK" or "ERROR"
	Error  string `json:"error,omitempty"`
}

type JobMsg struct {
	Type        Type   `json:"type"`
	Username    string `json:"username"`
	FullHash    string `json:"full_hash"`
	Alg         string `json:"alg"`          // "yescrypt"|"bcrypt"|"sha256"|"sha512"|"md5"
	Charset     string `json:"charset"`      // must match required 79-char set
	PasswordLen int    `json:"password_len"` // must be 3
}

type ResultMsg struct {
	Type            Type   `json:"type"`
	Status          string `json:"status"` // "FOUND"|"NOT_FOUND"|"ERROR"
	Password        string `json:"password,omitempty"`
	Error           string `json:"error,omitempty"`
	WorkerComputeNs int64  `json:"worker_compute_ns"`
}

// --- Simple NDJSON helpers (one JSON object per line) ---
func Send(conn net.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(b, '\n'))
	return err
}

// RecvLine reads one line and unmarshals into out.
func RecvLine(r *bufio.Reader, out any) error {
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("empty message")
	}
	return json.Unmarshal([]byte(line), out)
}
