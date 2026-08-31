package denominator

const (
	ExactBeforeAfter     = "EXACT_BEFORE_AFTER"
	ImprovementConfirmed = "IMPROVEMENT_CONFIRMED"
	NoImprovement        = "NO_IMPROVEMENT"
)

func AssessImprovement(pair ImprovementPair) ImprovementReport {
	report := ImprovementReport{Schema: PairSchema, Improvement: false}
	if pair.Schema != PairSchema {
		return unknownImprovement(report, "PAIR", "VALIDATE_PAIR", "MALFORMED_IMPROVEMENT_PAIR", "RESTORE_IMPROVEMENT_PAIR")
	}
	if pair.ComparisonMethod != ExactBeforeAfter {
		return unknownImprovement(report, "PAIR", "REQUIRE_EXACT_BEFORE_AFTER", "EXACT_BEFORE_AFTER_PAIR_REQUIRED", "PROVIDE_EXACT_BEFORE_AFTER_PAIR")
	}
	if !snapshotComplete(pair.Before) || !snapshotComplete(pair.After) {
		return unknownImprovement(report, "PAIR", "BIND_SNAPSHOT_DIGESTS", "EXACT_BEFORE_AFTER_PAIR_REQUIRED", "PROVIDE_INPUT_TOOL_CONTRACT_AND_ARTIFACT_DIGESTS")
	}
	if pair.Before.InputDigest != pair.After.InputDigest || pair.Before.ToolDigest != pair.After.ToolDigest || pair.Before.ContractDigest != pair.After.ContractDigest {
		return unknownImprovement(report, "PAIR", "COMPARE_STABLE_DIGESTS", "EXACT_BEFORE_AFTER_PAIR_REQUIRED", "REPLAY_WITH_IDENTICAL_INPUT_TOOL_AND_CONTRACT")
	}
	report.ExactPair = true
	switch pair.EvaluatorDecision {
	case ImprovementConfirmed:
		report.Decision = Closed
		report.Improvement = true
		report.Claim = Claim{State: Closed, Reason: "EXPLICIT_EVALUATOR_CONFIRMED_IMPROVEMENT", NextOperation: "RETAIN_EXACT_PAIR", BlockedBy: []string{}}
	case NoImprovement:
		report.Decision = Closed
		report.Claim = Claim{State: Closed, Reason: "EXPLICIT_EVALUATOR_FOUND_NO_IMPROVEMENT", NextOperation: "RETAIN_EXACT_PAIR", BlockedBy: []string{}}
	default:
		return unknownImprovement(report, "PAIR", "READ_EVALUATOR_DECISION", "IMPROVEMENT_EVALUATOR_UNAVAILABLE", "PROVIDE_EXPLICIT_IMPROVEMENT_EVALUATOR")
	}
	return report
}

func snapshotComplete(snapshot Snapshot) bool {
	return snapshot.InputDigest != "" && snapshot.ToolDigest != "" && snapshot.ContractDigest != "" && snapshot.ArtifactDigest != ""
}

func unknownImprovement(report ImprovementReport, stage, step, reason, next string) ImprovementReport {
	report.Decision = Unknown
	report.ExactPair = false
	report.Claim = Claim{State: Unknown, Stage: stage, Step: step, Reason: reason, UnknownClass: "DIRECT_MISSING", NextOperation: next, BlockedBy: []string{}}
	return report
}
