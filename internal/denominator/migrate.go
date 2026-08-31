package denominator

import "fmt"

const (
	MigrationAdd    = "ADD"
	MigrationSplit  = "SPLIT"
	MigrationRetire = "RETIRE"
)

func ValidateMigration(from, to Denominator, receipt MigrationReceipt) MigrationReport {
	fromDigest, fromErr := ContractDigest(from)
	toDigest, toErr := ContractDigest(to)
	report := MigrationReport{
		Schema: ReceiptSchema, MigrationID: receipt.MigrationID,
		FromVersion: from.Version, ToVersion: to.Version,
		Claim: Claim{BlockedBy: []string{}},
	}
	if fromErr != nil || toErr != nil {
		return failMigration(report, "MIGRATION", "VALIDATE_CONTRACTS", "MALFORMED_MIGRATION_CONTRACT", "RESTORE_VALID_CONTRACTS")
	}
	if receipt.Schema != ReceiptSchema || receipt.MigrationID == "" || receipt.FromContractID == "" || receipt.ToContractID == "" || receipt.Reason == "" || !receipt.ProofChoice.Valid() || len(receipt.AffectedClaims) == 0 {
		return failMigration(report, "MIGRATION", "VALIDATE_RECEIPT", "MALFORMED_MIGRATION_RECEIPT", "RESTORE_MIGRATION_RECEIPT")
	}
	if from.ContractID != to.ContractID || receipt.FromContractID != from.ContractID || receipt.ToContractID != to.ContractID || receipt.FromVersion != from.Version || receipt.ToVersion != to.Version || from.Version+1 != to.Version || receipt.FromContractDigest != fromDigest || receipt.ToContractDigest != toDigest {
		return failMigration(report, "IDENTITY", "BIND_MIGRATION_DIGESTS", "MIGRATION_IDENTITY_MISMATCH", "RECREATE_RECEIPT_FOR_EXACT_VERSIONS")
	}
	fromByID := make(map[string]DenominatorCell, len(from.Cells))
	toByID := make(map[string]DenominatorCell, len(to.Cells))
	for _, cell := range from.Cells {
		fromByID[cell.ID] = cell
	}
	for _, cell := range to.Cells {
		toByID[cell.ID] = cell
	}
	for id, oldCell := range fromByID {
		if newCell, exists := toByID[id]; exists && !sameCell(oldCell, newCell) {
			return failMigration(report, "MIGRATION", "PRESERVE_CELL_IDENTITY", "CELL_ID_REUSED_WITH_NEW_BINDING", "SPLIT_OR_ADD_WITH_NEW_CELL_ID")
		}
	}
	claimedFrom := map[string]bool{}
	claimedTo := map[string]bool{}
	for _, operation := range receipt.Operations {
		if operation.Reason == "" || !operation.ProofChoice.Valid() || len(operation.AffectedClaims) == 0 {
			return failMigration(report, "MIGRATION", "VALIDATE_OPERATION", "MALFORMED_MIGRATION_OPERATION", "RESTORE_OPERATION_REASON_PROOF_AND_CLAIMS")
		}
		switch operation.Kind {
		case MigrationAdd:
			if operation.SourceCellID != "" || len(operation.TargetCellIDs) != 1 {
				return failMigration(report, "MIGRATION", "VALIDATE_ADD", "INVALID_ADD_OPERATION", "DECLARE_ONE_NEW_CELL")
			}
			for _, id := range operation.TargetCellIDs {
				if !cellExists(toByID, id) || cellExists(fromByID, id) || claimedTo[id] {
					return failMigration(report, "MIGRATION", "VALIDATE_ADD", "ADD_TARGET_NOT_NEW", "DECLARE_UNIQUE_NEW_CELL")
				}
				claimedTo[id] = true
			}
			report.Added++
		case MigrationSplit:
			if operation.SourceCellID == "" || len(operation.TargetCellIDs) < 2 || !cellExists(fromByID, operation.SourceCellID) || cellExists(toByID, operation.SourceCellID) || claimedFrom[operation.SourceCellID] {
				return failMigration(report, "MIGRATION", "VALIDATE_SPLIT", "INVALID_SPLIT_OPERATION", "DECLARE_REMOVED_SOURCE_AND_TWO_TARGETS")
			}
			claimedFrom[operation.SourceCellID] = true
			for _, id := range operation.TargetCellIDs {
				if !cellExists(toByID, id) || cellExists(fromByID, id) || claimedTo[id] {
					return failMigration(report, "MIGRATION", "VALIDATE_SPLIT", "SPLIT_TARGET_NOT_NEW", "DECLARE_UNIQUE_SPLIT_TARGETS")
				}
				claimedTo[id] = true
			}
			report.Split++
		case MigrationRetire:
			if operation.SourceCellID == "" || len(operation.TargetCellIDs) != 0 || !cellExists(fromByID, operation.SourceCellID) || cellExists(toByID, operation.SourceCellID) || claimedFrom[operation.SourceCellID] {
				return failMigration(report, "MIGRATION", "VALIDATE_RETIRE", "INVALID_RETIRE_OPERATION", "DECLARE_ONE_REMOVED_CELL")
			}
			if !validRetirementEvidence(operation.RetirementEvidence, fromByID[operation.SourceCellID]) {
				return failMigration(report, "MIGRATION", "VALIDATE_RETIRE", "RETIREMENT_EVIDENCE_MISSING", "PROVIDE_CLOSED_RETIREMENT_EVIDENCE")
			}
			claimedFrom[operation.SourceCellID] = true
			report.Retired++
		default:
			return failMigration(report, "MIGRATION", "VALIDATE_OPERATION", fmt.Sprintf("UNRECOGNIZED_MIGRATION_KIND_%s", operation.Kind), "USE_ADD_SPLIT_OR_RETIRE")
		}
	}
	for id := range fromByID {
		if cellExists(toByID, id) {
			continue
		}
		if !claimedFrom[id] {
			return failMigration(report, "MIGRATION", "ACCOUNT_FOR_REMOVED_CELL", "UNACCOUNTED_CELL_REMOVAL", "ADD_EXPLICIT_SPLIT_OR_RETIRE_RECEIPT")
		}
	}
	for id := range toByID {
		if cellExists(fromByID, id) {
			continue
		}
		if !claimedTo[id] {
			return failMigration(report, "MIGRATION", "ACCOUNT_FOR_NEW_CELL", "UNACCOUNTED_CELL_ADDITION", "ADD_EXPLICIT_ADD_OR_SPLIT_RECEIPT")
		}
	}
	if report.Added+report.Split+report.Retired == 0 {
		return failMigration(report, "MIGRATION", "REQUIRE_CHANGE", "EMPTY_MIGRATION", "DECLARE_EXPLICIT_VERSION_CHANGE")
	}
	report.Decision = Closed
	report.Claim = Claim{State: Closed, BlockedBy: []string{}}
	return report
}

