package encryption

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignAndCheckData(t *testing.T) {
	r := require.New(t)

	key := make([]byte, 32)
	_, err := rand.Read(key)
	r.NoError(err, "failed to generate random key")

	data := []byte("hello world")

	signature, err := SignData(data, key)
	r.NoError(err, "failed to sign data")

	err = CheckData(data, key, signature)
	r.NoError(err, "failed to check signature")
}

func TestEncryptDecrypt(t *testing.T) {
	r := require.New(t)

	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	r.NoError(err, "failed to generate RSA private key")

	rsaPublicKey := &rsaPrivateKey.PublicKey

	b, err := x509.MarshalPKCS8PrivateKey(rsaPrivateKey)
	r.NoError(err, "faield to marshal RSA private key")
	privateKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: b,
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(rsaPublicKey)
	r.NoError(err, "failed to marshal public key")
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	parsedPublicKey, err := ParseRSAPublicKey(publicKeyPEM)
	r.NoError(err, "failed to parse public key")

	parsedPrivateKey, err := ParseRSAPrivateKey(privateKeyBytes)
	r.NoError(err, "failed to parse private key")

	message := []byte("hello world")

	encrypted, err := EncryptOAEP(parsedPublicKey, message)
	r.NoError(err, "encryption failed")

	decrypted, err := DecryptOAEP(parsedPrivateKey, encrypted)
	r.NoError(err, "decryption failed")

	r.Equal(string(message), string(decrypted), "decrypted message does not match original")
}

func TestEncryptDecryptAES(t *testing.T) {
	r := require.New(t)

	key := make([]byte, 32) // AES-256 key
	_, err := rand.Read(key)
	r.NoError(err, "error generating random key")

	originalData := []byte("Hello, world!")

	encryptedData, err := EncryptAES(originalData, key)
	r.NoError(err, "error encrypting data")

	decryptedData, err := DecryptAES(encryptedData, key)
	r.NoError(err, "error decrypting data")

	r.Equal(originalData, decryptedData, "decrypted data does not match original data")
}

func TestEncryptDecryptRequestBody(t *testing.T) {
	r := require.New(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate RSA key pair")
	publicKey := &privateKey.PublicKey

	data := []byte("Hello, world!")
	dataCheck := []byte("Hello, world!")
	encryptedBody, err := EncryptRequestBody(bytes.NewBuffer(data), publicKey)
	r.NoError(err, "failed to encrypt request body")

	decryptedData, err := DecryptRequestBody(encryptedBody, privateKey)
	r.NoError(err, "failed to decrypt request body")

	r.Equal(dataCheck, decryptedData.Bytes(), "decrypted data does not match original data")
}
