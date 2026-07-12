package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestClassifyIntent_WorkQueryUsesEmployeeReference(t *testing.T) {
	intent := classifyIntent("How many tasks are assigned to Jamie today?")

	require.Equal(t, "work", intent.Domain)
	require.Equal(t, "Jamie", intent.PersonName)
	require.Equal(t, "employee", intent.ReferenceKind)
}

func TestResolveBestEntityReference_SupplierAndEmployee(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierCode: "SUP-EH-001",
		SupplierName: "Rhine Instruments AG",
		Country:      "Switzerland",
		SupplierType: "Manufacturer",
	}
	require.NoError(t, app.db.Create(&supplier).Error)

	employee := Employee{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EmployeeCode:     "EMP-009",
		FullName:         "Jamie Wong",
		PreferredName:    "Jamie",
		Department:       "Operations",
		JobTitle:         "Coordinator",
		EmploymentStatus: "active",
		IsActive:         true,
	}
	require.NoError(t, app.db.Create(&employee).Error)

	supplierResolution := app.resolveBestEntityReference(Intent{
		Domain:        "supplier",
		EntityName:    "Rhine Instruments",
		ReferenceKind: "supplier",
	})
	require.NotNil(t, supplierResolution)
	require.Equal(t, "supplier", supplierResolution.EntityType)
	require.Equal(t, supplier.SupplierName, supplierResolution.DisplayName)

	employeeResolution := app.resolveBestEntityReference(Intent{
		Domain:        "work",
		PersonName:    "Jamie",
		ReferenceKind: "employee",
	})
	require.NotNil(t, employeeResolution)
	require.Equal(t, "employee", employeeResolution.EntityType)
	require.Equal(t, "Jamie", employeeResolution.DisplayName)
}

func TestTryGroundedWorkFastPath_TaskSummaryByEmployee(t *testing.T) {
	app := setupTestApp(t)
	migrateWorkflowRegressionTables(t, app)

	employee := Employee{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EmployeeCode:     "EMP-002",
		FullName:         "Jamie Wong",
		PreferredName:    "Jamie",
		Department:       "Operations",
		JobTitle:         "Coordinator",
		EmploymentStatus: "active",
		IsActive:         true,
	}
	require.NoError(t, app.db.Create(&employee).Error)

	overdue := time.Now().AddDate(0, 0, -2)
	require.NoError(t, app.db.Create(&TaskItem{
		Base:               Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Review calibration backlog",
		Status:             "open",
		Priority:           "high",
		TaskType:           "service",
		AssigneeEmployeeID: &employee.ID,
		CreatorEmployeeID:  employee.ID,
		DueDate:            &overdue,
	}).Error)
	require.NoError(t, app.db.Create(&TaskItem{
		Base:               Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Confirm bank statement mapping",
		Status:             "blocked",
		Priority:           "medium",
		TaskType:           "finance",
		AssigneeEmployeeID: &employee.ID,
		CreatorEmployeeID:  employee.ID,
	}).Error)
	require.NoError(t, app.db.Create(&TaskItem{
		Base:               Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Close spare parts quote",
		Status:             "completed",
		Priority:           "low",
		TaskType:           "sales",
		AssigneeEmployeeID: &employee.ID,
		CreatorEmployeeID:  employee.ID,
	}).Error)

	require.NoError(t, app.db.Create(&Notification{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EmployeeID:       employee.ID,
		NotificationType: "task",
		Title:            "Task reassigned",
		Message:          "A finance task was assigned to Jamie.",
		Status:           "unread",
		SourceType:       "task",
		SourceID:         uuid.New().String(),
	}).Error)

	reply, handled := app.tryGroundedWorkFastPath(
		Intent{
			RawQuery:      "How many tasks are assigned to Jamie right now?",
			Domain:        "work",
			PersonName:    "Jamie",
			ReferenceKind: "employee",
		},
		"How many tasks are assigned to Jamie right now?",
	)

	require.True(t, handled)
	require.Contains(t, reply, "Jamie")
	require.Contains(t, reply, "2 active task(s)")
	require.Contains(t, reply, "1 blocked task(s)")
	require.Contains(t, reply, "1 overdue task(s)")
	require.Contains(t, reply, "1 unread notification(s)")
	require.Contains(t, reply, "Review calibration backlog")
}

