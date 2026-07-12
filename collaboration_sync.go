package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

var collaborativeSyncTables = []string{
	"employees",
	"employee_access_links",
	"projects",
	"project_members",
	"task_items",
	"task_comments",
	"task_activity",
	"notifications",
	"notification_receipts",
	"delete_approval_requests",
	"opportunity_edit_conflicts",
}

const collaborativeSyncCursorSetting = "collaboration_sync_last_at"

func (a *App) ensureCollaborativeRemoteSchema() error {
	if a.dbManager == nil || a.dbManager.Remote() == nil {
		return fmt.Errorf("remote collaboration sync unavailable")
	}

	remote := a.dbManager.Remote()
	if remote.Migrator().HasTable("task_items") && remote.Migrator().HasTable("employees") && remote.Migrator().HasTable("opportunity_edit_conflicts") {
		return nil
	}

	return remote.AutoMigrate(
		&Employee{},
		&EmployeeDocument{}, // Wave 9.8 B4
		&EmployeeAccessLink{},
		&Project{},
		&ProjectMember{},
		&Notification{},
		&NotificationReceipt{},
		&DeleteApprovalRequest{},
		&TaskItem{},
		&TaskComment{},
		&TaskActivity{},
		&OpportunityEditConflict{},
	)
}

func (a *App) getCollaborativeSyncCursor() time.Time {
	if a.db == nil {
		return time.Time{}
	}

	var setting Setting
	if err := a.db.Where("key = ?", collaborativeSyncCursorSetting).First(&setting).Error; err != nil {
		return time.Time{}
	}

	cursor, err := time.Parse(time.RFC3339, setting.Value)
	if err != nil {
		return time.Time{}
	}
	return cursor
}

func (a *App) setCollaborativeSyncCursor(t time.Time) {
	if a.db == nil || t.IsZero() {
		return
	}
	a.saveSetting(collaborativeSyncCursorSetting, t.UTC().Format(time.RFC3339))
}

func (a *App) collectDueCollaborativeOperations() []CollaborativePendingOperation {
	if a.db == nil {
		return nil
	}

	now := time.Now()
	var ops []CollaborativePendingOperation
	_ = a.db.
		Where("status IN ?", []string{"pending", "failed"}).
		Where("next_attempt_at IS NULL OR next_attempt_at <= ?", now).
		Order("created_at ASC").
		Find(&ops).Error
	return ops
}

func (a *App) markCollaborativeOperationsSynced(ops []CollaborativePendingOperation) {
	if a.db == nil || len(ops) == 0 {
		return
	}

	ids := make([]string, 0, len(ops))
	for _, op := range ops {
		ids = append(ids, op.ID)
	}

	now := time.Now()
	_ = a.db.Model(&CollaborativePendingOperation{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":          "synced",
			"error_message":   "",
			"last_attempt_at": &now,
			"next_attempt_at": nil,
		}).Error
}

func (a *App) markCollaborativeOperationsFailed(ops []CollaborativePendingOperation, syncErr error) {
	if a.db == nil || len(ops) == 0 {
		return
	}

	errMsg := "collaboration sync failed"
	if syncErr != nil {
		errMsg = syncErr.Error()
	}

	now := time.Now()
	for _, op := range ops {
		backoffMinutes := op.Attempts + 1
		if backoffMinutes > 15 {
			backoffMinutes = 15
		}
		nextAttempt := now.Add(time.Duration(backoffMinutes) * time.Minute)
		_ = a.db.Model(&CollaborativePendingOperation{}).
			Where("id = ?", op.ID).
			Updates(map[string]any{
				"status":          "failed",
				"attempts":        op.Attempts + 1,
				"error_message":   errMsg,
				"last_attempt_at": &now,
				"next_attempt_at": &nextAttempt,
			}).Error
	}
}

func (a *App) emitCollaborativePullEvents(pulledByTable map[string]int) {
	taskChanges := pulledByTable["task_items"] + pulledByTable["task_comments"] + pulledByTable["task_activity"]
	notificationChanges := pulledByTable["notifications"] + pulledByTable["notification_receipts"]
	projectChanges := pulledByTable["projects"] + pulledByTable["project_members"]
	employeeChanges := pulledByTable["employees"] + pulledByTable["employee_access_links"]

	if employeeChanges > 0 {
		a.emitCollaborationEvent("employees:updated", map[string]any{
			"records": employeeChanges,
			"source":  "remote_sync",
		})
	}
	if projectChanges > 0 {
		a.emitCollaborationEvent("projects:updated", map[string]any{
			"records": projectChanges,
			"source":  "remote_sync",
		})
	}
	if taskChanges > 0 {
		a.emitCollaborationEvent("tasks:updated", map[string]any{
			"records": taskChanges,
			"action":  "remote_sync",
		})
	}
	if notificationChanges > 0 {
		a.emitCollaborationEvent("notifications:updated", map[string]any{
			"records": notificationChanges,
			"source":  "remote_sync",
		})
	}
}

