// Package fulfillment contains the delivery and serial lifecycle services.
//
// The generic Handlers service below is the historical trampoline idiom
// (closures back into the host) and covers delivery notes only; the serial
// lifecycle moved inward for real in Wave 5 — see serials.go.
package fulfillment

import "gorm.io/gorm"

type Handlers[DeliveryNoteModel any] struct {
	CreateDeliveryNote   func(dn DeliveryNoteModel) (DeliveryNoteModel, error)
	GetDeliveryNotes     func() ([]DeliveryNoteModel, error)
	GetDeliveryNoteByID  func(id string) (DeliveryNoteModel, error)
	UpdateDeliveryNote   func(dn DeliveryNoteModel) (DeliveryNoteModel, error)
	DispatchDeliveryNote func(id, driverName, vehicleNumber string) error
	// ConfirmDeliveryNote returns (warning, error): warning is a non-fatal,
	// human-readable message when the DN confirmed but a downstream
	// order-progression step failed; empty string means fully clean.
	ConfirmDeliveryNote func(id, signedBy string) (string, error)
}

type Service[DeliveryNoteModel any] struct {
	db       *gorm.DB
	handlers Handlers[DeliveryNoteModel]
}

func New[DeliveryNoteModel any](db *gorm.DB, handlers Handlers[DeliveryNoteModel]) *Service[DeliveryNoteModel] {
	return &Service[DeliveryNoteModel]{db: db, handlers: handlers}
}

func (s *Service[DeliveryNoteModel]) CreateDeliveryNote(dn DeliveryNoteModel) (DeliveryNoteModel, error) {
	return s.handlers.CreateDeliveryNote(dn)
}

func (s *Service[DeliveryNoteModel]) GetDeliveryNotes() ([]DeliveryNoteModel, error) {
	return s.handlers.GetDeliveryNotes()
}

func (s *Service[DeliveryNoteModel]) GetDeliveryNoteByID(id string) (DeliveryNoteModel, error) {
	return s.handlers.GetDeliveryNoteByID(id)
}

func (s *Service[DeliveryNoteModel]) UpdateDeliveryNote(dn DeliveryNoteModel) (DeliveryNoteModel, error) {
	return s.handlers.UpdateDeliveryNote(dn)
}

func (s *Service[DeliveryNoteModel]) DispatchDeliveryNote(id, driverName, vehicleNumber string) error {
	return s.handlers.DispatchDeliveryNote(id, driverName, vehicleNumber)
}

func (s *Service[DeliveryNoteModel]) ConfirmDeliveryNote(id, signedBy string) (string, error) {
	return s.handlers.ConfirmDeliveryNote(id, signedBy)
}
