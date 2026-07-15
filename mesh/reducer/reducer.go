// Package reducer is the pure, deterministic inventory apply-reducer for the
// Sovereign Mesh determinism spike (Mission A of FABLE_CAMPAIGN_SOVEREIGN_MESH.md).
//
// It is deliberately dependency-free and free of build tags so it can be:
//   - unit-tested on the host (normal GOOS) — see reducer_test.go
//   - compiled to wasip1 and driven from the JS/Autobase host — see ../cmd/reducer
//
// THE SPIKE'S CLAIM (what Mission A must prove): the function that guards a
// business invariant is byte-identical whether it runs on one node or a
// thousand. Autobase linearizes every writer's append-only log into ONE order
// every peer agrees on, then replays it here. So this reducer must be a PURE,
// DETERMINISTIC function: same ops (as a set) → byte-identical state on every
// peer, forever.
//
// The four determinism landmines (§4 invariant 1 of the campaign) and how this
// file avoids each:
//  1. map iteration order is randomized  -> we NEVER range a map for output;
//     every SKU/actor traversal goes through a sorted key slice.
//  2. time.Now()/rand are forbidden      -> no clock, no randomness anywhere;
//     an op's timestamp is DATA carried in Op.TS, used only as a tie-breaker.
//  3. floats drift                        -> all quantities are int64 minor
//     units (whole instrument counts here); no float appears.
//  4. unstable linearization              -> Apply canonically re-sorts the ops
//     by (Seq, Actor, SKU, TS) so a permuted delivery order still converges.
//
// Domain: INVENTORY (chosen per Mission A as the invariant-bound case). The
// floor invariant is "stock must never go below zero" — a concurrent-offline
// oversell is deterministically REJECTED on every peer (skipped + recorded),
// which is the CORRECT behaviour for money/stock, not a silent bad number.
package reducer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
)

// Op is one append-only inventory movement from some writer's Hypercore log.
// A negative Delta is a sale/consumption; a positive Delta is a receipt/restock.
// Nothing here is wall-clock derived: TS is the event's own recorded time,
// present only so the canonical ordering is a total order (never a live clock).
type Op struct {
	Seq   int64  `json:"seq"`   // per-writer monotonic sequence (Hypercore index)
	Actor string `json:"actor"` // writer/device id (Ed25519 key id in the real mesh)
	SKU   string `json:"sku"`   // stock-keeping unit / instrument token
	Delta int64  `json:"delta"` // signed movement in whole units
	TS    int64  `json:"ts"`    // event-data timestamp (tie-breaker ONLY, never a clock)
}

// Rejection records an op the floor invariant refused, so the UX can surface a
// typed Unconfirmed/Rejected state (Mechanism 2) instead of a silent −1.
type Rejection struct {
	Seq    int64  `json:"seq"`
	Actor  string `json:"actor"`
	SKU    string `json:"sku"`
	Delta  int64  `json:"delta"`
	Reason string `json:"reason"`
}

// State is the converged, deterministic result of replaying an op set.
// Digest is a sha256 over the canonical (sorted) state — two peers agree iff
// their digests match, which is the cheap byte-identical convergence check
// Mission A's 3-peer test asserts.
type State struct {
	Stock     map[string]int64 `json:"stock"`     // sku -> quantity (never < 0)
	Rejected  []Rejection      `json:"rejected"`  // ops the floor invariant refused
	Applied   int              `json:"applied"`   // count of ops that landed
	Digest    string           `json:"digest"`    // sha256 of the canonical state
	OpsHashed int              `json:"opsHashed"` // total ops considered (applied+rejected)
}

// canonicalLess is the total order Autobase-style linearization must agree on.
// Sorting by (Seq, Actor, SKU, TS) makes replay independent of network delivery
// order: the same SET of ops always produces the same sequence, hence the same
// state, on every peer. TS is only the last tie-breaker and is event data.
func canonicalLess(a, b Op) bool {
	if a.Seq != b.Seq {
		return a.Seq < b.Seq
	}
	if a.Actor != b.Actor {
		return a.Actor < b.Actor
	}
	if a.SKU != b.SKU {
		return a.SKU < b.SKU
	}
	return a.TS < b.TS
}

// Apply replays ops through the floor invariant and returns the converged State.
// It is the whole spike: a pure function of its input, with no I/O, clock, or
// randomness. In the mesh this becomes the Autobase apply() reducer (compiled to
// wasip1); on the host it is an ordinary testable function.
func Apply(ops []Op) State {
	// 1. Canonicalize the order (landmine #4). Copy first — never mutate input.
	sorted := make([]Op, len(ops))
	copy(sorted, ops)
	sort.SliceStable(sorted, func(i, j int) bool { return canonicalLess(sorted[i], sorted[j]) })

	stock := make(map[string]int64)
	rejected := make([]Rejection, 0)
	applied := 0

	// 2. Fold. Integer-only math (landmine #3). Floor invariant enforced here.
	for _, op := range sorted {
		next := stock[op.SKU] + op.Delta
		if next < 0 {
			// Concurrent-offline oversell: reject deterministically on every peer.
			rejected = append(rejected, Rejection{
				Seq:    op.Seq,
				Actor:  op.Actor,
				SKU:    op.SKU,
				Delta:  op.Delta,
				Reason: "floor invariant: stock would fall below 0 (have " +
					strconv.FormatInt(stock[op.SKU], 10) + ", delta " +
					strconv.FormatInt(op.Delta, 10) + ")",
			})
			continue
		}
		stock[op.SKU] = next
		applied++
	}

	st := State{
		Stock:     stock,
		Rejected:  rejected,
		Applied:   applied,
		OpsHashed: len(sorted),
	}
	st.Digest = digest(st)
	return st
}

// digest computes a stable sha256 over the canonical state. It NEVER ranges a
// map directly for hashing (landmine #1): SKUs are hashed in sorted key order,
// rejections are already in canonical apply order.
func digest(st State) string {
	skus := make([]string, 0, len(st.Stock))
	for k := range st.Stock {
		skus = append(skus, k)
	}
	sort.Strings(skus)

	// Build a canonical, map-free projection and hash its JSON encoding.
	type kv struct {
		SKU string `json:"sku"`
		Qty int64  `json:"qty"`
	}
	proj := struct {
		Stock    []kv        `json:"stock"`
		Rejected []Rejection `json:"rejected"`
		Applied  int         `json:"applied"`
	}{
		Stock:    make([]kv, 0, len(skus)),
		Rejected: st.Rejected,
		Applied:  st.Applied,
	}
	for _, s := range skus {
		proj.Stock = append(proj.Stock, kv{SKU: s, Qty: st.Stock[s]})
	}
	b, _ := json.Marshal(proj)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
