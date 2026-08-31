package denominator

import "fmt"

func Evaluate(contract Denominator, evidence Evidence) Report {
	contractDigest, contractErr := ContractDigest(contract)
	report := baseReport(contract, contractDigest, evidence)
	if contractErr != nil {
		return refuteAll(report, contract, "CONTRACT", "VALIDATE_DENOMINATOR", contractErr.Error(), "RESTORE_VALID_DENOMINATOR")
	}
	if err := ValidateEvidence(evidence); err != nil {
		return refuteAll(report, contract, "EVIDENCE", "VALIDATE_EVIDENCE", err.Error(), "RESTORE_VALID_EVIDENCE")
	}
	if evidence.ContractID != contract.ContractID || evidence.ContractVersion != contract.Version || evidence.ContractDigest != contractDigest {
		return refuteAll(report, contract, "IDENTITY", "BIND_CONTRACT_DIGEST", "CONTRACT_DIGEST_MISMATCH", "REPLAY_WITH_EXACT_CONTRACT")
	}
	contractIDs := cellIDs(contract)
	if evidence.Run.DenominatorDigestAtStart != contractDigest || evidence.Run.DenominatorDigestAtEnd != contractDigest || !equalStrings(evidence.Run.CellIDsAtStart, contractIDs) || !equalStrings(evidence.Run.CellIDsAtEnd, contractIDs) {
		return refuteAll(report, contract, "RUN", "VERIFY_DENOMINATOR_CARDINALITY", "DENOMINATOR_REDUCED_DURING_RUN", "RESTART_WITH_IMMUTABLE_DENOMINATOR")
	}
	if evidence.Run.SuccessBeforeChange && evidence.Run.CriteriaDigestAtStart != evidence.Run.CriteriaDigestAtEnd {
		return refuteAll(report, contract, "RUN", "VERIFY_CRITERIA_AFTER_SUCCESS", "CRITERIA_CHANGED_AFTER_SUCCESS", "REPLAY_WITH_VERSIONED_CRITERIA")
	}
	if evidence.Run.PrivilegeEscalationRequested && !contract.RunPolicy.AllowPrivilegeEscalation {
		return refuteAll(report, contract, "AUTHORITY", "CHECK_PRIVILEGE_BOUNDARY", "PRIVILEGE_ESCALATION_REQUESTED", "REMOVE_PRIVILEGE_ESCALATION")
	}

	evidenceByID := make(map[string]CellEvidence, len(evidence.Cells))
	for _, item := range evidence.Cells {
		if _, exists := evidenceByID[item.CellID]; exists {
			return refuteAll(report, contract, "EVIDENCE", "BIND_CELL_EVIDENCE", "DUPLICATE_CELL_EVIDENCE", "REMOVE_DUPLICATE_CELL_EVIDENCE")
		}
		evidenceByID[item.CellID] = item
	}
	for _, item := range evidence.Cells {
		if !containsID(contractIDs, item.CellID) {
			return refuteAll(report, contract, "EVIDENCE", "BIND_CELL_EVIDENCE", "EVIDENCE_REFERENCES_UNKNOWN_CELL", "REMOVE_UNKNOWN_CELL_EVIDENCE")
		}
	}

	for _, cell := range contract.Cells {
		item, exists := evidenceByID[cell.ID]
		if !exists {
			report.Cells = append(report.Cells, CellResult{
				CellID: cell.ID, ProofChoice: cell.ProofChoice, IndicatorClass: cell.IndicatorClass,
				State: Unknown, Claim: unknownClaim(cell, "EVIDENCE_MISSING", "DIRECT_MISSING", "PROVIDE_CELL_EVIDENCE", nil),
			})
			continue
		}
		if !sameBinding(cell, item) {
			report.Cells = append(report.Cells, CellResult{
				CellID: cell.ID, ProofChoice: cell.ProofChoice, IndicatorClass: cell.IndicatorClass,
				State: Refuted, Claim: refutedClaim(cell.Stage, cell.Step, "CELL_BINDING_MISMATCH", "RESTORE_ONE_TO_ONE_CELL_BINDINGS", nil),
			})
			continue
		}
		result := CellResult{CellID: cell.ID, ProofChoice: cell.ProofChoice, IndicatorClass: cell.IndicatorClass}
		switch item.Decision {
		case string(Closed):
			if item.Claim.State != "" && item.Claim.State != Closed {
				result.State = Refuted
				result.Claim = refutedClaim(cell.Stage, cell.Step, "CLOSED_EVIDENCE_HAS_NONCLOSED_CLAIM", "RESTORE_CLOSED_CLAIM", nil)
			} else {
				result.State = Closed
				result.Claim = Claim{State: Closed, BlockedBy: []string{}}
			}
		case string(Unknown):
			result.State = Unknown
			result.Claim = item.Claim
		case string(Refuted):
			result.State = Refuted
			result.Claim = item.Claim
			if result.Claim.State == "" {
				result.Claim = refutedClaim(cell.Stage, cell.Step, "EVIDENCE_REFUTED", "RESTORE_REFUTED_EVIDENCE", nil)
			}
		default:
			result.State = Refuted
			result.Claim = refutedClaim(cell.Stage, cell.Step, fmt.Sprintf("UNRECOGNIZED_DECISION_%s", item.Decision), "USE_CLOSED_UNKNOWN_OR_REFUTED", nil)
		}
		report.Cells = append(report.Cells, result)
	}
	return finalizeReport(report, contract)
}

