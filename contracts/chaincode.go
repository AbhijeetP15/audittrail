package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Minimal on-chain metadata; keep PII off-ledger.
type Credential struct {
	CredID     string `json:"credId"`
	HolderDID  string `json:"holderDid"`
	CredType   string `json:"credType"`
	HashedData string `json:"hashedData"`
	IssuerID   string `json:"issuerId"`
	Status     string `json:"status"`     // Active | Revoked
	CreatedAt  string `json:"createdAt"`  // RFC3339
	UpdatedAt  string `json:"updatedAt"`  // RFC3339
}

// AccessEvent captures audit trail entries.
type AccessEvent struct {
	EventID    string `json:"eventId"`
	CredID     string `json:"credId"`
	HolderDID  string `json:"holderDid"`
	Action     string `json:"action"`     // Issue | Verify | Revoke
	ActorID    string `json:"actorId"`    // issuer | verifier | revoker
	Outcome    string `json:"outcome"`    // Success | Failure
	Reason     string `json:"reason"`     // optional
	OccurredAt string `json:"occurredAt"` // RFC3339
}

type VerificationResult struct {
	CredID      string `json:"credId"`
	IsActive    bool   `json:"isActive"`
	HashMatches bool   `json:"hashMatches"`
	CheckedAt   string `json:"checkedAt"`
}

type SmartContract struct {
	contractapi.Contract
}

// IssueCreds creates a credential and records an Issue event.
func (s *SmartContract) IssueCreds(ctx contractapi.TransactionContextInterface,
	credID, holderDID, credType, hashedData, issuerID string) error {

	exists, err := s.credExists(ctx, credID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("credential %s already exists", credID)
	}

	now := nowRFC3339()
	cred := &Credential{
		CredID:     credID,
		HolderDID:  holderDID,
		CredType:   credType,
		HashedData: hashedData,
		IssuerID:   issuerID,
		Status:     "Active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	bz, _ := json.Marshal(cred)
	if err := ctx.GetStub().PutState(credKey(credID), bz); err != nil {
		return err
	}

	return s.recordEvent(ctx, credID, holderDID, "Issue", issuerID, "Success", "")
}

// VerifyCreds records a verify event and returns a verification result.
// HashMatches is a placeholder until off-chain hash checks are wired.
func (s *SmartContract) VerifyCreds(ctx contractapi.TransactionContextInterface,
	credID, verifierID string) (*VerificationResult, error) {

	cred, err := s.getCred(ctx, credID)
	if err != nil {
		return nil, err
	}

	res := &VerificationResult{
		CredID:      credID,
		IsActive:    cred.Status == "Active",
		HashMatches: true,
		CheckedAt:   nowRFC3339(),
	}

	if err := s.recordEvent(ctx, credID, cred.HolderDID, "Verify", verifierID, "Success", ""); err != nil {
		return nil, err
	}
	return res, nil
}

// RevokeCreds marks the credential revoked and records the event.
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

	return s.recordEvent(ctx, credID, cred.HolderDID, "Revoke", revokerID, "Success", reason)
}

// QueryAuditTrail returns paginated events for a holder DID.
func (s *SmartContract) QueryAuditTrail(ctx contractapi.TransactionContextInterface,
	holderDID string, pageSize int32, bookmark string) ([]AccessEvent, string, error) {

	iter, nextBookmark, err := ctx.GetStub().GetStateByPartialCompositeKeyWithPagination(
		"event~holder", []string{holderDID}, pageSize, bookmark)
	if err != nil {
		return nil, "", err
	}
	defer iter.Close()

	var events []AccessEvent
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, "", err
		}
		var evt AccessEvent
		if err := json.Unmarshal(kv.Value, &evt); err != nil {
			return nil, "", err
		}
		events = append(events, evt)
	}
	return events, nextBookmark, nil
}

// ===== Helpers =====

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

func (s *SmartContract) recordEvent(ctx contractapi.TransactionContextInterface,
	credID, holderDID, action, actorID, outcome, reason string) error {

	evt := AccessEvent{
		EventID:    newEventID(),
		CredID:     credID,
		HolderDID:  holderDID,
		Action:     action,
		ActorID:    actorID,
		Outcome:    outcome,
		Reason:     reason,
		OccurredAt: nowRFC3339(),
	}
	bz, _ := json.Marshal(evt)

	ck, err := ctx.GetStub().CreateCompositeKey("event~holder", []string{holderDID, credID, evt.EventID})
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(ck, bz); err != nil {
		return err
	}
	ctx.GetStub().SetEvent("AuditTrail", bz)
	return nil
}

func credKey(credID string) string { return "cred:" + credID }

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func newEventID() string {
	now := time.Now().UTC().UnixNano()
	var n uint64
	_ = binary.Read(rand.Reader, binary.BigEndian, &n)
	return fmt.Sprintf("%d-%d", now, n)
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
