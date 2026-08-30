#!/usr/bin/env bash
set -euo pipefail

bin=$1
root=$2
out=$3
mkdir -p "$out/cases"

contract_v1="$root/contracts/denominator-v1.json"
contract_v2="$root/contracts/denominator-v2.json"
receipt="$root/contracts/migration-v1-v2.json"
evidence_v1="$root/fixtures/replay/evidence-v1.json"
evidence_v2="$root/fixtures/replay/evidence-v2.json"
pair="$root/fixtures/replay/improvement-exact.json"

"$bin" migrate --from "$contract_v1" --to "$contract_v2" --receipt "$receipt" --output "$out/cases/migration-v1-v2.json"
jq -e '.decision=="CLOSED" and .added==1 and .split==1 and .retired==1 and .fail_closed==false' "$out/cases/migration-v1-v2.json" >/dev/null

"$bin" evaluate --contract "$contract_v1" --evidence "$evidence_v1" --output "$out/cases/replay-v1.json"
jq -e '.decision=="CLOSED" and .fail_closed==false and .summary=={total:6,closed:6,unknown:0,refuted:0} and (.metrics|length)==6 and ([.metrics[]|select(.denominator==1 and .meta_activity!="" and .source!="" and .ir!="" and .generated_artifact!="" and .evaluator!="")]|length)==6' "$out/cases/replay-v1.json" >/dev/null

"$bin" evaluate --contract "$contract_v2" --evidence "$evidence_v2" --output "$out/cases/replay-v2.json"
jq -e '.decision=="CLOSED" and .fail_closed==false and .summary=={total:7,closed:7,unknown:0,refuted:0} and (.metrics|length)==7 and ([.cells[]|select(.proof_choice=="FOUNDATION")]|length)==3 and ([.cells[]|select(.proof_choice=="COHERENCE")]|length)==2 and ([.cells[]|select(.proof_choice=="REGRESSION")]|length)==2 and ([.cells[]|select(.indicator_class=="DRIVER")]|length)==3 and ([.cells[]|select(.indicator_class=="OUTCOME")]|length)==2 and ([.cells[]|select(.indicator_class=="GUARDRAIL")]|length)==2' "$out/cases/replay-v2.json" >/dev/null

jq 'del(.cells[2])' "$evidence_v1" > "$out/cases/evidence-unknown.json"
"$bin" evaluate --contract "$contract_v1" --evidence "$out/cases/evidence-unknown.json" --output "$out/cases/replay-unknown.json"
jq -e '.decision=="UNKNOWN" and .fail_closed==false and .summary=={total:6,closed:5,unknown:1,refuted:0} and ([.cells[]|select(.state=="UNKNOWN" and .claim.stage!="" and .claim.step!="" and .claim.reason!="" and .claim.unknown_class!="" and .claim.next_operation!="" and (.claim.blocked_by|type)=="array")]|length)==1' "$out/cases/replay-unknown.json" >/dev/null

jq '(.cells[0].decision)="FIXED_POINT"' "$evidence_v1" > "$out/cases/evidence-fixed-point.json"
set +e
"$bin" evaluate --contract "$contract_v1" --evidence "$out/cases/evidence-fixed-point.json" --output "$out/cases/replay-fixed-point.json"
status=$?
set -e
test "$status" -ne 0
jq -e '.decision=="REFUTED" and .fail_closed==true and .summary.refuted==1 and .claim.reason=="UNRECOGNIZED_DECISION_FIXED_POINT"' "$out/cases/replay-fixed-point.json" >/dev/null

jq '(.run.cell_ids_at_end |= map(select(. != "no-write-regression-guardrail")))' "$evidence_v1" > "$out/cases/evidence-lowered-denominator.json"
set +e
"$bin" evaluate --contract "$contract_v1" --evidence "$out/cases/evidence-lowered-denominator.json" --output "$out/cases/replay-lowered-denominator.json"
status=$?
set -e
test "$status" -ne 0
jq -e '.decision=="REFUTED" and .fail_closed==true and .summary=={total:6,closed:0,unknown:0,refuted:6} and .claim.reason=="DENOMINATOR_REDUCED_DURING_RUN"' "$out/cases/replay-lowered-denominator.json" >/dev/null

jq '(.run.success_before_change)=true | (.run.criteria_digest_at_end)="criteria:language-development:v2"' "$evidence_v1" > "$out/cases/evidence-criteria-change.json"
set +e
"$bin" evaluate --contract "$contract_v1" --evidence "$out/cases/evidence-criteria-change.json" --output "$out/cases/replay-criteria-change.json"
status=$?
set -e
test "$status" -ne 0
jq -e '.decision=="REFUTED" and .fail_closed==true and .claim.reason=="CRITERIA_CHANGED_AFTER_SUCCESS"' "$out/cases/replay-criteria-change.json" >/dev/null

