// capability.go — Mission D: the Ed25519 grant-with-epochs capability layer,
// enforced INSIDE the reducer (kernel law), not in the host.
//
// Doctrine (campaign §Mission D): transport-auth ≠ capability-auth. A Holesail
// connection key is a static, non-revocable byte pipe; Autobase's writer set is
// the replication plane. NEITHER is the permission model. The permission model
// is here: every op is signed by its device's Ed25519 key, and a device may
// only mutate state while it holds a GRANT — issued by the mesh authority and
// tied to the current grant EPOCH. Revocation = the authority bumps the epoch
// and re-issues grants to the still-trusted; every other grant goes stale at
// the app layer even though the old pipe still opens and the old writer still
// replicates. The reducer folds this deterministically, so a revoked device's
// ops are rejected byte-identically on every honest peer.
//
// Why signature verification is allowed in a "pure" reducer: Ed25519 verify is
// deterministic math over op bytes — no clock, no randomness, no I/O. The four
// determinism landmines (reducer.go header) are all respected.
//
// Enforcement is opt-in per mesh via Config.AuthorityPub (the owner's root
// public key, distributed with the app config exactly like the Autobase
// bootstrap key). With no authority configured the reducer behaves exactly as
// Missions A+C shipped it — legacy goldens stay byte-stable (MESH-D12).
package reducer

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Config carries mesh-wide constants that are part of the fold's law but not
// of any single op. It is data (distributed at mesh genesis), never ambient.
type Config struct {
	// AuthorityPub is the hex Ed25519 public key of the mesh authority (the
	// owner's root key). Empty = capability enforcement OFF (legacy mode).
	AuthorityPub string `json:"authorityPub,omitempty"`
}

// GrantState is the converged capability of one device public key.
type GrantState struct {
	Role  string `json:"role"`
	Epoch int64  `json:"epoch"` // the grant epoch it was issued under
}

// signable builds the canonical byte payload an op's signature covers:
// a version tag plus every semantic field (all except Sig itself) as
// length-prefixed netstrings in FIXED order. Netstrings — not JSON — because
// JS and Go must produce byte-identical payloads and JSON gives neither stable
// key order nor identical escaping across the two runtimes.
// MIRROR: mesh/host/capability.mjs signable() must match this byte-for-byte.
//
// VERSIONING (MSG-D2): the payload version is selected by op.Kind. Room kinds
// (room.manifest / msg.*) sign "meshop.v2" = the v1 field list PLUS the room
// fields appended. Every legacy kind keeps the exact v1 bytes, so Mission A-D
// signatures, test vectors, and goldens are untouched. Room fields present on
// a NON-room op are therefore unsigned — and ignored: no legacy handler reads
// them, and the room fold only ever sees room kinds.
func signable(op Op) []byte {
	if isInviteKind(op.Kind) {
		return signableV3(op)
	}
	if isRoomKind(op.Kind) {
		return signableV2(op)
	}
	return signableV1(op)
}

// isRoomKind reports whether kind belongs to the Messenger room vocabulary.
func isRoomKind(kind string) bool {
	return kind == "room.manifest" || kind == "room.claim" || (len(kind) > 4 && kind[:4] == "msg.")
}

// isInviteKind reports whether kind belongs to the M2 invite vocabulary.
func isInviteKind(kind string) bool {
	return len(kind) > 7 && kind[:7] == "invite."
}

func signableV1(op Op) []byte {
	fields := []string{
		strconv.FormatInt(op.Seq, 10),
		op.Actor,
		strconv.FormatInt(op.TS, 10),
		op.Kind,
		op.SKU,
		strconv.FormatInt(op.Delta, 10),
		op.Customer,
		strconv.FormatInt(op.AmountMinor, 10),
		strconv.FormatInt(op.LimitMinor, 10),
		op.Currency,
		op.Subject,
		op.SubjectType,
		op.Decision,
		op.Reason,
		op.CorrelationID,
		op.ActorType,
		strconv.Itoa(op.Authority),
		op.PolicyID,
		op.Device,
		op.Role,
		strconv.FormatInt(op.Epoch, 10),
		op.DevicePub,
	}
	return netstrings("meshop.v1", fields)
}

