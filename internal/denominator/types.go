package denominator

import "encoding/json"

const (
	ContractSchema = "gooo/denominator/contract/v1"
	EvidenceSchema = "gooo/denominator/evidence/v1"
	ReceiptSchema  = "gooo/denominator/migration-receipt/v1"
	PairSchema     = "gooo/denominator/improvement-pair/v1"
	ReportSchema   = "gooo/denominator/report/v1"
)

type State string

const (
	Closed  State = "CLOSED"
	Unknown State = "UNKNOWN"
	Refuted State = "REFUTED"
)

type ProofChoice string

const (
	Foundation ProofChoice = "FOUNDATION"
	Coherence  ProofChoice = "COHERENCE"
	Regression ProofChoice = "REGRESSION"
)

type IndicatorClass string

const (
	Driver    IndicatorClass = "DRIVER"
	Outcome   IndicatorClass = "OUTCOME"
	Guardrail IndicatorClass = "GUARDRAIL"
)

type Denominator struct {
	Schema     string       `json:"schema"`
	ContractID string       `json:"contract_id"`
	Version    int          `json:"version"`
	Description string      `json:"description"`
	RunPolicy  RunPolicy    `json:"run_policy"`
	Cells      []DenominatorCell `json:"cells"`
}

type RunPolicy struct {
	AllowDenominatorChangeDuringRun bool `json:"allow_denominator_change_during_run"`
	AllowCriteriaChangeAfterSuccess bool `json:"allow_criteria_change_after_success"`
	AllowPrivilegeEscalation        bool `json:"allow_privilege_escalation"`
}

type DenominatorCell struct {
	ID                 string         `json:"id"`
	Stage              string         `json:"stage"`
	Step               string         `json:"step"`
	ProofChoice        ProofChoice    `json:"proof_choice"`
	IndicatorClass     IndicatorClass `json:"indicator_class"`
	MetaActivity       string         `json:"meta_activity"`
	Source             string         `json:"source"`
	IR                 string         `json:"ir"`
	GeneratedArtifact  string         `json:"generated_artifact"`
	Evaluator          string         `json:"evaluator"`
	MetricID           string         `json:"metric_id"`
	MetricDenominator  int            `json:"metric_denominator"`
	DependsOn          []string       `json:"depends_on"`
}

type Evidence struct {
	Schema         string          `json:"schema"`
	ContractID     string          `json:"contract_id"`
	ContractVersion int            `json:"contract_version"`
	ContractDigest string          `json:"contract_digest"`
	InputDigest    string          `json:"input_digest"`
	ToolDigest     string          `json:"tool_digest"`
	CriteriaDigest string          `json:"criteria_digest"`
	Run            RunObservation `json:"run"`
	Cells          []CellEvidence `json:"cells"`
}

type RunObservation struct {
	DenominatorDigestAtStart string   `json:"denominator_digest_at_start"`
	DenominatorDigestAtEnd   string   `json:"denominator_digest_at_end"`
	CellIDsAtStart           []string `json:"cell_ids_at_start"`
	CellIDsAtEnd             []string `json:"cell_ids_at_end"`
	CriteriaDigestAtStart    string   `json:"criteria_digest_at_start"`
	CriteriaDigestAtEnd      string   `json:"criteria_digest_at_end"`
	SuccessBeforeChange      bool     `json:"success_before_change"`
	PrivilegeEscalationRequested bool `json:"privilege_escalation_requested"`
}

