package storage

// state.go — Redis-backed ephemeral store for OAuth state tokens.
//
// OAuth flows need to carry state across two HTTP requests:
//   1. Connect()  → generate state, store payload, redirect to platform
//   2. Callback() → receive state, retrieve payload, exchange code
//
// The payload includes workspaceID and, for PKCE flows (Twitter), the
// code_verifier so it can be sent in the token exchange.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const oauthStateTTL = 10 * time.Minute

// OAuthStatePayload is what we store per state token.
type OAuthStatePayload struct {
	WorkspaceID  string `json:"workspace_id"`
	CodeVerifier string `json:"code_verifier,omitempty"` // PKCE only (Twitter)
	Platform     string `json:"platform"`
}

type StateStore struct {
	rdb *redis.Client
}

func NewStateStore(redisURL string) (*StateStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("state store: parse redis URL: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("state store: redis ping: %w", err)
	}
	return &StateStore{rdb: rdb}, nil
}

func (s *StateStore) Save(ctx context.Context, state string, payload OAuthStatePayload) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, oauthStateKey(state), b, oauthStateTTL).Err()
}

func (s *StateStore) Get(ctx context.Context, state string) (*OAuthStatePayload, error) {
	b, err := s.rdb.Get(ctx, oauthStateKey(state)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("state expired or not found")
		}
		return nil, err
	}
	var payload OAuthStatePayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// Pop retrieves and immediately deletes the state (one-time use).
func (s *StateStore) Pop(ctx context.Context, state string) (*OAuthStatePayload, error) {
	payload, err := s.Get(ctx, state)
	if err != nil {
		return nil, err
	}
	_ = s.rdb.Del(ctx, oauthStateKey(state))
	return payload, nil
}

func oauthStateKey(state string) string {
	return "oauth:state:" + state
}
