package main

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

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
		// crypt(3) handles these on Linux (if supported by your libc/libxcrypt)
		got, err := cryptHash(candidate, fullHash)
		if err != nil {
			return false, err
		}
		return got == fullHash, nil

	default:
		// If you’re already sending alg strings like "sha512",
		// this should never happen — but keep it explicit.
		return false, fmt.Errorf("unsupported alg: %s", strings.TrimSpace(alg))
	}
}
