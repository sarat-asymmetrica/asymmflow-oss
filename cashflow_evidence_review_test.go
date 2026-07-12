package main

import (
	"fmt"
	"testing"
)

func TestNormalizeCashflowProposalReviewStatus(t *testing.T) {
	cases := map[string]string{
		"":               CashflowProposalStatusPending,
		"pending":        CashflowProposalStatusPending,
		"APPROVE":        CashflowProposalStatusApproved,
		"approved":       CashflowProposalStatusApproved,
		"needs input":    CashflowProposalStatusNeedsInput,
		"needs_input":    CashflowProposalStatusNeedsInput,
		"reject":         CashflowProposalStatusRejected,
		"rejected":       CashflowProposalStatusRejected,
		"superseded":     CashflowProposalStatusSuperseded,
		"execute action": "",
	}

	for input, want := range cases {
		if got := normalizeCashflowProposalReviewStatus(input); got != want {
			t.Fatalf("normalizeCashflowProposalReviewStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestValidCashflowProposalTransitions(t *testing.T) {
	valid := [][2]string{
		{CashflowProposalStatusPending, CashflowProposalStatusApproved},
		{CashflowProposalStatusPending, CashflowProposalStatusRejected},
		{CashflowProposalStatusPending, CashflowProposalStatusNeedsInput},
		{CashflowProposalStatusPending, CashflowProposalStatusSuperseded},
		{CashflowProposalStatusNeedsInput, CashflowProposalStatusPending},
		{CashflowProposalStatusNeedsInput, CashflowProposalStatusApproved},
		{CashflowProposalStatusNeedsInput, CashflowProposalStatusRejected},
		{CashflowProposalStatusNeedsInput, CashflowProposalStatusSuperseded},
		{CashflowProposalStatusApproved, CashflowProposalStatusSuperseded},
		{CashflowProposalStatusRejected, CashflowProposalStatusPending},
		{CashflowProposalStatusRejected, CashflowProposalStatusSuperseded},
	}

	for _, pair := range valid {
		from, to := pair[0], pair[1]
		t.Run(fmt.Sprintf("%s_to_%s", from, to), func(t *testing.T) {
			if !validCashflowProposalTransition(from, to) {
				t.Errorf("expected transition %q -> %q to be valid, got false", from, to)
			}
		})
	}
}

func TestInvalidCashflowProposalTransitions(t *testing.T) {
	invalid := [][2]string{
		{CashflowProposalStatusSuperseded, CashflowProposalStatusPending},
		{CashflowProposalStatusSuperseded, CashflowProposalStatusApproved},
		{CashflowProposalStatusSuperseded, CashflowProposalStatusRejected},
		{CashflowProposalStatusSuperseded, CashflowProposalStatusNeedsInput},
		{CashflowProposalStatusApproved, CashflowProposalStatusPending},
		{CashflowProposalStatusApproved, CashflowProposalStatusRejected},
		{CashflowProposalStatusApproved, CashflowProposalStatusNeedsInput},
		{CashflowProposalStatusRejected, CashflowProposalStatusApproved},
		{CashflowProposalStatusRejected, CashflowProposalStatusNeedsInput},
		{"", CashflowProposalStatusApproved},
		{"unknown", CashflowProposalStatusApproved},
	}

	for _, pair := range invalid {
		from, to := pair[0], pair[1]
		t.Run(fmt.Sprintf("%s_to_%s", from, to), func(t *testing.T) {
			if validCashflowProposalTransition(from, to) {
				t.Errorf("expected transition %q -> %q to be invalid, got true", from, to)
			}
		})
	}
}

func TestSelfTransitionIsInvalid(t *testing.T) {
	statuses := []string{
		CashflowProposalStatusPending,
		CashflowProposalStatusApproved,
		CashflowProposalStatusRejected,
		CashflowProposalStatusNeedsInput,
		CashflowProposalStatusSuperseded,
	}

	for _, s := range statuses {
		t.Run(fmt.Sprintf("%s_to_%s", s, s), func(t *testing.T) {
			if validCashflowProposalTransition(s, s) {
				t.Errorf("expected self-transition %q -> %q to be invalid, got true", s, s)
			}
		})
	}
}
