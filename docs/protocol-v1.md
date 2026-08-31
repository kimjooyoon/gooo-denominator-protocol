# Denominator Protocol v1

## Purpose

The protocol answers a bounded completion question from a declared set of
cells. The denominator is an immutable versioned contract, not a live list that
can shrink as a run progresses.

Each cell declares six independent anchors:

```text
cell → meta_activity → source → semantic IR → generated artifact → evaluator
```

The Go validator requires every anchor and metric ID to be unique across the
active contract. A cell metric has a fixed positive integer denominator and is
never replaced by a percentage. Aggregate reports keep exact integer counts for
the active cells and for the proof/indicator buckets.

## State and precedence

Every active cell is exactly one of:

- `CLOSED`: the evaluator accepted the exact bindings and evidence.
- `UNKNOWN`: the evidence is insufficient; the six claim fields are mandatory:
  `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and
  `blocked_by`.
- `REFUTED`: the evidence or contract contradicts the claim.

The report precedence is `REFUTED > UNKNOWN > CLOSED`. A report includes the
three counts and total, never a combined completion percentage.

## Migration receipts

`v1 → v2` demonstrates all three permitted structural operations:

1. `SPLIT` replaces one historical cell with two new IDs.
2. `ADD` introduces a new cell.
3. `RETIRE` removes one cell only with a closed retirement-evidence record that
   repeats the historical bindings.

The validator rejects an unaccounted addition/removal, reuse of an old ID with
new bindings, a missing reason/proof choice/affected claim, a wrong contract
digest, and an evidence-free retirement. The old v1 contract and its replay
report remain present after v2 is introduced.

## Run guards

The run records the denominator digest and ordered cell IDs at start and end.
Any difference is `REFUTED`, including a denominator lowered after a partial
run. If a run had already succeeded, changing the criteria digest is also
`REFUTED`. A privilege-escalation request is fail-closed under the v1 policy.

## Improvement claims

An improvement is `CLOSED` only when both snapshots have identical input, tool,
and contract digests, the comparison method is `EXACT_BEFORE_AFTER`, and an
explicit evaluator says `IMPROVEMENT_CONFIRMED` or `NO_IMPROVEMENT`. A better
looking number without that exact pair remains `UNKNOWN`.

## CI evidence

The GitHub Actions workflow runs formatting, build, test, vet, race tests, Gooo
binding checks, replay, migration, malformed-input, `FIXED_POINT`, denominator
lowering, post-success criteria change, privilege escalation, and retirement
without evidence. The uploaded artifact contains integer measurements for
repository shape, physical Go/Gooo files and lines (blank and comment lines
included), build/test/conformance wall time and peak RSS, executed/reused/skipped
test events, artifact files/bytes, and repository writes. The root `README.md`
is explicitly excluded from the repository counts.
