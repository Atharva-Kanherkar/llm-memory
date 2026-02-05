package oauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Token represents an OAuth token with metadata.
// SECURITY: This struct should NEVER be logged.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope,omitempty"`
}

// IsExpired returns true if the token has expired.
func (t *Token) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	// Consider expired if within 5 minutes of expiry
	return time.Now().Add(5 * time.Minute).After(t.Expiry)
}

// Valid returns true if the token is present and not expired.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && !t.IsExpired()
}

// TokenStore manages encrypted storage of OAuth tokens.
// SECURITY: All tokens are encrypted at rest using AES-256-GCM.
type TokenStore struct {
	baseDir string
	key     []byte
	mu      sync.RWMutex
}

// NewTokenStore creates a new token store with encryption.
func NewTokenStore(baseDir string) (*TokenStore, error) {
	key, _, err := GetOrCreateEncryptionKey(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	return &TokenStore{
		baseDir: baseDir,
		key:     key,
	}, nil
}

// tokenFilePath returns the path for a provider's token file.
func (s *TokenStore) tokenFilePath(provider string) string {
	// Use a hash-like filename to avoid leaking provider names in the filesystem
	return filepath.Join(s.baseDir, fmt.Sprintf(".token_%s.enc", provider))
}

// SaveToken encrypts and saves a token for a provider.
// SECURITY: Token is encrypted before writing to disk.
func (s *TokenStore) SaveToken(provider string, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize token to JSON
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	// Encrypt the token data
	encrypted, err := Encrypt(data, s.key)
	if err != nil {
		SecureWipe(data) // Clear sensitive data from memory
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Clear the plaintext from memory
	SecureWipe(data)

	// Write to file with secure permissions
	path := s.tokenFilePath(provider)
	if err := os.WriteFile(path, []byte(encrypted), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken loads and decrypts a token for a provider.
// SECURITY: Token is decrypted only when needed.
func (s *TokenStore) LoadToken(provider string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.tokenFilePath(provider)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // No token stored
	}

	// Read encrypted data
	encrypted, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Decrypt the token data
	data, err := Decrypt(string(encrypted), s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Deserialize token
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		SecureWipe(data)
		return nil, fmt.Errorf("failed to deserialize token: %w", err)
	}

	// Clear decrypted data from memory
	SecureWipe(data)

	return &token, nil
}

// DeleteToken removes a stored token.
func (s *TokenStore) DeleteToken(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.tokenFilePath(provider)

	// Overwrite with random data before deleting (paranoid secure delete)
	if _, err := os.Stat(path); err == nil {
		randomData, _ := GenerateRandomBytes(1024)
		os.WriteFile(path, randomData, 0600)
	}

	return os.Remove(path)
}

// HasToken checks if a token exists for a provider.
func (s *TokenStore) HasToken(provider string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.tokenFilePath(provider)
	_, err := os.Stat(path)
	return err == nil
}

// ListProviders returns a list of providers with stored tokens.
func (s *TokenStore) ListProviders() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var providers []string
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 11 && name[:7] == ".token_" && name[len(name)-4:] == ".enc" {
			provider := name[7 : len(name)-4]
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// Close securely wipes the encryption key from memory.
func (s *TokenStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	SecureWipe(s.key)
}
