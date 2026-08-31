package denominator

import (
	"strings"
	"testing"
)

func TestContractRequiresBalancedOneToOneBindings(t *testing.T) {
	contract := testContract(1)
	if err := ValidateContract(contract); err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
	if _, err := ContractDigest(contract); err != nil {
		t.Fatalf("valid contract has no digest: %v", err)
	}
	contract.Cells[0].Source = contract.Cells[1].Source
	if err := ValidateContract(contract); err == nil {
		t.Fatal("shared source binding was accepted")
	}
}

func TestMigrationV1ToV2AccountsForAddSplitAndRetire(t *testing.T) {
	from := testContract(1)
	to := testContract(2)
	fromDigest, _ := ContractDigest(from)
	toDigest, _ := ContractDigest(to)
	receipt := MigrationReceipt{
		Schema: ReceiptSchema, MigrationID: "language-development-v1-v2",
		FromContractID: from.ContractID, FromVersion: from.Version, FromContractDigest: fromDigest,
		ToContractID: to.ContractID, ToVersion: to.Version, ToContractDigest: toDigest,
		Reason:      "split semantic proof, add migration proof, and retire replay proof with observed evidence",
		ProofChoice: Regression, AffectedClaims: []string{"language-development/completion", "language-development/replay"},
		Operations: []MigrationOperation{
			{Kind: MigrationSplit, SourceCellID: "semantic-foundation-outcome", TargetCellIDs: []string{"semantic-ir-foundation-outcome", "semantic-generated-foundation-outcome"}, Reason: "separate IR and generated artifact proof", ProofChoice: Foundation, AffectedClaims: []string{"language-development/completion"}},
			{Kind: MigrationAdd, TargetCellIDs: []string{"migration-regression-driver"}, Reason: "make migration receipts observable", ProofChoice: Regression, AffectedClaims: []string{"language-development/migration"}},
			{Kind: MigrationRetire, SourceCellID: "replay-regression-outcome", Reason: "replace the old replay cell with versioned replay evidence", ProofChoice: Regression, AffectedClaims: []string{"language-development/replay"}, RetirementEvidence: &RetirementEvidence{CellID: "replay-regression-outcome", Decision: string(Closed), Reason: "replay evidence retained in the immutable v1 report", MetaActivity: from.Cells[4].MetaActivity, Source: from.Cells[4].Source, IR: from.Cells[4].IR, GeneratedArtifact: from.Cells[4].GeneratedArtifact, Evaluator: from.Cells[4].Evaluator}},
		},
	}
	report := ValidateMigration(from, to, receipt)
	if report.Decision != Closed || report.FailClosed || report.Added != 1 || report.Split != 1 || report.Retired != 1 {
		t.Fatalf("unexpected migration report: %+v", report)
	}
}

func TestEvidenceStatesHaveExactCountsAndUnknownFields(t *testing.T) {
	contract := testContract(1)
	evidence := testEvidence(t, contract)
	complete := Evaluate(contract, evidence)
	if complete.Decision != Closed || complete.Summary != (Summary{Total: 6, Closed: 6}) {
		t.Fatalf("unexpected complete report: %+v", complete.Summary)
	}

	evidence.Cells = append(evidence.Cells[:2], evidence.Cells[3:]...)
	unknown := Evaluate(contract, evidence)
	if unknown.Decision != Unknown || unknown.Summary != (Summary{Total: 6, Closed: 5, Unknown: 1}) {
		t.Fatalf("unexpected unknown report: %+v", unknown.Summary)
	}
	claim := unknown.Cells[2].Claim
	if claim.State != Unknown || claim.Stage == "" || claim.Step == "" || claim.Reason == "" || claim.UnknownClass == "" || claim.NextOperation == "" || claim.BlockedBy == nil {
		t.Fatalf("unknown fields were not preserved: %+v", claim)
	}

	evidence = testEvidence(t, contract)
	evidence.Cells[0].Decision = "FIXED_POINT"
	refuted := Evaluate(contract, evidence)
	if refuted.Decision != Refuted || !refuted.FailClosed || refuted.Summary.Refuted != 1 {
		t.Fatalf("unexpected fixed-point report: %+v", refuted)
	}
}

func TestRunCannotChangeDenominatorCriteriaOrAuthority(t *testing.T) {
	contract := testContract(1)
	cases := []struct {
		name   string
		mutate func(*Evidence)
		reason string
	}{
		{name: "denominator", mutate: func(e *Evidence) { e.Run.CellIDsAtEnd = e.Run.CellIDsAtEnd[:5] }, reason: "DENOMINATOR_REDUCED_DURING_RUN"},
		{name: "criteria", mutate: func(e *Evidence) { e.Run.SuccessBeforeChange = true; e.Run.CriteriaDigestAtEnd = "criteria:v2" }, reason: "CRITERIA_CHANGED_AFTER_SUCCESS"},
		{name: "authority", mutate: func(e *Evidence) { e.Run.PrivilegeEscalationRequested = true }, reason: "PRIVILEGE_ESCALATION_REQUESTED"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			evidence := testEvidence(t, contract)
			test.mutate(&evidence)
			report := Evaluate(contract, evidence)
			if report.Decision != Refuted || !report.FailClosed || report.Claim.Reason != test.reason {
				t.Fatalf("unexpected guard report: %+v", report)
			}
		})
	}
}

