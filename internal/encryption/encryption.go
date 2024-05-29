// Package encryption Module for signing and checking signature.
package encryption

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

const encKeyLen = 8

// SignData calculates the HMAC (Hash-based Message Authentication Code) using SHA-256 hash function
// for the given data and key. It returns the hexadecimal representation of the HMAC.
// Parameters:
//   - data: The data to be signed.
//   - key: The key used for generating the HMAC.
//
// Returns:
//   - string: The hexadecimal representation of the HMAC.
//   - error: An error if any occurs during the HMAC calculation.
func SignData(data, key []byte) (string, error) {
	hmac := hmac.New(sha256.New, key)
	_, err := hmac.Write(data)
	if err != nil {
		return "", err
	}

	hash := hmac.Sum(nil)

	return hex.EncodeToString(hash), nil
}

// CheckData verifies the integrity of the data by comparing its HMAC
// with the provided hash value, using the same key. It returns an error
// if the HMAC of the data doesn't match the provided hash.
// Parameters:
//   - data: The data whose integrity needs to be verified.
//   - key: The key used for generating the HMAC.
//   - hash: The hexadecimal representation of the expected HMAC.
//
// Returns:
//   - error: An error if the calculated HMAC doesn't match the provided hash.
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

// ParseRSAPublicKey parses the given PEM-encoded private key data and returns
// the corresponding RSA private key. It expects the private key data to be in PKCS#8 format.
// Parameters:
//   - privateKeyBytes: The PEM-encoded private key data.
//
// Returns:
//   - *rsa.PrivateKey: The RSA private key parsed from the PEM data.
//   - error: An error if any occurs during parsing.
func ParseRSAPublicKey(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("parsed public key is not RSA")
	}

	return rsaPubKey, nil
}

// ParseRSAPrivateKey parses the given PEM-encoded private key data and returns
// the corresponding RSA private key. It expects the private key data to be in PKCS#8 format.
// Parameters:
//   - privateKeyBytes: The PEM-encoded private key data.
//
// Returns:
//   - *rsa.PrivateKey: The RSA private key parsed from the PEM data.
//   - error: An error if any occurs during parsing.
func ParseRSAPrivateKey(privateKeyBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %v", err)
	}
	return privateKey.(*rsa.PrivateKey), nil
}

// DecryptOAEP decrypts bytes using OAEP
// Parameters:
//   - key: The RSA private key used for decryption.
//   - encData: The encrypted data to be decrypted.
//
// Returns:
//   - []byte: The decrypted data.
//   - error: An error if decryption fails.
func DecryptOAEP(key *rsa.PrivateKey, encData []byte) ([]byte, error) {
	decData, err := rsa.DecryptOAEP(crypto.SHA256.New(), rand.Reader, key, encData, nil)
	if err != nil {
		return nil, err
	}

	return decData, nil
}

// EncryptOAEP encrypts bytes using OAEP
// Parameters:
//   - key: The RSA public key used for encryption.
//   - data: The data to be encrypted.
//
// Returns:
//   - []byte: The encrypted data.
//   - error: An error if encryption fails.
func EncryptOAEP(key *rsa.PublicKey, data []byte) ([]byte, error) {
	encData, err := rsa.EncryptOAEP(crypto.SHA256.New(), rand.Reader, key, data, nil)
	if err != nil {
		return nil, err
	}

	return encData, nil
}

// EncryptAES encrypts bytes using AES key
// Parameters:
//   - data: The data to be encrypted.
//   - key: The AES key used for encryption.
//
// Returns:
//   - []byte: The encrypted data.
//   - error: An error if encryption fails.
func EncryptAES(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// PKCS#7 padding
	padding := aes.BlockSize - len(data)%aes.BlockSize
	pad := bytes.Repeat([]byte{byte(padding)}, padding)
	data = append(data, pad...)

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], data)

	return ciphertext, nil
}

