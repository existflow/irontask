package sync

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	keySize          = 32 // AES-256
	nonceSize        = 12 // GCM standard nonce size
	saltSize         = 16
	pbkdf2Iterations = 100000
)

// Crypto handles encryption/decryption
type Crypto struct {
	key []byte
}

// NewCrypto creates a crypto instance with derived key from password
func NewCrypto(password string, salt []byte) *Crypto {
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keySize, sha256.New)
	return &Crypto{key: key}
}

// GenerateSalt generates a random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt encrypts data using AES-256-GCM
func (c *Crypto) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends nonce + ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts data using AES-256-GCM
func (c *Crypto) Decrypt(encrypted string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: invalid key or corrupted data")
	}

	return plaintext, nil
}

// DeriveKey derives a key from password for display (first 8 chars of base64)
func DeriveKeyDisplay(password string, salt []byte) string {
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keySize, sha256.New)
	return base64.StdEncoding.EncodeToString(key)[:16]
}