func TestImprovementNeedsExactPairAndExplicitEvaluator(t *testing.T) {
	pair := ImprovementPair{Schema: PairSchema, ComparisonMethod: ExactBeforeAfter, EvaluatorDecision: ImprovementConfirmed, Before: Snapshot{InputDigest: "input:v1", ToolDigest: "tool:v1", ContractDigest: "contract:v1", ArtifactDigest: "artifact:before", MetricValues: map[string]int{"closed": 5}}, After: Snapshot{InputDigest: "input:v1", ToolDigest: "tool:v1", ContractDigest: "contract:v1", ArtifactDigest: "artifact:after", MetricValues: map[string]int{"closed": 6}}}
	confirmed := AssessImprovement(pair)
	if confirmed.Decision != Closed || !confirmed.ExactPair || !confirmed.Improvement {
		t.Fatalf("explicit improvement was not closed: %+v", confirmed)
	}
	pair.After.InputDigest = "input:v2"
	unknown := AssessImprovement(pair)
	if unknown.Decision != Unknown || unknown.ExactPair || unknown.Claim.Reason != "EXACT_BEFORE_AFTER_PAIR_REQUIRED" {
		t.Fatalf("nonmatching pair was inferred: %+v", unknown)
	}
}

func testContract(version int) Denominator {
	cells := []DenominatorCell{
		newCell("source-foundation-driver", Foundation, Driver),
		newCell("semantic-foundation-outcome", Foundation, Outcome),
		newCell("artifact-coherence-guardrail", Coherence, Guardrail),
		newCell("evaluator-coherence-driver", Coherence, Driver),
		newCell("replay-regression-outcome", Regression, Outcome),
		newCell("no-write-regression-guardrail", Regression, Guardrail),
	}
	if version == 2 {
		cells = []DenominatorCell{
			cells[0],
			newCell("semantic-ir-foundation-outcome", Foundation, Outcome),
			newCell("semantic-generated-foundation-outcome", Foundation, Outcome),
			cells[2], cells[3], cells[5],
			newCell("migration-regression-driver", Regression, Driver),
		}
	}
	return Denominator{Schema: ContractSchema, ContractID: "language-development", Version: version, Description: "fixed language-development denominator", RunPolicy: RunPolicy{}, Cells: cells}
}

func newCell(id string, proof ProofChoice, indicator IndicatorClass) DenominatorCell {
	activity := "Observe_" + strings.ReplaceAll(id, "-", "_")
	return DenominatorCell{ID: id, Stage: string(proof), Step: "EVALUATE_" + id, ProofChoice: proof, IndicatorClass: indicator, MetaActivity: activity, Source: "examples/v1-to-v2/meta.gooo#" + activity, IR: "fixtures/ir/" + id + ".json", GeneratedArtifact: "fixtures/generated/" + id + ".json", Evaluator: "internal/evaluator/" + id, MetricID: "gooo.metric.cell." + id + ".v1", MetricDenominator: 1, DependsOn: []string{}}
}

func testEvidence(t *testing.T, contract Denominator) Evidence {
	t.Helper()
	digest, err := ContractDigest(contract)
	if err != nil {
		t.Fatalf("contract digest: %v", err)
	}
	evidence := Evidence{Schema: EvidenceSchema, ContractID: contract.ContractID, ContractVersion: contract.Version, ContractDigest: digest, InputDigest: "input:language-corpus:v1", ToolDigest: "tool:denominator:v1", CriteriaDigest: "criteria:language-development:v1", Run: RunObservation{DenominatorDigestAtStart: digest, DenominatorDigestAtEnd: digest, CellIDsAtStart: cellIDs(contract), CellIDsAtEnd: cellIDs(contract), CriteriaDigestAtStart: "criteria:language-development:v1", CriteriaDigestAtEnd: "criteria:language-development:v1"}, Cells: []CellEvidence{}}
	for _, cell := range contract.Cells {
		evidence.Cells = append(evidence.Cells, CellEvidence{CellID: cell.ID, MetaActivity: cell.MetaActivity, Source: cell.Source, IR: cell.IR, GeneratedArtifact: cell.GeneratedArtifact, Evaluator: cell.Evaluator, Decision: string(Closed), Claim: Claim{State: Closed, BlockedBy: []string{}}})
	}
	return evidence
}
