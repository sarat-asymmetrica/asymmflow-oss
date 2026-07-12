// Package sync defines the synchronization domain ports.
package sync

import (
	"context"
	"time"
)

type SyncEngine interface {
	Start(ctx context.Context, interval time.Duration) error
	Stop(ctx context.Context) error
	Push(ctx context.Context, table string) error
	Pull(ctx context.Context, table string) error
	SyncNow(ctx context.Context) error
	Health(ctx context.Context) (map[string]any, error)
}

type CloudStorage interface {
	Upload(ctx context.Context, localPath, remotePath string) error
	Download(ctx context.Context, remotePath, localPath string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, remotePath string) error
}

type CollaborationService interface {
	ListPendingOperations(ctx context.Context, employeeID string) ([]map[string]any, error)
	RecordOperation(ctx context.Context, operation map[string]any) error
	ApplyOperation(ctx context.Context, operationID string) error
	AcknowledgeNotification(ctx context.Context, notificationID, employeeID string) error
}
