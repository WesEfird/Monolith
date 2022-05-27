package fileutil

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// GetShaChecksum takes in a filePath argument and returns a hex encoded string of the calculated sha-256 file checksum
func GetShaChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		return "", errors.New("err creating a file handle on: " + filePath)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", errors.New("err calculating sha-256 checksum of file: " + filePath)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
