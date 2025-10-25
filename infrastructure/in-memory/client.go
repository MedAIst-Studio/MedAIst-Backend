package inmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
)

type InMemoryConfig struct {
	OAuthStateTTL   time.Duration
	OAuthSessionTTL time.Duration
}

type InMemoryClient struct {
	mu     sync.RWMutex
	data   map[string][]byte
	expiry map[string]time.Time
	config InMemoryConfig
}

func NewInMemoryClient(config InMemoryConfig) *InMemoryClient {
	client := &InMemoryClient{
		data:   make(map[string][]byte),
		expiry: make(map[string]time.Time),
		config: config,
	}

	// Background goroutine to clean expired keys
	go client.cleanupExpired()

	return client
}

func (c *InMemoryClient) cleanupExpired() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, expireTime := range c.expiry {
			if now.After(expireTime) {
				delete(c.data, key)
				delete(c.expiry, key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *InMemoryClient) set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
	if ttl > 0 {
		c.expiry[key] = time.Now().Add(ttl)
	}
}

func (c *InMemoryClient) get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	if expireTime, hasExpiry := c.expiry[key]; hasExpiry {
		if time.Now().After(expireTime) {
			return nil, fmt.Errorf("key expired")
		}
	}

	return value, nil
}

func (c *InMemoryClient) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	delete(c.expiry, key)
}

func (c *InMemoryClient) SaveOAuthState(ctx context.Context, state string) error {
	c.set(state, []byte("true"), c.config.OAuthStateTTL)
	return nil
}

func (c *InMemoryClient) ValidateOAuthState(ctx context.Context, state string) (bool, error) {
	value, err := c.get(state)
	if err != nil {
		return false, nil // State not found or expired
	}
	return string(value) == "true", nil
}

func (c *InMemoryClient) DeleteOAuthState(ctx context.Context, state string) error {
	c.delete(state)
	return nil
}

func (c *InMemoryClient) SaveOAuthSession(ctx context.Context, session *auth.OAuthSession) error {
	data := map[string]interface{}{
		"access_token":  session.AccessToken,
		"refresh_token": session.RefreshToken,
		"session_key":   session.SessionKey,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal OAuth session: %w", err)
	}

	c.set(session.SessionKey, jsonData, c.config.OAuthSessionTTL)
	return nil
}

func (c *InMemoryClient) GetOAuthSession(ctx context.Context, sessionKey string) (*auth.OAuthSession, error) {
	data, err := c.get(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	session := &auth.OAuthSession{
		SessionKey:   sessionData["session_key"].(string),
		AccessToken:  sessionData["access_token"].(string),
		RefreshToken: sessionData["refresh_token"].(string),
	}

	return session, nil
}

func (c *InMemoryClient) DeleteOAuthSession(ctx context.Context, sessionKey string) error {
	c.delete(sessionKey)
	return nil
}
