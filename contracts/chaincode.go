package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ======= Data Model (draft; no PII on-chain) =======

// Credential is minimal metadata stored on-chain. Actual attributes should be kept off-chain.
// 'hashedData' is a content hash (e.g., SHA-256) over the canonicalized VC payload.
type Credential struct {
	CredID     string `json:"credId"`      // unique credential id
	HolderDID  string `json:"holderDid"`   // DID of holder
	CredType   string `json:"credType"`    // e.g., Diploma, License
	HashedData string `json:"hashedData"`  // hash of VC payload
	IssuerID   string `json:"issuerId"`    // org/user id within the network
	Status     string `json:"status"`      // Active | Revoked
	CreatedAt  string `json:"createdAt"`   // RFC3339 timestamp
	UpdatedAt  string `json:"updatedAt"`   // RFC3339 timestamp
}

// AccessEvent captures verification/access events for audit trail.
type AccessEvent struct {
	EventID    string `json:"eventId"`     // unique id
	CredID     string `json:"credId"`
	Action     string `json:"action"`      // Verify | Revoke | Issue
	ActorID    string `json:"actorId"`     // verifier / issuer / revoker
	Outcome    string `json:"outcome"`     // Success | Failure
	Reason     string `json:"reason"`      // optional notes
	OccurredAt string `json:"occurredAt"`  // RFC3339 timestamp
}

// VerificationResult is returned by VerifyCreds.
type VerificationResult struct {
	CredID      string `json:"credId"`
	IsActive    bool   `json:"isActive"`
	HashMatches bool   `json:"hashMatches"` // placeholder until off-chain check is wired
	CheckedAt   string `json:"checkedAt"`
}

// ======= Smart Contract =======

type SmartContract struct {
	contractapi.Contract
}

// IssueCreds creates a new credential record and emits an 'Issue' event.
//
// Arguments (draft):
// - credID: unique id for the VC
// - holderDID: DID of the holder
// - credType: "Diploma" | "License"
// - hashedData: SHA-256 of the canonical VC (off-chain)
// - issuerID: id of the issuer (MSP subject / client id)
func (s *SmartContract) IssueCreds(ctx contractapi.TransactionContextInterface,
	credID, holderDID, credType, hashedData, issuerID string) error {

	// TODO: add auth checks based on client identity (issuer role)
	exists, err := s.credExists(ctx, credID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("credential %s already exists", credID)
	}

	cred := &Credential{
		CredID:     credID,
		HolderDID:  holderDID,
		CredType:   credType,
		HashedData: hashedData,
		IssuerID:   issuerID,
		Status:     "Active",
		CreatedAt:  nowRFC3339(),
		UpdatedAt:  nowRFC3339(),
	}

	bz, _ := json.Marshal(cred)
	if err := ctx.GetStub().PutState(credKey(credID), bz); err != nil {
		return err
	}

	evt := AccessEvent{
		EventID:    newULID(),
		CredID:     credID,
		Action:     "Issue",
		ActorID:    issuerID,
		Outcome:    "Success",
		Reason:     "",
		OccurredAt: nowRFC3339(),
	}
	emitEvent(ctx, "AuditTrail", evt)
	return nil
}

// VerifyCreds records a verify event and returns a verification result.
// NOTE: hash matching against off-chain data is a placeholder until wired.
func (s *SmartContract) VerifyCreds(ctx contractapi.TransactionContextInterface,
	credID, verifierID string) (*VerificationResult, error) {

	cred, err := s.getCred(ctx, credID)
	if err != nil {
		return nil, err
	}
	isActive := cred.Status == "Active"

	res := &VerificationResult{
		CredID:      credID,
		IsActive:    isActive,
		HashMatches: true, // TODO: connect to off-chain hash check
		CheckedAt:   nowRFC3339(),
	}

	evt := AccessEvent{
		EventID:    newULID(),
		CredID:     credID,
		Action:     "Verify",
		ActorID:    verifierID,
		Outcome:    "Success",
		Reason:     "",
		OccurredAt: nowRFC3339(),
	}
	emitEvent(ctx, "AuditTrail", evt)

	return res, nil
}

// RevokeCreds marks a credential as revoked and stores a revocation event.
func (s *SmartContract) RevokeCreds(ctx contractapi.TransactionContextInterface,
	credID, reason, revokerID string) error {

	cred, err := s.getCred(ctx, credID)
	if err != nil {
		return err
	}
	if cred.Status == "Revoked" {
		return fmt.Errorf("credential %s is already revoked", credID)
	}

	cred.Status = "Revoked"
	cred.UpdatedAt = nowRFC3339()

	bz, _ := json.Marshal(cred)
	if err := ctx.GetStub().PutState(credKey(credID), bz); err != nil {
		return err
	}

	evt := AccessEvent{
		EventID:    newULID(),
		CredID:     credID,
		Action:     "Revoke",
		ActorID:    revokerID,
		Outcome:    "Success",
		Reason:     reason,
		OccurredAt: nowRFC3339(),
	}
	emitEvent(ctx, "AuditTrail", evt)
	return nil
}

// QueryAuditTrail returns paginated events for a holder DID.
// Pagination uses Fabric bookmarks; in a real system we'd index by holderDID.
func (s *SmartContract) QueryAuditTrail(ctx contractapi.TransactionContextInterface,
	holderDID string, pageSize int32, bookmark string) ([]AccessEvent, string, error) {

	// Placeholder: scan over a composite key "event~credId"
	// TODO: use rich queries when CouchDB is enabled
	iter, _, err := ctx.GetStub().GetStateByPartialCompositeKeyWithPagination("event~credId", []string{}, pageSize, bookmark)
	if err != nil {
		return nil, "", err
	}
	defer iter.Close()

	var out []AccessEvent
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, "", err
		}
		var evt AccessEvent
		if err := json.Unmarshal(kv.Value, &evt); err == nil {
			// Filter by holder if needed (requires joining with credID->holderDID)
			out = append(out, evt)
		}
	}
	// NOTE: bookmark passthrough is omitted in this stub. Return empty.
	return out, "", nil
}

// ======= Helpers (stubs) =======

func (s *SmartContract) credExists(ctx contractapi.TransactionContextInterface, credID string) (bool, error) {
	val, err := ctx.GetStub().GetState(credKey(credID))
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

func (s *SmartContract) getCred(ctx contractapi.TransactionContextInterface, credID string) (*Credential, error) {
	bz, err := ctx.GetStub().GetState(credKey(credID))
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, fmt.Errorf("credential %s not found", credID)
	}
	var cred Credential
	if err := json.Unmarshal(bz, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func credKey(credID string) string { return "cred:" + credID }

// nowRFC3339, newULID, emitEvent are intentionally simple placeholders.

func nowRFC3339() string { return "2025-11-06T00:00:00Z" }

func newULID() string { return "01HFYH7Y00000000000000" }

func emitEvent(ctx contractapi.TransactionContextInterface, name string, payload interface{}) {
	if bz, err := json.Marshal(payload); err == nil {
		_ = ctx.GetStub().SetEvent(name, bz)
	}
}

func main() {
	cc, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		panic(err)
	}
	if err := cc.Start(); err != nil {
		panic(err)
	}
}