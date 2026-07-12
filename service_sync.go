package main

import (
	"log"
	"time"
)

// SyncServiceBinding exposes domain-specific Wails bindings by delegating to App.
type SyncServiceBinding struct {
	app *App
}

func NewSyncServiceBinding(app *App) *SyncServiceBinding {
	return &SyncServiceBinding{app: app}
}

// --- collaboration_service.go ---

func (s *SyncServiceBinding) AddCollaborativeProjectMember(projectID, employeeID, role string, allocationPercent float64) (ProjectMember, error) {
	return s.app.AddCollaborativeProjectMember(projectID, employeeID, role, allocationPercent)
}

func (s *SyncServiceBinding) GetProjectTaskCounts() (map[string]int, error) {
	return s.app.GetProjectTaskCounts()
}

func (s *SyncServiceBinding) AddCollaborativeTaskComment(taskID, body string) (TaskComment, error) {
	return s.app.AddCollaborativeTaskComment(taskID, body)
}

func (s *SyncServiceBinding) ArchiveCollaborativeProject(projectID, reason string) (Project, error) {
	return s.app.ArchiveCollaborativeProject(projectID, reason)
}

func (s *SyncServiceBinding) CreateCollaborativeProject(project Project) (Project, error) {
	return s.app.CreateCollaborativeProject(project)
}

func (s *SyncServiceBinding) CreateCollaborativeTask(task TaskItem) (TaskItem, error) {
	return s.app.CreateCollaborativeTask(task)
}

func (s *SyncServiceBinding) CreateEmployeeAccessLink(link EmployeeAccessLink) (EmployeeAccessLink, error) {
	return s.app.CreateEmployeeAccessLink(link)
}

func (s *SyncServiceBinding) CreateEmployeeProfile(employee Employee) (Employee, error) {
	return s.app.CreateEmployeeProfile(employee)
}

func (s *SyncServiceBinding) DeleteCollaborativeProject(projectID, reason string) error {
	return s.app.DeleteCollaborativeProject(projectID, reason)
}

func (s *SyncServiceBinding) DeleteCollaborativeTask(taskID string) error {
	return s.app.DeleteCollaborativeTask(taskID)
}

func (s *SyncServiceBinding) ShelveCollaborativeProject(projectID, reason string) (Project, error) {
	return s.app.ShelveCollaborativeProject(projectID, reason)
}

func (s *SyncServiceBinding) UpdateCollaborativeProject(projectID string, updates map[string]any) (Project, error) {
	return s.app.UpdateCollaborativeProject(projectID, updates)
}

func (s *SyncServiceBinding) EnsureCollaborativeFoundation() error {
	return s.app.EnsureCollaborativeFoundation()
}

func (s *SyncServiceBinding) GetCollaborativeTask(taskID string) (*TaskItem, error) {
	return s.app.GetCollaborativeTask(taskID)
}

func (s *SyncServiceBinding) GetCurrentEmployeeContext() (CurrentEmployeeContext, error) {
	return s.app.GetCurrentEmployeeContext()
}

func (s *SyncServiceBinding) GetUnreadNotificationsCount() (int, error) {
	return s.app.GetUnreadNotificationsCount()
}

func (s *SyncServiceBinding) ListCollaborativeProjectActivity(projectID string) ([]TaskActivity, error) {
	return s.app.ListCollaborativeProjectActivity(projectID)
}

func (s *SyncServiceBinding) ListCollaborativeProjectMembers(projectID string) ([]ProjectMember, error) {
	return s.app.ListCollaborativeProjectMembers(projectID)
}

func (s *SyncServiceBinding) ListCollaborativeProjectTasks(projectID string, includeCompleted bool) ([]TaskItem, error) {
	return s.app.ListCollaborativeProjectTasks(projectID, includeCompleted)
}

func (s *SyncServiceBinding) ListCollaborativeProjects(activeOnly bool) ([]Project, error) {
	return s.app.ListCollaborativeProjects(activeOnly)
}

func (s *SyncServiceBinding) ListCollaborativeTaskActivity(taskID string) ([]TaskActivity, error) {
	return s.app.ListCollaborativeTaskActivity(taskID)
}

func (s *SyncServiceBinding) ListCollaborativeTaskComments(taskID string) ([]TaskComment, error) {
	return s.app.ListCollaborativeTaskComments(taskID)
}

func (s *SyncServiceBinding) ListCollaborativeTasksForEmployee(employeeID string, includeCompleted bool) ([]TaskItem, error) {
	return s.app.ListCollaborativeTasksForEmployee(employeeID, includeCompleted)
}

func (s *SyncServiceBinding) ListCollaborativeTeamTasks(includeCompleted bool) ([]TaskItem, error) {
	return s.app.ListCollaborativeTeamTasks(includeCompleted)
}

func (s *SyncServiceBinding) ListEmployeeAccessLinks() ([]EmployeeAccessLink, error) {
	return s.app.ListEmployeeAccessLinks()
}

func (s *SyncServiceBinding) ListEmployeeContributionSummaries() ([]EmployeeContributionSummary, error) {
	return s.app.ListEmployeeContributionSummaries()
}

func (s *SyncServiceBinding) ListEmployeeProfiles(activeOnly bool) ([]Employee, error) {
	return s.app.ListEmployeeProfiles(activeOnly)
}

func (s *SyncServiceBinding) ListEmployeeProjectAssignments(employeeID string) ([]ProjectMember, error) {
	return s.app.ListEmployeeProjectAssignments(employeeID)
}

