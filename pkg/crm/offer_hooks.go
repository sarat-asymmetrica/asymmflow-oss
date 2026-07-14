package crm

import (
	"strings"

	"gorm.io/gorm"
)

// ComposeOfferDeliveryTerms composes an offer's default delivery-terms string
// from its division, when the offer is created without an explicit value.
// It is a dependency-injection seam: package main wires it at startup to a
// closure backed by the active overlay (pkg/crm must not import pkg/overlay).
// When nil (e.g. in unit tests that don't wire it), the hook leaves DeliveryTerms
// empty and the GORM column default applies — byte-identical to legacy behavior.
var ComposeOfferDeliveryTerms func(division string) string

// BeforeCreate mints the ID (via the embedded Base) and, when DeliveryTerms is
// empty, fills it with the per-division composed value so a non-default-division
// offer gets that division's delivery-terms string instead of the hardcoded
// default-division column default (Wave 12.5 B3).
func (o *Offer) BeforeCreate(tx *gorm.DB) error {
	if err := o.Base.BeforeCreate(tx); err != nil {
		return err
	}
	if strings.TrimSpace(o.DeliveryTerms) == "" && ComposeOfferDeliveryTerms != nil {
		o.DeliveryTerms = ComposeOfferDeliveryTerms(o.Division)
	}
	return nil
}