jq '(.run.privilege_escalation_requested)=true' "$evidence_v1" > "$out/cases/evidence-privilege-escalation.json"
set +e
"$bin" evaluate --contract "$contract_v1" --evidence "$out/cases/evidence-privilege-escalation.json" --output "$out/cases/replay-privilege-escalation.json"
status=$?
set -e
test "$status" -ne 0
jq -e '.decision=="REFUTED" and .fail_closed==true and .claim.reason=="PRIVILEGE_ESCALATION_REQUESTED"' "$out/cases/replay-privilege-escalation.json" >/dev/null

jq '(.operations[2].retirement_evidence)=null' "$receipt" > "$out/cases/receipt-retirement-without-evidence.json"
set +e
"$bin" migrate --from "$contract_v1" --to "$contract_v2" --receipt "$out/cases/receipt-retirement-without-evidence.json" --output "$out/cases/migration-without-retirement-evidence.json"
status=$?
set -e
test "$status" -ne 0
jq -e '.decision=="REFUTED" and .fail_closed==true and .claim.reason=="RETIREMENT_EVIDENCE_MISSING"' "$out/cases/migration-without-retirement-evidence.json" >/dev/null

printf '{' > "$out/cases/malformed.json"
set +e
"$bin" evaluate --contract "$contract_v1" --evidence "$out/cases/malformed.json" --output "$out/cases/replay-malformed.json"
status=$?
set -e
test "$status" -ne 0
jq -e '.decision=="REFUTED" and .fail_closed==true' "$out/cases/replay-malformed.json" >/dev/null

"$bin" improvement --pair "$pair" --output "$out/cases/improvement-exact.json"
jq -e '.decision=="CLOSED" and .exact_pair==true and .improvement==true and .claim.state=="CLOSED"' "$out/cases/improvement-exact.json" >/dev/null

jq '(.after.input_digest)="input:language-corpus:v2"' "$pair" > "$out/cases/improvement-without-exact-pair.json"
"$bin" improvement --pair "$out/cases/improvement-without-exact-pair.json" --output "$out/cases/improvement-unknown.json"
jq -e '.decision=="UNKNOWN" and .exact_pair==false and .claim.reason=="EXACT_BEFORE_AFTER_PAIR_REQUIRED" and .claim.unknown_class!="" and (.claim.blocked_by|type)=="array"' "$out/cases/improvement-unknown.json" >/dev/null

jq -S -n \
	--arg migration "$out/cases/migration-v1-v2.json" \
	--arg replay_v1 "$out/cases/replay-v1.json" \
	--arg replay_v2 "$out/cases/replay-v2.json" \
	--arg unknown "$out/cases/replay-unknown.json" \
	--arg fixed_point "$out/cases/replay-fixed-point.json" \
	--arg lowered "$out/cases/replay-lowered-denominator.json" \
	--arg criteria "$out/cases/replay-criteria-change.json" \
	--arg privilege "$out/cases/replay-privilege-escalation.json" \
	--arg retireless "$out/cases/migration-without-retirement-evidence.json" \
	--arg malformed "$out/cases/replay-malformed.json" \
	--arg improvement "$out/cases/improvement-exact.json" \
	--arg improvement_unknown "$out/cases/improvement-unknown.json" \
	'{schema:"gooo/denominator/conformance/v1",cases:[
		{id:"migration-v1-v2",state:"CLOSED",report:$migration},
		{id:"replay-v1",state:"CLOSED",report:$replay_v1},
		{id:"replay-v2",state:"CLOSED",report:$replay_v2},
		{id:"replay-unknown",state:"UNKNOWN",report:$unknown},
		{id:"replay-fixed-point",state:"REFUTED",report:$fixed_point},
		{id:"replay-lowered-denominator",state:"REFUTED",report:$lowered},
		{id:"replay-criteria-change",state:"REFUTED",report:$criteria},
		{id:"replay-privilege-escalation",state:"REFUTED",report:$privilege},
		{id:"migration-without-retirement-evidence",state:"REFUTED",report:$retireless},
		{id:"replay-malformed",state:"REFUTED",report:$malformed},
		{id:"improvement-exact",state:"CLOSED",report:$improvement},
		{id:"improvement-unknown",state:"UNKNOWN",report:$improvement_unknown}
	]}' > "$out/conformance-index.json"
