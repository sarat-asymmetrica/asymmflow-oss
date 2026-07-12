// Package license contains the concrete license service implementation.
package license

import "gorm.io/gorm"

type Handlers[LicenseKeyModel any, ActivationResultModel any, ValidationResultModel any] struct {
	GenerateLicenseKey                    func(role, notes, createdBy string) (string, error)
	GenerateBatchLicenseKeys              func(role string, count int, notes, createdBy string) ([]string, error)
	ActivateLicense                       func(key string) (ActivationResultModel, error)
	ValidateLicense                       func() (ValidationResultModel, error)
	GetLicenseRole                        func() string
	HasLicensePermission                  func(permission string) bool
	ListLicenseKeys                       func() ([]LicenseKeyModel, error)
	UpdateLicenseDisplayName              func(key, displayName string) (LicenseKeyModel, error)
	RevokeLicense                         func(key string) error
	EnsureLicenseTableExists              func() error
	SeedLicenseKeys                       func() error
	ApplyDeploymentLicenseActivationFlush func() error
	SeedEmployeeKeys                      func() error
	CheckFirstInstall                     func() bool
	NeedsLicenseActivation                func() (bool, error)
}

type Service[LicenseKeyModel any, ActivationResultModel any, ValidationResultModel any] struct {
	db       *gorm.DB
	handlers Handlers[LicenseKeyModel, ActivationResultModel, ValidationResultModel]
}

func New[LicenseKeyModel any, ActivationResultModel any, ValidationResultModel any](db *gorm.DB, handlers Handlers[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel] {
	return &Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]{db: db, handlers: handlers}
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) GenerateLicenseKey(role, notes, createdBy string) (string, error) {
	return s.handlers.GenerateLicenseKey(role, notes, createdBy)
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) GenerateBatchLicenseKeys(role string, count int, notes, createdBy string) ([]string, error) {
	return s.handlers.GenerateBatchLicenseKeys(role, count, notes, createdBy)
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) ActivateLicense(key string) (ActivationResultModel, error) {
	return s.handlers.ActivateLicense(key)
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) ValidateLicense() (ValidationResultModel, error) {
	return s.handlers.ValidateLicense()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) GetLicenseRole() string {
	return s.handlers.GetLicenseRole()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) HasLicensePermission(permission string) bool {
	return s.handlers.HasLicensePermission(permission)
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) ListLicenseKeys() ([]LicenseKeyModel, error) {
	return s.handlers.ListLicenseKeys()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) UpdateLicenseDisplayName(key, displayName string) (LicenseKeyModel, error) {
	return s.handlers.UpdateLicenseDisplayName(key, displayName)
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) RevokeLicense(key string) error {
	return s.handlers.RevokeLicense(key)
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) EnsureLicenseTableExists() error {
	return s.handlers.EnsureLicenseTableExists()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) SeedLicenseKeys() error {
	return s.handlers.SeedLicenseKeys()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) ApplyDeploymentLicenseActivationFlush() error {
	return s.handlers.ApplyDeploymentLicenseActivationFlush()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) SeedEmployeeKeys() error {
	return s.handlers.SeedEmployeeKeys()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) CheckFirstInstall() bool {
	return s.handlers.CheckFirstInstall()
}

func (s *Service[LicenseKeyModel, ActivationResultModel, ValidationResultModel]) NeedsLicenseActivation() (bool, error) {
	return s.handlers.NeedsLicenseActivation()
}