// DecryptAES decrypts data encrypted with AES in CBC mode
// Parameters:
//   - ciphertext: The ciphertext to be decrypted.
//   - key: The AES key used for decryption.
//
// Returns:
//   - []byte: The decrypted data.
//   - error: An error if decryption fails.
func DecryptAES(ciphertext, key []byte) ([]byte, error) {
	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Check if the ciphertext length is valid
	blockSize := block.BlockSize()
	if len(ciphertext) < blockSize || len(ciphertext)%blockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}

	// Extract IV from the ciphertext
	iv := ciphertext[:blockSize]
	ciphertext = ciphertext[blockSize:]

	// Decrypt data using CBC mode
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Remove PKCS#7 padding
	plaintext, err := unpadPKCS7(ciphertext)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// unpadPKCS7 removes PKCS#7 padding from the decrypted data
func unpadPKCS7(data []byte) ([]byte, error) {
	padding := int(data[len(data)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, errors.New("invalid padding")
	}

	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:len(data)-padding], nil
}

// EncryptRequestBody encrypts the request body with the AES symmetric key
// and prepends body with 8 bytes of size of the key in bigendian format
// and RSA encrypted symmetric key right after
// Parameters:
//   - data: The request body data to be encrypted.
//   - publicKey: The RSA public key used for encrypting the symmetric key.
//
// Returns:
//   - []byte: The encrypted payload.
//   - error: An error if encryption fails.
func EncryptRequestBody(data *bytes.Buffer, publicKey *rsa.PublicKey) (*bytes.Buffer, error) {
	// Generate a random symmetric key
	const keySize = 32
	symmetricKey := make([]byte, keySize) // AES-256 key
	_, err := rand.Read(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random symmetric key: %w", err)
	}

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, symmetricKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt symmetric key with RSA-OAEP: %w", err)
	}

	encryptedData, err := EncryptAES(data.Bytes(), symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data with AES: %w", err)
	}

	keyLen := make([]byte, encKeyLen)
	binary.BigEndian.PutUint64(keyLen, uint64(len(encryptedKey)))
	data.Reset()
	data.Write(keyLen)
	data.Write(encryptedKey)
	data.Write(encryptedData)

	return data, nil
}

// DecryptRequestBody extracts symmetric AES key, decrypts it with RSA key and decrypts body
// Parameters:
//   - body: The encrypted request body.
//   - key: The RSA private key used for decrypting the symmetric key.
//
// Returns:
//   - []byte: The decrypted request body data.
//   - error: An error if decryption fails.
func DecryptRequestBody(body *bytes.Buffer, key *rsa.PrivateKey) (*bytes.Buffer, error) {
	bodyBytes := body.Bytes()
	if len(bodyBytes) < encKeyLen {
		return nil, fmt.Errorf("failed to decrypt body. Body is too small: %d", len(bodyBytes))
	}

	keySize := binary.BigEndian.Uint64(bodyBytes[:encKeyLen])

	if keySize >= uint64(len(bodyBytes)) {
		return nil, fmt.Errorf("invalid symmetric key size: %d", keySize)
	}

	encryptedKey := bodyBytes[encKeyLen : encKeyLen+keySize]
	encryptedData := bodyBytes[encKeyLen+keySize:]

	symmetricKey, err := DecryptOAEP(key, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt symmetric key %w", err)
	}

	decryptedData, err := DecryptAES(encryptedData, symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt symmetric key %w", err)
	}

	return bytes.NewBuffer(decryptedData), nil
}

func setupTLS(caPath, pKeyPath, certPath string) (myCert *tls.Certificate, caCert []byte, err error) {
	cert, err := tls.LoadX509KeyPair(certPath, pKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load server certificate. CertPath: %v, KeyPath: %v: %w", certPath, pKeyPath, err)
	}

	caCert, err = os.ReadFile(caPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load CA certificate. CertPath: %v: %w", caPath, err)
	}

	return &cert, caCert, nil
}

func SetupClientTLS(caPath, pKeyPath, certPath string) (credentials.TransportCredentials, error) {
	myCert, caCert, err := setupTLS(caPath, pKeyPath, certPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*myCert},
		RootCAs:      caCertPool,
	})

	return creds, nil
}

func SetupServerTLS(caPath, pKeyPath, certPath string) (credentials.TransportCredentials, error) {
	myCert, caCert, err := setupTLS(caPath, pKeyPath, certPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*myCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})

	return creds, nil
}
