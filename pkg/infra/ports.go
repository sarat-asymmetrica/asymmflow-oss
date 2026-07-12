// Package infra defines shared infrastructure ports.
package infra

import (
	"context"
	"time"
)

type Database interface {
	Exec(ctx context.Context, query string, args ...any) error
	Query(ctx context.Context, query string, args ...any) ([]map[string]any, error)
	Transaction(ctx context.Context, fn func(Database) error) error
	Ping(ctx context.Context) error
	Close() error
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

type JobQueue interface {
	Enqueue(ctx context.Context, job Job) (Job, error)
	Get(ctx context.Context, jobID string) (Job, error)
	UpdateProgress(ctx context.Context, jobID string, progress int) error
	Cancel(ctx context.Context, jobID string) error
	List(ctx context.Context, status string, limit int) ([]Job, error)
}

type AuthManager interface {
	Authenticate(ctx context.Context, username, password string) (UserSession, error)
	ValidateSession(ctx context.Context, token string) (User, error)
	RequirePermission(ctx context.Context, permission string) error
	RefreshSession(ctx context.Context, refreshToken string) (UserSession, error)
	Logout(ctx context.Context, token string) error
}

type CryptoService interface {
	EncryptField(plaintext string) (string, error)
	DecryptField(ciphertext string) (string, error)
	HashDocument(parts ...string) (string, error)
	RotateKey(ctx context.Context) error
}
