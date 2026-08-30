package denominator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func DecodeFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("decode %s: trailing JSON value", path)
	}
	return nil
}

func WriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func ContractDigest(contract Denominator) (string, error) {
	if err := ValidateContract(contract); err != nil {
		return "", err
	}
	return digestValue(contract)
}

func digestValue(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var canonical any
	if err := json.Unmarshal(encoded, &canonical); err != nil {
		return "", err
	}
	encoded, err = json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func ValidateContract(contract Denominator) error {
	if contract.Schema != ContractSchema {
		return fmt.Errorf("schema must be %q", ContractSchema)
	}
	if contract.ContractID == "" || contract.Version < 1 || contract.Description == "" {
		return errors.New("contract identity and description are required")
	}
	if len(contract.Cells) == 0 {
		return errors.New("contract must declare at least one cell")
	}
	ids := make(map[string]bool, len(contract.Cells))
	seen := map[string]map[string]string{
		"meta_activity": {}, "source": {}, "ir": {}, "generated_artifact": {}, "evaluator": {}, "metric_id": {},
	}
	proofs := map[ProofChoice]int{}
	indicators := map[IndicatorClass]int{}
	for _, cell := range contract.Cells {
		if cell.ID == "" || ids[cell.ID] {
			return fmt.Errorf("cell id must be unique: %q", cell.ID)
		}
		ids[cell.ID] = true
		if cell.Stage == "" || cell.Step == "" || !cell.ProofChoice.Valid() || !cell.IndicatorClass.Valid() {
			return fmt.Errorf("cell %q has invalid stage, step, proof choice, or indicator class", cell.ID)
		}
		if cell.MetricID == "" || cell.MetricDenominator < 1 {
			return fmt.Errorf("cell %q must have a fixed positive metric denominator", cell.ID)
		}
		values := map[string]string{
			"meta_activity": cell.MetaActivity,
			"source": cell.Source,
			"ir": cell.IR,
			"generated_artifact": cell.GeneratedArtifact,
			"evaluator": cell.Evaluator,
			"metric_id": cell.MetricID,
		}
		for kind, value := range values {
			if value == "" {
				return fmt.Errorf("cell %q has empty %s binding", cell.ID, kind)
			}
			if prior, exists := seen[kind][value]; exists {
				return fmt.Errorf("%s %q is shared by %q and %q", kind, value, prior, cell.ID)
			}
			seen[kind][value] = cell.ID
		}
		proofs[cell.ProofChoice]++
		indicators[cell.IndicatorClass]++
	}
	for _, cell := range contract.Cells {
		for _, dependency := range cell.DependsOn {
			if dependency == cell.ID || !ids[dependency] {
				return fmt.Errorf("cell %q has invalid dependency %q", cell.ID, dependency)
			}
		}
	}
	if !balanced(proofs) || !balanced(indicators) {
		return errors.New("proof choices and indicator classes must remain balanced")
	}
	return nil
}

func ValidateEvidence(evidence Evidence) error {
	if evidence.Schema != EvidenceSchema {
		return fmt.Errorf("schema must be %q", EvidenceSchema)
	}
	if evidence.ContractID == "" || evidence.ContractVersion < 1 || evidence.ContractDigest == "" || evidence.InputDigest == "" || evidence.ToolDigest == "" || evidence.CriteriaDigest == "" {
		return errors.New("evidence identity, digests, and criteria are required")
	}
	if evidence.Run.DenominatorDigestAtStart == "" || evidence.Run.DenominatorDigestAtEnd == "" || evidence.Run.CriteriaDigestAtStart == "" || evidence.Run.CriteriaDigestAtEnd == "" || evidence.Run.CellIDsAtStart == nil || evidence.Run.CellIDsAtEnd == nil {
		return errors.New("run observation is incomplete")
	}
	seen := map[string]bool{}
	for _, cell := range evidence.Cells {
		if cell.CellID == "" || seen[cell.CellID] {
			return fmt.Errorf("evidence cell id must be unique: %q", cell.CellID)
		}
		seen[cell.CellID] = true
		if cell.MetaActivity == "" || cell.Source == "" || cell.IR == "" || cell.GeneratedArtifact == "" || cell.Evaluator == "" || cell.Decision == "" {
			return fmt.Errorf("evidence for %q is incomplete", cell.CellID)
		}
		if cell.Decision == string(Unknown) && !validUnknownClaim(cell.Claim) {
			return fmt.Errorf("UNKNOWN evidence for %q must preserve the six unknown fields", cell.CellID)
		}
	}
	return nil
}

func validUnknownClaim(claim Claim) bool {
	return claim.State == Unknown && claim.Stage != "" && claim.Step != "" && claim.Reason != "" && claim.UnknownClass != "" && claim.NextOperation != "" && claim.BlockedBy != nil
}

func balanced[T comparable](counts map[T]int) bool {
	if len(counts) != 3 {
		return false
	}
	minimum, maximum := -1, -1
	for _, count := range counts {
		if minimum == -1 || count < minimum {
			minimum = count
		}
		if count > maximum {
			maximum = count
		}
	}
	return maximum-minimum <= 1
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