func TestTryGroundedWorkFastPath_SelfTaskQueryUsesCurrentEmployee(t *testing.T) {
	app := setupHybridFeatureTestApp(t)
	current := seedCurrentAdminContext(t, app, "Jordan")

	dueToday := time.Now()
	require.NoError(t, app.db.Create(&TaskItem{
		Base:               Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Title:              "Call AquaPure about delivery note",
		Status:             "open",
		Priority:           "high",
		TaskType:           "operations",
		AssigneeEmployeeID: &current.EmployeeID,
		CreatorEmployeeID:  current.EmployeeID,
		DueDate:            &dueToday,
	}).Error)

	reply, handled := app.tryGroundedWorkFastPath(
		classifyIntent("what are the new task for me today?"),
		"what are the new task for me today?",
	)

	require.True(t, handled)
	require.Contains(t, reply, "Jordan")
	require.Contains(t, reply, "Call AquaPure about delivery note")
}

func TestTryGroundedSupplierFastPath_PaymentAndPurchaseHistory(t *testing.T) {
	app, _ := seedButlerBusinessHarnessFixture(t)

	paymentReply, handled := app.tryGroundedSupplierFastPath(
		classifyIntent("Tell me about Rhine Instruments payment history"),
		"Tell me about Rhine Instruments payment history",
	)
	require.True(t, handled)
	require.Contains(t, paymentReply, "supplier payment history for Rhine Instruments AG")
	require.Contains(t, paymentReply, "Recorded supplier invoices")
	require.Contains(t, paymentReply, "Recent payments")

	purchaseReply, handled := app.tryGroundedSupplierFastPath(
		classifyIntent("What did we buy from Rhine Instruments?"),
		"What did we buy from Rhine Instruments?",
	)
	require.True(t, handled)
	require.Contains(t, purchaseReply, "what we have bought from Rhine Instruments AG")
	require.Contains(t, purchaseReply, "Recent supplier invoices")
	require.Contains(t, purchaseReply, "Recent purchased line items")
}

func TestTryGroundedCustomerFastPath_UsesCustomerNotes(t *testing.T) {
	app, _ := seedButlerBusinessHarnessFixture(t)

	reply, handled := app.tryGroundedCustomerFastPath(
		classifyIntent("What notes do we have for National Petroleum Co.?"),
		"What notes do we have for National Petroleum Co.?",
	)
	require.True(t, handled)
	require.Contains(t, reply, "customer note(s) for National Petroleum Co.")
	require.Contains(t, reply, "Recent notes")
	require.Contains(t, reply, "split pricing")
}

func TestTryGroundedTaskCreationFastPath_CreatesTaskAndNotification(t *testing.T) {
	app := setupHybridFeatureTestApp(t)
	seedCurrentAdminContext(t, app, "Jordan")

	jamie := createEmployeeForTest(t, app, "Jamie Wong", "Jamie", "Sales", "Coordinator")

	reply, handled := app.tryGroundedTaskCreationFastPath(
		classifyIntent("can you create a task for Jamie to follow up on National Petroleum Co. lead. make sure he gets the notifications"),
		"can you create a task for Jamie to follow up on National Petroleum Co. lead. make sure he gets the notifications",
	)

	require.True(t, handled)
	require.Contains(t, reply, "Created the task")
	require.Contains(t, reply, "Jamie")
	require.Contains(t, reply, "assignment notification")

	var tasks []TaskItem
	require.NoError(t, app.db.Where("assignee_employee_id = ?", jamie.ID).Find(&tasks).Error)
	require.Len(t, tasks, 1)
	require.Equal(t, "Follow up on National Petroleum Co. lead", tasks[0].Title)

	var notificationCount int64
	require.NoError(t, app.db.Model(&Notification{}).
		Where("employee_id = ? AND source_type = ? AND source_id = ? AND notification_type = ?", jamie.ID, "task", tasks[0].ID, "task").
		Count(&notificationCount).Error)
	require.EqualValues(t, 1, notificationCount)
}