func (a *App) runCollaborativeSync(trigger string) error {
	if a.db == nil || a.dbManager == nil || !a.dbManager.IsSyncEnabled() {
		return nil
	}

	a.collaborationSyncMu.Lock()
	defer a.collaborationSyncMu.Unlock()

	if !a.dbManager.IsOnline() {
		if err := a.dbManager.ConnectRemote(); err != nil {
			return err
		}
	}
	if !a.dbManager.CheckConnectivity() {
		return fmt.Errorf("collaboration sync remote is offline")
	}
	if err := a.ensureCollaborativeRemoteSchema(); err != nil {
		return err
	}

	since := a.getCollaborativeSyncCursor()
	now := time.Now()
	dueOps := a.collectDueCollaborativeOperations()
	pulledByTable := make(map[string]int)
	var firstErr error

	for _, table := range collaborativeSyncTables {
		if count, err := a.dbManager.syncTable(table, since, "push"); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("push %s: %w", table, err)
			}
		} else if count > 0 {
			log.Printf("collaboration sync push (%s): %s -> %d", trigger, table, count)
		}
	}

	for _, table := range collaborativeSyncTables {
		count, err := a.dbManager.syncTable(table, since, "pull")
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("pull %s: %w", table, err)
			}
			continue
		}
		if count > 0 {
			pulledByTable[table] = count
			log.Printf("collaboration sync pull (%s): %s -> %d", trigger, table, count)
		}
	}

	if firstErr != nil {
		a.markCollaborativeOperationsFailed(dueOps, firstErr)
		return firstErr
	}

	a.setCollaborativeSyncCursor(now)
	a.markCollaborativeOperationsSynced(dueOps)
	a.emitCollaborativePullEvents(pulledByTable)
	return nil
}

func (a *App) queueCollaborativeSync(trigger string) {
	if a.dbManager == nil || !a.dbManager.IsSyncEnabled() {
		return
	}

	go func() {
		if err := a.runCollaborativeSync(trigger); err != nil && !strings.Contains(strings.ToLower(err.Error()), "offline") {
			log.Printf("collaboration sync (%s) failed: %v", trigger, err)
		}
	}()
}

func (a *App) StartCollaborativeSyncLoop(interval time.Duration) {
	if a.dbManager == nil || !a.dbManager.IsSyncEnabled() {
		return
	}
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	a.collaborationSyncInitMu.Lock()
	if a.collaborationSyncStop != nil {
		a.collaborationSyncInitMu.Unlock()
		return
	}
	a.collaborationSyncStop = make(chan struct{})
	a.collaborationSyncStopOnce = sync.Once{}
	a.collaborationSyncInitMu.Unlock()

	a.collaborationSyncWG.Add(1)
	go func() {
		defer a.collaborationSyncWG.Done()

		select {
		case <-time.After(5 * time.Second):
		case <-a.collaborationSyncStop:
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			if err := a.runCollaborativeSync("background_poll"); err != nil && !strings.Contains(strings.ToLower(err.Error()), "offline") {
				log.Printf("collaboration background sync failed: %v", err)
			}

			select {
			case <-ticker.C:
				continue
			case <-a.collaborationSyncStop:
				return
			}
		}
	}()
}

func (a *App) StopCollaborativeSyncLoop() {
	a.collaborationSyncInitMu.Lock()
	stopChan := a.collaborationSyncStop
	a.collaborationSyncInitMu.Unlock()

	if stopChan != nil {
		a.collaborationSyncStopOnce.Do(func() {
			close(stopChan)
		})
	}
	a.collaborationSyncWG.Wait()

	a.collaborationSyncInitMu.Lock()
	a.collaborationSyncStop = nil
	a.collaborationSyncInitMu.Unlock()
}

func (a *App) RefreshCollaborativeWorkspace() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := a.GetLicenseRole(); err == "" {
		if _, ctxErr := a.GetCurrentEmployeeContext(); ctxErr != nil {
			return ctxErr
		}
	}

	return a.runCollaborativeSync("manual_refresh")
}
