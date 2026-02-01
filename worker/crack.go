package main

import (
	"fmt"
	"strings"

	"assign1/internal/constants"
	"assign1/internal/messages"

	"golang.org/x/crypto/bcrypt"
)

func crack(job *messages.JobMsg) *messages.ResultMsg {
	// Deterministic sequential enumeration over full search space.
	charset := job.Charset
	base := len(charset)

	for length :=1; length <= constants.MaxPasswordLen; length++ {
		space := 1
		for i := 0; i < length; i++ {
			space *= base
		}

		for idx := 0; idx < space; idx++ {
			cand := indexToCandidate(idx, charset, length)

			ok, err := verifyCandidate(job.Alg, cand, job.FullHash)
			if err != nil {
				return &messages.ResultMsg{Type: messages.RESULT, Status: "ERROR", Error: err.Error()}
			}
			if ok {
				return &messages.ResultMsg{Type: messages.RESULT, Status: "FOUND", Password: cand}
			}
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

func verifyCandidate(alg, candidate, fullHash string) (bool, error) {
	switch alg {
	case "bcrypt":
		// bcrypt has its own format ($2b$...)
		err := bcrypt.CompareHashAndPassword([]byte(fullHash), []byte(candidate))
		if err == nil {
			return true, nil
		}
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err

	case "md5", "sha256", "sha512", "yescrypt":
		// crypt() handles on Linux (if supported by your libc/libxcrypt)
		got, err := cryptHash(candidate, fullHash)
		if err != nil {
			return false, err
		}
		return got == fullHash, nil

	default:
		// this should never happen (keeping it explicit).
		return false, fmt.Errorf("unsupported alg: %s", strings.TrimSpace(alg))
	}
}
