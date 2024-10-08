package model

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"io"
)

const HashesPath = "hashes.json"

type HashList map[string]string

// HashReader returns the HMAC-SHA512 hash of the reader's contents.
func HashReader(secret string, reader io.Reader) (string, error) {
	hasher := hmac.New(sha512.New, []byte(secret))

	_, err := io.Copy(hasher, reader)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
