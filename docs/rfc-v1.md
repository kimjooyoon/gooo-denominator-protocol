# RFC: Versioned Denominator Meta-Protocol

Status: implemented in this repository.

The protocol exists to make a claim like “6 of 6 language-development cells are
complete” reproducible. It does not estimate unobserved work, infer utility
from a favorable metric, or rewrite history when the denominator changes.

## Contract law

For a contract `D_v`, the active denominator is the exact ordered set
`D_v.cells`. Every cell carries a fixed integer metric denominator and a unique
Gooo meta activity, source selector, semantic IR selector, generated-artifact
selector, and evaluator selector. The evaluator may only emit `CLOSED`,
`UNKNOWN`, or `REFUTED`.

## Receipt law

For a migration receipt `R(v,w)`, both contract digests and versions are bound.
Every ID that disappears or appears between the two snapshots must occur in
exactly one explicit `SPLIT`, `ADD`, or `RETIRE` operation. `RETIRE` requires
closed evidence of the historical binding. An old version is never rewritten.

## Run law

Start and end denominator digests and cell ID lists must equal the contract
being evaluated. Criteria changes after success and privilege escalation are
fail-closed. A malformed JSON document or unrecognized `FIXED_POINT` result is
not a soft unknown.

## Comparison law

The improvement evaluator compares an explicit before/after pair. Input, tool,
and contract digests must be equal and an evaluator decision must be present.
Metric values are carried as evidence only; the protocol never promotes a
larger value to an improvement by itself.
