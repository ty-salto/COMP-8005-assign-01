//go:build linux

package main

/*
#cgo LDFLAGS: -lcrypt
#include <crypt.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

func cryptHash(candidate, fullHash string) (string, error) {
	// For modern hashes, the "salt" parameter is actually the full hash prefix
	// (e.g., "$6$salt$...") which includes algorithm + salt + params.
	cCand := C.CString(candidate)
	cSalt := C.CString(fullHash)
	defer C.free(unsafe.Pointer(cCand))
	defer C.free(unsafe.Pointer(cSalt))

	out := C.crypt(cCand, cSalt)
	if out == nil {
		return "", errors.New("crypt() returned NULL")
	}
	return C.GoString(out), nil
}