func baseReport(contract Denominator, contractDigest string, evidence Evidence) Report {
	return Report{
		Schema: ReportSchema, ContractID: contract.ContractID, ContractVersion: contract.Version,
		ContractDigest: contractDigest, InputDigest: evidence.InputDigest, ToolDigest: evidence.ToolDigest,
		Claim: Claim{BlockedBy: []string{}},
	}
}

func refuteAll(report Report, contract Denominator, stage, step, reason, next string) Report {
	for _, cell := range contract.Cells {
		report.Cells = append(report.Cells, CellResult{
			CellID: cell.ID, ProofChoice: cell.ProofChoice, IndicatorClass: cell.IndicatorClass,
			State: Refuted, Claim: refutedClaim(cell.Stage, cell.Step, reason, next, nil),
		})
	}
	if len(report.Cells) == 0 {
		report.Claim = refutedClaim(stage, step, reason, next, nil)
		report.Decision = Refuted
		report.FailClosed = true
		report.Summary = Summary{}
		return report
	}
	return finalizeReportWithContract(report, contract, true)
}

func finalizeReport(report Report, contract Denominator) Report {
	return finalizeReportWithContract(report, contract, false)
}

func finalizeReportWithContract(report Report, contract Denominator, failClosed bool) Report {
	for _, cell := range contract.Cells {
		for _, result := range report.Cells {
			if result.CellID != cell.ID {
				continue
			}
			numerator := 0
			if result.State == Closed {
				numerator = 1
			}
			report.Metrics = append(report.Metrics, MetricResult{
				ID: cell.MetricID, CellID: cell.ID, Numerator: numerator, Denominator: cell.MetricDenominator,
				State: result.State, MetaActivity: cell.MetaActivity, Source: cell.Source, IR: cell.IR,
				GeneratedArtifact: cell.GeneratedArtifact, Evaluator: cell.Evaluator,
			})
			break
		}
	}
	return finalizeReportFromCells(report, failClosed)
}

func finalizeReportFromCells(report Report, failClosed bool) Report {
	report.Summary = Summary{Total: len(report.Cells)}
	proofs := map[ProofChoice]*BucketResult{}
	for _, choice := range []ProofChoice{Foundation, Coherence, Regression} {
		bucket := BucketResult{Name: string(choice)}
		proofs[choice] = &bucket
	}
	indicators := map[IndicatorClass]*BucketResult{}
	for _, class := range []IndicatorClass{Driver, Outcome, Guardrail} {
		bucket := BucketResult{Name: string(class)}
		indicators[class] = &bucket
	}
	for _, result := range report.Cells {
		switch result.State {
		case Closed:
			report.Summary.Closed++
		case Unknown:
			report.Summary.Unknown++
		case Refuted:
			report.Summary.Refuted++
		}
		if bucket := proofs[result.ProofChoice]; bucket != nil {
			bucket.Total++
			incrementBucket(bucket, result.State)
		}
		if bucket := indicators[result.IndicatorClass]; bucket != nil {
			bucket.Total++
			incrementBucket(bucket, result.State)
		}
	}
	report.Decision = Closed
	if report.Summary.Unknown > 0 {
		report.Decision = Unknown
	}
	if report.Summary.Refuted > 0 {
		report.Decision = Refuted
	}
	report.FailClosed = failClosed || report.Summary.Refuted > 0
	report.Proofs = make([]BucketResult, 0, 3)
	for _, choice := range []ProofChoice{Foundation, Coherence, Regression} {
		report.Proofs = append(report.Proofs, *proofs[choice])
	}
	report.Indicators = make([]BucketResult, 0, 3)
	for _, class := range []IndicatorClass{Driver, Outcome, Guardrail} {
		report.Indicators = append(report.Indicators, *indicators[class])
	}
	for _, result := range report.Cells {
		if result.State == Refuted {
			report.Claim = result.Claim
			return report
		}
	}
	for _, result := range report.Cells {
		if result.State == Unknown {
			report.Claim = result.Claim
			return report
		}
	}
	report.Claim = Claim{State: Closed, BlockedBy: []string{}}
	return report
}

func incrementBucket(bucket *BucketResult, state State) {
	switch state {
	case Closed:
		bucket.Closed++
	case Unknown:
		bucket.Unknown++
	case Refuted:
		bucket.Refuted++
	}
}

func unknownClaim(cell DenominatorCell, reason, class, next string, blocked []string) Claim {
	if blocked == nil {
		blocked = []string{}
	}
	return Claim{State: Unknown, Stage: cell.Stage, Step: cell.Step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: blocked}
}

func refutedClaim(stage, step, reason, next string, blocked []string) Claim {
	if blocked == nil {
		blocked = []string{}
	}
	return Claim{State: Refuted, Stage: stage, Step: step, Reason: reason, NextOperation: next, BlockedBy: blocked}
}

func sameBinding(cell DenominatorCell, evidence CellEvidence) bool {
	return cell.MetaActivity == evidence.MetaActivity && cell.Source == evidence.Source && cell.IR == evidence.IR && cell.GeneratedArtifact == evidence.GeneratedArtifact && cell.Evaluator == evidence.Evaluator
}

func cellIDs(contract Denominator) []string {
	ids := make([]string, 0, len(contract.Cells))
	for _, cell := range contract.Cells {
		ids = append(ids, cell.ID)
	}
	return ids
}

func containsID(ids []string, wanted string) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}