// signableV2 = the v1 field list + the room fields, prefix "meshop.v2".
// Expectation/Assignee (MSG-D16) are appended at the END of the field list —
// the v2 field list GROWS, but every earlier field keeps its position, so
// this is the only re-golden this wave causes (room goldens only).
// PredecessorRoomKey (MSG-D20) is the field list's THIRD growth, appended
// after Assignee for the same reason.
// MIRROR: mesh/host/capability.mjs FIELDS_V2 must match byte-for-byte.
func signableV2(op Op) []byte {
	fields := []string{
		strconv.FormatInt(op.Seq, 10),
		op.Actor,
		strconv.FormatInt(op.TS, 10),
		op.Kind,
		op.SKU,
		strconv.FormatInt(op.Delta, 10),
		op.Customer,
		strconv.FormatInt(op.AmountMinor, 10),
		strconv.FormatInt(op.LimitMinor, 10),
		op.Currency,
		op.Subject,
		op.SubjectType,
		op.Decision,
		op.Reason,
		op.CorrelationID,
		op.ActorType,
		strconv.Itoa(op.Authority),
		op.PolicyID,
		op.Device,
		op.Role,
		strconv.FormatInt(op.Epoch, 10),
		op.DevicePub,
		// room fields (Messenger Wave 1)
		op.MsgID,
		op.Body,
		op.ReplyTo,
		op.Emoji,
		strconv.FormatBool(op.On),
		op.UpToActor,
		strconv.FormatInt(op.UpToSeq, 10),
		op.Title,
		op.AnchorType,
		op.AnchorID,
		strconv.FormatBool(op.Observers),
		op.Draft,
		op.Attachment,
		// expectation tags + claim/assign (MSG-D16, appended at the end)
		op.Expectation,
		op.Assignee,
		// predecessor room pointer (MSG-D20, appended at the end — third growth)
		op.PredecessorRoomKey,
	}
	return netstrings("meshop.v2", fields)
}

// signableV3 = the v2 field list + the invite fields, prefix "meshop.v3".
// Selected ONLY by invite.* kinds, so v1 AND v2 payloads (and their goldens)
// stay byte-stable (MSG-D11, same pattern as MSG-D2).
// MIRROR: mesh/host/capability.mjs FIELDS_V3 must match byte-for-byte.
func signableV3(op Op) []byte {
	fields := []string{
		strconv.FormatInt(op.Seq, 10),
		op.Actor,
		strconv.FormatInt(op.TS, 10),
		op.Kind,
		op.SKU,
		strconv.FormatInt(op.Delta, 10),
		op.Customer,
		strconv.FormatInt(op.AmountMinor, 10),
		strconv.FormatInt(op.LimitMinor, 10),
		op.Currency,
		op.Subject,
		op.SubjectType,
		op.Decision,
		op.Reason,
		op.CorrelationID,
		op.ActorType,
		strconv.Itoa(op.Authority),
		op.PolicyID,
		op.Device,
		op.Role,
		strconv.FormatInt(op.Epoch, 10),
		op.DevicePub,
		op.MsgID,
		op.Body,
		op.ReplyTo,
		op.Emoji,
		strconv.FormatBool(op.On),
		op.UpToActor,
		strconv.FormatInt(op.UpToSeq, 10),
		op.Title,
		op.AnchorType,
		op.AnchorID,
		strconv.FormatBool(op.Observers),
		op.Draft,
		op.Attachment,
		// expectation tags + claim/assign (MSG-D16, appended at the end of v2)
		op.Expectation,
		op.Assignee,
		// predecessor room pointer (MSG-D20, same relative slot as v2: right
		// before the invite fields, since v3 = "v2 + invite fields")
		op.PredecessorRoomKey,
		// invite fields (Mission M2)
		op.InviteID,
		op.InvitePub,
		op.InviteProof,
		strconv.FormatInt(op.ExpiresAt, 10),
		strconv.FormatInt(op.MaxUses, 10),
	}
	return netstrings("meshop.v3", fields)
}

// inviteProofPayload is the byte payload the INVITE key signs at redemption:
// possession of the invite secret, bound to the joining device's public key so
// a captured proof cannot admit any other device.
// MIRROR: mesh/host/capability.mjs inviteProofPayload() must match.
func inviteProofPayload(devicePubHex string) []byte {
	return append([]byte("meshinvite.v1:"), devicePubHex...)
}