func failMigration(report MigrationReport, stage, step, reason, next string) MigrationReport {
	report.Decision = Refuted
	report.FailClosed = true
	report.Claim = refutedClaim(stage, step, reason, next, nil)
	return report
}

func cellExists(cells map[string]DenominatorCell, id string) bool {
	if id == "" {
		return false
	}
	_, exists := cells[id]
	return exists
}

func sameCell(left, right DenominatorCell) bool {
	return left.ID == right.ID && left.Stage == right.Stage && left.Step == right.Step && left.ProofChoice == right.ProofChoice && left.IndicatorClass == right.IndicatorClass && left.MetaActivity == right.MetaActivity && left.Source == right.Source && left.IR == right.IR && left.GeneratedArtifact == right.GeneratedArtifact && left.Evaluator == right.Evaluator && left.MetricID == right.MetricID && left.MetricDenominator == right.MetricDenominator && equalStrings(left.DependsOn, right.DependsOn)
}

func validRetirementEvidence(evidence *RetirementEvidence, cell DenominatorCell) bool {
	if evidence == nil || evidence.CellID != cell.ID || evidence.Decision != string(Closed) || evidence.Reason == "" {
		return false
	}
	return evidence.MetaActivity == cell.MetaActivity && evidence.Source == cell.Source && evidence.IR == cell.IR && evidence.GeneratedArtifact == cell.GeneratedArtifact && evidence.Evaluator == cell.Evaluator
}