type CellEvidence struct {
	CellID            string `json:"cell_id"`
	MetaActivity      string `json:"meta_activity"`
	Source            string `json:"source"`
	IR                string `json:"ir"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator         string `json:"evaluator"`
	Decision          string `json:"decision"`
	Claim             Claim  `json:"claim"`
}

type Claim struct {
	State        State    `json:"state"`
	Stage        string   `json:"stage"`
	Step         string   `json:"step"`
	Reason       string   `json:"reason"`
	UnknownClass string   `json:"unknown_class"`
	NextOperation string `json:"next_operation"`
	BlockedBy    []string `json:"blocked_by"`
}

type Summary struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type CellResult struct {
	CellID        string `json:"cell_id"`
	ProofChoice   ProofChoice `json:"proof_choice"`
	IndicatorClass IndicatorClass `json:"indicator_class"`
	State         State `json:"state"`
	Claim         Claim `json:"claim"`
}

type MetricResult struct {
	ID                string         `json:"id"`
	CellID            string         `json:"cell_id"`
	Numerator         int            `json:"numerator"`
	Denominator       int            `json:"denominator"`
	State             State          `json:"state"`
	MetaActivity      string         `json:"meta_activity"`
	Source            string         `json:"source"`
	IR                string         `json:"ir"`
	GeneratedArtifact string         `json:"generated_artifact"`
	Evaluator         string         `json:"evaluator"`
}

type BucketResult struct {
	Name    string `json:"name"`
	Total   int    `json:"total"`
	Closed  int    `json:"closed"`
	Unknown int    `json:"unknown"`
	Refuted int    `json:"refuted"`
}

type Report struct {
	Schema         string         `json:"schema"`
	Decision       State          `json:"decision"`
	FailClosed     bool           `json:"fail_closed"`
	ContractID     string         `json:"contract_id"`
	ContractVersion int            `json:"contract_version"`
	ContractDigest string         `json:"contract_digest"`
	InputDigest    string         `json:"input_digest"`
	ToolDigest     string         `json:"tool_digest"`
	Summary        Summary        `json:"summary"`
	Cells          []CellResult   `json:"cells"`
	Metrics        []MetricResult `json:"metrics"`
	Proofs         []BucketResult `json:"proofs"`
	Indicators     []BucketResult `json:"indicators"`
	Claim          Claim          `json:"claim"`
}

type MigrationReceipt struct {
	Schema             string              `json:"schema"`
	MigrationID        string              `json:"migration_id"`
	FromContractID     string              `json:"from_contract_id"`
	FromVersion        int                 `json:"from_version"`
	FromContractDigest string              `json:"from_contract_digest"`
	ToContractID       string              `json:"to_contract_id"`
	ToVersion          int                 `json:"to_version"`
	ToContractDigest   string              `json:"to_contract_digest"`
	Reason             string              `json:"reason"`
	ProofChoice        ProofChoice         `json:"proof_choice"`
	AffectedClaims     []string            `json:"affected_claims"`
	Operations         []MigrationOperation `json:"operations"`
}

type MigrationOperation struct {
	Kind             string              `json:"kind"`
	SourceCellID     string              `json:"source_cell_id"`
	TargetCellIDs    []string            `json:"target_cell_ids"`
	Reason           string              `json:"reason"`
	ProofChoice      ProofChoice         `json:"proof_choice"`
	AffectedClaims   []string            `json:"affected_claims"`
	RetirementEvidence *RetirementEvidence `json:"retirement_evidence"`
}

type RetirementEvidence struct {
	CellID            string `json:"cell_id"`
	Decision          string `json:"decision"`
	Reason            string `json:"reason"`
	MetaActivity      string `json:"meta_activity"`
	Source            string `json:"source"`
	IR                string `json:"ir"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator         string `json:"evaluator"`
}

type MigrationReport struct {
	Schema         string `json:"schema"`
	Decision       State  `json:"decision"`
	FailClosed     bool   `json:"fail_closed"`
	MigrationID    string `json:"migration_id"`
	FromVersion    int    `json:"from_version"`
	ToVersion      int    `json:"to_version"`
	Added          int    `json:"added"`
	Split          int    `json:"split"`
	Retired        int    `json:"retired"`
	Claim          Claim  `json:"claim"`
}

type ImprovementPair struct {
	Schema            string   `json:"schema"`
	ComparisonMethod  string   `json:"comparison_method"`
	EvaluatorDecision string   `json:"evaluator_decision"`
	Before            Snapshot `json:"before"`
	After             Snapshot `json:"after"`
}

type Snapshot struct {
	InputDigest    string         `json:"input_digest"`
	ToolDigest     string         `json:"tool_digest"`
	ContractDigest string         `json:"contract_digest"`
	ArtifactDigest string         `json:"artifact_digest"`
	MetricValues   map[string]int `json:"metric_values"`
}

type ImprovementReport struct {
	Schema           string `json:"schema"`
	Decision         State  `json:"decision"`
	FailClosed       bool   `json:"fail_closed"`
	ExactPair        bool   `json:"exact_pair"`
	Improvement      bool   `json:"improvement"`
	Claim            Claim  `json:"claim"`
}

func (s State) Valid() bool {
	return s == Closed || s == Unknown || s == Refuted
}

func (p ProofChoice) Valid() bool {
	return p == Foundation || p == Coherence || p == Regression
}

func (i IndicatorClass) Valid() bool {
	return i == Driver || i == Outcome || i == Guardrail
}

func (c Claim) MarshalJSON() ([]byte, error) {
	type alias Claim
	if c.BlockedBy == nil {
		c.BlockedBy = []string{}
	}
	return json.Marshal(alias(c))
}
