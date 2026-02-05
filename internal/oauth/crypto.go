// Package oauth provides secure OAuth 2.0 integration for external services.
//
// Security measures:
// - AES-256-GCM encryption for token storage
// - Encryption key from system keyring or secure random generation
// - Never logs tokens or secrets
// - Secure file permissions (0600)
// - CSRF protection with cryptographic state
package oauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// keySize is 32 bytes for AES-256
	keySize = 32
	// saltSize for PBKDF2
	saltSize = 32
	// nonceSize for AES-GCM
	nonceSize = 12
	// iterations for PBKDF2
	iterations = 100000
)

// DeriveKey derives an encryption key from a password/secret using PBKDF2.
// This is used when we have a master secret but need a proper AES key.
func DeriveKey(secret []byte, salt []byte) []byte {
	return pbkdf2.Key(secret, salt, iterations, keySize, sha256.New)
}

// GenerateRandomBytes generates cryptographically secure random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateState generates a cryptographically secure state parameter for OAuth.
// This prevents CSRF attacks.
func GenerateState() (string, error) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns nonce + ciphertext, all base64 encoded.
func Encrypt(plaintext []byte, key []byte) (string, error) {
	if len(key) != keySize {
		return "", fmt.Errorf("invalid key size: expected %d, got %d", keySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateRandomBytes(nonceSize)
	if err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt.
func Decrypt(encrypted string, key []byte) ([]byte, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", keySize, len(key))
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// GetOrCreateEncryptionKey gets or creates a master encryption key.
// The key is stored in a secure file with 0600 permissions.
// Returns the derived key and salt.
func GetOrCreateEncryptionKey(baseDir string) (key []byte, salt []byte, err error) {
	keyFile := filepath.Join(baseDir, ".oauth_key")
	saltFile := filepath.Join(baseDir, ".oauth_salt")

	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if key and salt files exist
	keyData, keyErr := os.ReadFile(keyFile)
	saltData, saltErr := os.ReadFile(saltFile)

	if keyErr != nil || saltErr != nil {
		// Generate new key and salt
		keyData, err = GenerateRandomBytes(keySize)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate key: %w", err)
		}

		saltData, err = GenerateRandomBytes(saltSize)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
		}

		// Write key file with secure permissions
		if err := os.WriteFile(keyFile, keyData, 0600); err != nil {
			return nil, nil, fmt.Errorf("failed to write key file: %w", err)
		}

		// Write salt file
		if err := os.WriteFile(saltFile, saltData, 0600); err != nil {
			return nil, nil, fmt.Errorf("failed to write salt file: %w", err)
		}
	}

	// Derive the actual encryption key
	derivedKey := DeriveKey(keyData, saltData)
	return derivedKey, saltData, nil
}

// SecureWipe attempts to overwrite memory containing sensitive data.
// Note: This is best-effort as Go's GC may have copied the data.
func SecureWipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