// verifyInviteProof checks op.InviteProof (hex Ed25519) over
// sha256(inviteProofPayload(op.DevicePub)) with the offer's invite public key.
func verifyInviteProof(invitePubHex string, op Op) bool {
	pub, err := hex.DecodeString(invitePubHex)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := hex.DecodeString(op.InviteProof)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	digest := sha256.Sum256(inviteProofPayload(op.DevicePub))
	return ed25519.Verify(ed25519.PublicKey(pub), digest[:], sig)
}

func netstrings(prefix string, fields []string) []byte {
	buf := []byte(prefix)
	for _, f := range fields {
		buf = append(buf, strconv.Itoa(len(f))...)
		buf = append(buf, ':')
		buf = append(buf, f...)
		buf = append(buf, ',')
	}
	return buf
}

// verifySig checks op.Sig (hex, Ed25519 detached) over sha256(signable(op))
// with op.DevicePub. Signing the 32-byte digest keeps the signed message tiny
// and identical on both sides of the runtime boundary.
func verifySig(op Op) bool {
	pub, err := hex.DecodeString(op.DevicePub)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := hex.DecodeString(op.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	digest := sha256.Sum256(signable(op))
	return ed25519.Verify(ed25519.PublicKey(pub), digest[:], sig)
}

func shortKey(hexKey string) string {
	if len(hexKey) > 8 {
		return hexKey[:8] + "…"
	}
	return hexKey
}

// checkCapability is the gatekeeper run before ANY op is folded when
// enforcement is on. It returns "" when the op may proceed, else the
// deterministic rejection reason. Grant-plane ops (cap.*) are additionally
// applied here (they mutate the grant table / epoch, not business state).
// Validity is evaluated at the op's position in the CANONICAL order — so
// whether an op beats a revocation is decided by the same total order every
// peer already agrees on, never by wall-clock or arrival time (MESH-D13).
func checkCapability(st *State, cfg Config, op Op) (handled bool, reason string) {
	return capabilityGate(st.Grants, &st.CapEpoch, cfg, op)
}

// capabilityGate is the shared enforcement core, parameterized over whichever
// fold's grant table it guards — the business State (Missions C/D) or a
// RoomState (Messenger Wave 1: each room Autobase carries its own per-room
// grant plane, same law). Pure code motion from checkCapability (2026-07-18);
// the Mission D unit tests + goldens are the proof it did not move semantics.
func capabilityGate(grants map[string]GrantState, capEpoch *int64, cfg Config, op Op) (handled bool, reason string) {
	// 1. Every op must carry a valid signature from its claimed device key.
	if op.DevicePub == "" || op.Sig == "" {
		return false, "capability: unsigned op (devicePub+sig required when an authority is configured)"
	}
	if !verifySig(op) {
		return false, "capability: signature verification failed for device " + shortKey(op.DevicePub)
	}

	isAuthority := op.DevicePub == cfg.AuthorityPub

	switch op.Kind {
	case "cap.grant":
		if !isAuthority {
			return true, "capability: grants must be signed by the mesh authority (got device " + shortKey(op.DevicePub) + ")"
		}
		if op.Device == "" {
			return true, "capability: grant missing device key"
		}
		role := op.Role
		if role == "" {
			role = "writer"
		}
		grants[op.Device] = GrantState{Role: role, Epoch: op.Epoch}
		return true, ""

	case "cap.epoch":
		if !isAuthority {
			return true, "capability: epoch bumps must be signed by the mesh authority"
		}
		if op.Epoch <= *capEpoch {
			return true, "capability: epoch must increase (have " +
				strconv.FormatInt(*capEpoch, 10) + ", got " + strconv.FormatInt(op.Epoch, 10) + ")"
		}
		*capEpoch = op.Epoch
		return true, ""

	case "cap.revoke":
		if !isAuthority {
			return true, "capability: revocations must be signed by the mesh authority"
		}
		delete(grants, op.Device)
		return true, ""
	}

	// 2. Domain ops: the authority is implicitly granted; everyone else needs
	//    a grant issued under the CURRENT epoch.
	if isAuthority {
		return false, ""
	}
	grant, ok := grants[op.DevicePub]
	if !ok {
		return false, "capability: no grant for device " + shortKey(op.DevicePub)
	}
	if grant.Epoch != *capEpoch {
		return false, "capability: grant epoch " + strconv.FormatInt(grant.Epoch, 10) +
			" is stale (current epoch " + strconv.FormatInt(*capEpoch, 10) +
			") — device " + shortKey(op.DevicePub) + " was not re-issued"
	}
	return false, ""
}
