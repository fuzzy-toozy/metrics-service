package encryption

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func SignData(data, key []byte) (string, error) {
	hmac := hmac.New(sha256.New, key)
	_, err := hmac.Write(data)
	if err != nil {
		return "", err
	}

	hash := hmac.Sum(nil)

	return hex.EncodeToString(hash), nil
}

func CheckData(data, key []byte, hash string) error {
	newHash, err := SignData(data, key)
	if err != nil {
		return err
	}

	if newHash != hash {
		return fmt.Errorf("signature is invalid")
	}

	return nil
}