func (s *SyncServiceBinding) ListMyCollaborativeTasks(includeCompleted bool) ([]TaskItem, error) {
	return s.app.ListMyCollaborativeTasks(includeCompleted)
}

func (s *SyncServiceBinding) ListNotificationFeed(limit int, unreadOnly bool) ([]Notification, error) {
	return s.app.ListNotificationFeed(limit, unreadOnly)
}

func (s *SyncServiceBinding) MarkNotificationAsRead(notificationID string) error {
	return s.app.MarkNotificationAsRead(notificationID)
}

func (s *SyncServiceBinding) ReassignCollaborativeTask(taskID, assigneeEmployeeID string) error {
	return s.app.ReassignCollaborativeTask(taskID, assigneeEmployeeID)
}

func (s *SyncServiceBinding) ReassignEmployeeLicenseAccess(employeeID, licenseKey string, syncDisplayName bool) (EmployeeAccessLink, error) {
	return s.app.ReassignEmployeeLicenseAccess(employeeID, licenseKey, syncDisplayName)
}

func (s *SyncServiceBinding) ReassignEmployeeManager(employeeID, managerEmployeeID string) (Employee, error) {
	return s.app.ReassignEmployeeManager(employeeID, managerEmployeeID)
}

func (s *SyncServiceBinding) SetEmployeeEmploymentState(employeeID string, isActive bool, employmentStatus string) (Employee, error) {
	return s.app.SetEmployeeEmploymentState(employeeID, isActive, employmentStatus)
}

func (s *SyncServiceBinding) UpdateCollaborativeTask(task TaskItem) (TaskItem, error) {
	return s.app.UpdateCollaborativeTask(task)
}

func (s *SyncServiceBinding) UpdateCollaborativeTaskDueDate(taskID, dueDateISO string) error {
	return s.app.UpdateCollaborativeTaskDueDate(taskID, dueDateISO)
}

func (s *SyncServiceBinding) UpdateCollaborativeTaskStatus(taskID, status, note string) error {
	return s.app.UpdateCollaborativeTaskStatus(taskID, status, note)
}

func (s *SyncServiceBinding) UpdateEmployeeProfile(employee Employee) (Employee, error) {
	return s.app.UpdateEmployeeProfile(employee)
}

// --- collaboration_sync.go ---

func (s *SyncServiceBinding) RefreshCollaborativeWorkspace() error {
	return s.app.RefreshCollaborativeWorkspace()
}

func (s *SyncServiceBinding) StartCollaborativeSyncLoop(interval time.Duration) {
	// Wave 8 P0: gate the frontend surface only. The *App method is shared with startup
	// (app.go), which runs before any session — so the gate lives here, not on *App.
	if err := s.app.requirePermission("settings:update"); err != nil {
		log.Printf("collaboration sync loop start denied: %v", err)
		return
	}
	s.app.StartCollaborativeSyncLoop(interval)
}

func (s *SyncServiceBinding) StopCollaborativeSyncLoop() {
	if err := s.app.requirePermission("settings:update"); err != nil {
		log.Printf("collaboration sync loop stop denied: %v", err)
		return
	}
	s.app.StopCollaborativeSyncLoop()
}

// --- db_manager.go ---

func (s *SyncServiceBinding) GetDBSyncStatus() map[string]any {
	return s.app.GetDBSyncStatus()
}

func (s *SyncServiceBinding) GetFirstRunSyncStatus() map[string]any {
	return s.app.GetFirstRunSyncStatus()
}

func (s *SyncServiceBinding) GetSyncHealth() SyncHealth {
	return s.app.GetSyncHealth()
}

func (s *SyncServiceBinding) InitDBManager() {
	s.app.InitDBManager()
}

func (s *SyncServiceBinding) PerformFirstRunSync() (map[string]any, error) {
	return s.app.PerformFirstRunSync()
}

func (s *SyncServiceBinding) TestSupabaseConnection(host, port, user, password, dbname, sslmode string) (bool, error) {
	return s.app.TestSupabaseConnection(host, port, user, password, dbname, sslmode)
}

func (s *SyncServiceBinding) TriggerManualSync() (map[string]any, error) {
	return s.app.TriggerManualSync()
}

// --- db_sync_service.go ---

func (s *SyncServiceBinding) FirstRunSyncWithProgress() DBSyncResult {
	return s.app.FirstRunSyncWithProgress()
}

func (s *SyncServiceBinding) GetDBSyncSettings() DBSyncSettings {
	return s.app.GetDBSyncSettings()
}

func (s *SyncServiceBinding) StartBackgroundDBSync() {
	s.app.StartBackgroundDBSync()
}

func (s *SyncServiceBinding) StopBackgroundDBSync() {
	s.app.StopBackgroundDBSync()
}

func (s *SyncServiceBinding) SyncNowWithProgress() DBSyncResult {
	return s.app.SyncNowWithProgress()
}

func (s *SyncServiceBinding) UpdateDBSyncSettings(autoEnabled bool, frequencyMin int) error {
	return s.app.UpdateDBSyncSettings(autoEnabled, frequencyMin)
}

// --- sync_service_impl.go ---

func (s *SyncServiceBinding) PullChanges(since time.Time) error {
	return s.app.PullChanges(since)
}

func (s *SyncServiceBinding) PushChanges() error {
	return s.app.PushChanges()
}

func (s *SyncServiceBinding) StartPeriodicSync(interval time.Duration) {
	s.app.StartPeriodicSync(interval)
}

func (s *SyncServiceBinding) StopPeriodicSync() {
	s.app.StopPeriodicSync()
}
