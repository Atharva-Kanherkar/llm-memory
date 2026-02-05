package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Provider represents an OAuth provider configuration.
// SECURITY: ClientSecret is loaded from environment, never logged.
type Provider struct {
	Name         string
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string // SECURITY: Never log this
	Scopes       []string
	RedirectURI  string
}

// Client handles OAuth flows for a provider.
type Client struct {
	provider   *Provider
	tokenStore *TokenStore
	httpClient *http.Client

	// State management for CSRF protection
	stateMu     sync.Mutex
	pendingAuth map[string]chan *Token // state -> result channel
}

// NewClient creates a new OAuth client.
func NewClient(provider *Provider, tokenStore *TokenStore) *Client {
	return &Client{
		provider:    provider,
		tokenStore:  tokenStore,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		pendingAuth: make(map[string]chan *Token),
	}
}

// GetAuthURL returns the URL to redirect the user to for authorization.
// SECURITY: Uses cryptographic state for CSRF protection.
func (c *Client) GetAuthURL() (string, string, error) {
	state, err := GenerateState()
	if err != nil {
		return "", "", err
	}

	params := url.Values{}
	params.Set("client_id", c.provider.ClientID)
	params.Set("redirect_uri", c.provider.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(c.provider.Scopes, " "))
	params.Set("state", state)
	params.Set("access_type", "offline") // Request refresh token
	params.Set("prompt", "consent")      // Force consent to get refresh token

	return c.provider.AuthURL + "?" + params.Encode(), state, nil
}

// ExchangeCode exchanges an authorization code for tokens.
// SECURITY: Tokens are encrypted before storage.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.provider.RedirectURI)
	data.Set("client_id", c.provider.ClientID)
	data.Set("client_secret", c.provider.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", c.provider.TokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}

	if tokenResp.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// Save token securely
	if err := c.tokenStore.SaveToken(c.provider.Name, token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}

// RefreshToken refreshes an expired access token.
// SECURITY: Uses refresh token, then securely stores new tokens.
func (c *Client) RefreshToken(ctx context.Context) (*Token, error) {
	currentToken, err := c.tokenStore.LoadToken(c.provider.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	if currentToken == nil || currentToken.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", currentToken.RefreshToken)
	data.Set("client_id", c.provider.ClientID)
	data.Set("client_secret", c.provider.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", c.provider.TokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
	}

	// Keep old refresh token if new one not provided
	if token.RefreshToken == "" {
		token.RefreshToken = currentToken.RefreshToken
	}

	if tokenResp.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// Save updated token
	if err := c.tokenStore.SaveToken(c.provider.Name, token); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return token, nil
}

// GetValidToken returns a valid access token, refreshing if necessary.
func (c *Client) GetValidToken(ctx context.Context) (*Token, error) {
	token, err := c.tokenStore.LoadToken(c.provider.Name)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	if !token.IsExpired() {
		return token, nil
	}

	// Token expired, try to refresh
	if token.RefreshToken != "" {
		return c.RefreshToken(ctx)
	}

	return nil, fmt.Errorf("token expired and no refresh token available")
}

// StartLocalAuthServer starts a local server to handle the OAuth callback.
// SECURITY: Only listens on localhost, validates state parameter.
func (c *Client) StartLocalAuthServer(ctx context.Context, state string) (*Token, error) {
	// Create a channel to receive the result
	resultChan := make(chan *Token, 1)
	errChan := make(chan error, 1)

	c.stateMu.Lock()
	c.pendingAuth[state] = resultChan
	c.stateMu.Unlock()

	defer func() {
		c.stateMu.Lock()
		delete(c.pendingAuth, state)
		c.stateMu.Unlock()
	}()

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Update redirect URI with actual port
	c.provider.RedirectURI = fmt.Sprintf("http://localhost:%d/callback", port)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only handle /callback
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			// SECURITY: Validate state parameter to prevent CSRF
			receivedState := r.URL.Query().Get("state")
			if receivedState != state {
				http.Error(w, "Invalid state parameter", http.StatusBadRequest)
				errChan <- fmt.Errorf("CSRF detected: invalid state parameter")
				return
			}

			// Check for error response
			if errMsg := r.URL.Query().Get("error"); errMsg != "" {
				errDesc := r.URL.Query().Get("error_description")
				http.Error(w, "Authorization failed: "+errDesc, http.StatusBadRequest)
				errChan <- fmt.Errorf("authorization failed: %s - %s", errMsg, errDesc)
				return
			}

			// Get authorization code
			code := r.URL.Query().Get("code")
			if code == "" {
				http.Error(w, "Missing authorization code", http.StatusBadRequest)
				errChan <- fmt.Errorf("missing authorization code")
				return
			}

			// Exchange code for token
			token, err := c.ExchangeCode(ctx, code)
			if err != nil {
				http.Error(w, "Token exchange failed", http.StatusInternalServerError)
				errChan <- err
				return
			}

			// Success response
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `
				<html>
				<head><title>Authorization Successful</title></head>
				<body>
					<h1>Authorization Successful!</h1>
					<p>You can close this window and return to Mnemosyne.</p>
					<script>window.close();</script>
				</body>
				</html>
			`)

			resultChan <- token
		}),
	}

	// Start server in goroutine
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for result or timeout
	select {
	case token := <-resultChan:
		server.Shutdown(ctx)
		return token, nil
	case err := <-errChan:
		server.Shutdown(ctx)
		return nil, err
	case <-ctx.Done():
		server.Shutdown(ctx)
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		server.Shutdown(ctx)
		return nil, fmt.Errorf("authorization timeout")
	}
}

// IsAuthenticated checks if valid tokens exist for this provider.
func (c *Client) IsAuthenticated() bool {
	token, err := c.tokenStore.LoadToken(c.provider.Name)
	if err != nil || token == nil {
		return false
	}
	// Check if we have a refresh token (can get new access tokens)
	return token.RefreshToken != "" || !token.IsExpired()
}

// Logout removes stored tokens for this provider.
func (c *Client) Logout() error {
	return c.tokenStore.DeleteToken(c.provider.Name)
}
