package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// PH convergence Row F (PH SPOC #9, a4dfeaf/223af59): credit-limit override is
// management-only, requires a reason, rides the kernel approval seam, and
// writes a CREDIT_LIMIT_OVERRIDE audit row. Everyone else keeps the hard block.
func TestCreateInvoiceFromOrderWithCreditOverride(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&AuditLog{}))

	customer := CustomerMaster{CustomerID: "CUST-CO1", CustomerCode: "CO1", BusinessName: "Atlas Traders", CreditLimitBHD: 100}
	require.NoError(t, a.db.Create(&customer).Error)

	newOrder := func(num string) Order {
		order := Order{
			OrderNumber: num, CustomerID: customer.CustomerID, CustomerName: customer.BusinessName,
			Status: "Confirmed", GrandTotalBHD: 500, TotalValueBHD: 500,
			Items: []OrderItem{{Description: "Flow transmitter", Quantity: 1, UnitPrice: 500, TotalPrice: 500}},
		}
		require.NoError(t, a.db.Create(&order).Error)
		return order
	}

	asRole := func(id, role string) {
		a.currentUserID = id
		a.currentUser = &User{Base: Base{ID: id}, Username: id, RoleName: role,
			Role: Role{Name: role, DisplayName: role, Permissions: `["*"]`}}
	}

	// Sales: hard block stands, and the override door is closed.
	asRole("sales-user", "sales")
	order1 := newOrder("ORD-CO-1")
	_, err := a.CreateInvoiceWithOptions(order1.ID, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "credit limit exceeded")
	_, err = a.CreateInvoiceFromOrderWithCreditOverride(order1.ID, "customer promised payment")
	require.Error(t, err)
	require.Contains(t, err.Error(), "CREDIT_OVERRIDE_DENIED")

	// Manager without a reason: refused.
	asRole("ops-manager", "manager")
	_, err = a.CreateInvoiceFromOrderWithCreditOverride(order1.ID, "  ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "CREDIT_OVERRIDE_REASON_REQUIRED")

	// Manager with a reason: invoice created despite the limit, audit row written.
	inv, err := a.CreateInvoiceFromOrderWithCreditOverride(order1.ID, "strategic account, payment scheduled")
	require.NoError(t, err)
	require.Greater(t, inv.GrandTotalBHD, 100.0)

	var auditCount int64
	a.db.Table("audit_logs").Where("action = ?", "CREDIT_LIMIT_OVERRIDE").Count(&auditCount)
	require.EqualValues(t, 1, auditCount, "override must leave a CREDIT_LIMIT_OVERRIDE audit row")
}
