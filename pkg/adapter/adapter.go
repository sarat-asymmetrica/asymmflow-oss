// Package adapter contains shared helpers for bridging GORM domain models and
// generated Cap'n Proto schema messages.
package adapter

import (
	"fmt"
	"strconv"
	"time"

	shareddomain "ph_holdings_app/pkg/domain"
	commonproto "ph_holdings_app/schemas/go/common"

	capnp "capnproto.org/go/capnp/v3"
	"gorm.io/gorm"
)

// TimeToText converts time.Time to RFC3339 text for Proto fields.
func TimeToText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// TimePtrToText converts *time.Time to RFC3339 text for Proto fields.
func TimePtrToText(t *time.Time) string {
	if t == nil {
		return ""
	}
	return TimeToText(*t)
}

// TextToTime converts RFC3339 text from Proto fields to time.Time.
func TextToTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// TextToTimePtr converts RFC3339 text to *time.Time.
func TextToTimePtr(s string) *time.Time {
	t := TextToTime(s)
	if t.IsZero() {
		return nil
	}
	return &t
}

// UintToText converts uint IDs to string for Proto Text fields.
func UintToText(id uint) string {
	return fmt.Sprintf("%d", id)
}

// TextToUint converts Proto Text IDs back to uint.
func TextToUint(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 64)
	return uint(id)
}

// DeletedAtToText converts gorm.DeletedAt to RFC3339 text.
func DeletedAtToText(deletedAt gorm.DeletedAt) string {
	if !deletedAt.Valid {
		return ""
	}
	return TimeToText(deletedAt.Time)
}

// TextToDeletedAt converts RFC3339 text to gorm.DeletedAt.
func TextToDeletedAt(s string) gorm.DeletedAt {
	t := TextToTime(s)
	if t.IsZero() {
		return gorm.DeletedAt{}
	}
	return gorm.DeletedAt{Time: t, Valid: true}
}

// IntToInt64 keeps generated Proto calls consistent for int fields.
func IntToInt64(v int) int64 {
	return int64(v)
}

// Int64ToInt converts Proto Int64 values back to int.
func Int64ToInt(v int64) int {
	return int(v)
}

// BaseToProto converts the shared persisted Base into the common Proto Base.
// Version has no Proto counterpart in Wave 9 schemas and is intentionally omitted.
func BaseToProto(seg *capnp.Segment, base shareddomain.Base) (commonproto.Base, error) {
	p, err := commonproto.NewBase(seg)
	if err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetId(base.ID); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetCreatedAt(TimeToText(base.CreatedAt)); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetUpdatedAt(TimeToText(base.UpdatedAt)); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetDeletedAt(DeletedAtToText(base.DeletedAt)); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetCreatedBy(base.CreatedBy); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetUpdatedBy(""); err != nil {
		return commonproto.Base{}, err
	}
	p.SetStatus(commonproto.RecordStatus_active)
	p.SetSyncState(commonproto.SyncState_synced)
	return p, nil
}

// BaseFromProto converts common Proto Base data back to the shared persisted Base.
func BaseFromProto(p commonproto.Base) (shareddomain.Base, error) {
	id, err := p.Id()
	if err != nil {
		return shareddomain.Base{}, err
	}
	createdAt, err := p.CreatedAt()
	if err != nil {
		return shareddomain.Base{}, err
	}
	updatedAt, err := p.UpdatedAt()
	if err != nil {
		return shareddomain.Base{}, err
	}
	deletedAt, err := p.DeletedAt()
	if err != nil {
		return shareddomain.Base{}, err
	}
	createdBy, err := p.CreatedBy()
	if err != nil {
		return shareddomain.Base{}, err
	}
	return shareddomain.Base{
		ID:        id,
		CreatedAt: TextToTime(createdAt),
		UpdatedAt: TextToTime(updatedAt),
		DeletedAt: TextToDeletedAt(deletedAt),
		CreatedBy: createdBy,
	}, nil
}
