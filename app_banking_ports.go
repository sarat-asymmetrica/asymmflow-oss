package main

import (
	"fmt"
	"log"
	"strings"
)

type appBankingAuthPort struct {
	app *App
}

func (p appBankingAuthPort) CurrentUserID() string {
	if p.app == nil {
		return "system"
	}
	return p.app.getCurrentUserID()
}

func (p appBankingAuthPort) HasPermission(action string) bool {
	if p.app == nil {
		return false
	}
	return p.app.requirePermission(action) == nil
}

type appBankingAuditPort struct {
	app *App
}

func (p appBankingAuditPort) LogAction(entityType, entityID, action, detail, userID string) error {
	if strings.TrimSpace(userID) == "" {
		userID = "system"
	}

	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(
			userID,
			action,
			entityType,
			entityID,
			0,
			"BHD",
			true,
			map[string]any{"detail": detail},
		)
		return nil
	}

	log.Printf("AUDIT banking action: entity=%s id=%s action=%s user=%s detail=%s", entityType, entityID, action, userID, detail)
	return nil
}

func (p appBankingAuditPort) LogFinancialTransaction(userID, action, entityType, entityID string, amount float64, currency string, details map[string]any) error {
	if strings.TrimSpace(userID) == "" {
		userID = "system"
	}
	if strings.TrimSpace(currency) == "" {
		currency = "BHD"
	}

	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(userID, action, entityType, entityID, amount, currency, true, details)
		return nil
	}

	log.Printf("AUDIT banking financial action: entity=%s id=%s action=%s user=%s amount=%.3f %s details=%v", entityType, entityID, action, userID, amount, currency, details)
	return nil
}

type appBankingDivisionPort struct {
	app *App
}

func (p appBankingDivisionPort) CurrentDivision() string {
	return activeOverlay.DefaultDivision()
}

func (p appBankingDivisionPort) ResolveDivision(entityID string) (string, error) {
	if p.app == nil {
		return activeOverlay.DefaultDivision(), fmt.Errorf("application not initialized")
	}

	if strings.TrimSpace(entityID) == "" {
		return p.CurrentDivision(), nil
	}

	return p.app.resolveBankAccountDivision(entityID), nil
}

type appBankingDeleteApprovalPort struct {
	app *App
}

func (p appBankingDeleteApprovalPort) GuardDeleteOrRequest(permission, entityType, entityID, entityLabel string) (bool, error) {
	if p.app == nil {
		return false, fmt.Errorf("application not initialized")
	}
	return p.app.guardDeleteOrRequest(permission, entityType, entityID, entityLabel)
}
