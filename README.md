# Gooo Denominator Protocol

The Gooo Denominator Protocol is a versioned, evidence-backed contract for
answering a bounded question such as “how many language-development cells are
complete?” without guessing.

The protocol keeps every denominator version immutable. Adding, splitting, or
retiring a cell creates a migration receipt with a reason, proof choice, and
the claims affected by the change. A run reports exact `CLOSED`, `UNKNOWN`,
and `REFUTED` counts; it never collapses them into a percentage.

The implementation and fixtures are intentionally self-contained. They model
the one-to-one chain required for every cell:

```text
Gooo meta activity → source evidence → semantic IR → generated artifact → evaluator
```

The first conformance fixture migrates denominator `v1` to `v2`, replays both
versions, and exercises normal, UNKNOWN, and REFUTED cases. CI is the authority
for formatting, tests, vetting, replay, and runtime measurements.

## Status vocabulary

`CLOSED` means the evaluator has exact, contract-bound evidence. `UNKNOWN`
means the claim cannot be decided and preserves `stage`, `step`, `reason`,
`unknown_class`, `next_operation`, and `blocked_by`. `REFUTED` means the
contract or evidence proves that the claim is invalid. Decision precedence is
`REFUTED` over `UNKNOWN` over `CLOSED`.

The protocol rejects denominator reduction during a run, post-success criteria
changes, evidence-free retirement, malformed receipts, the legacy
`FIXED_POINT` decision, and privilege-escalation requests.

## Development

Use the GitHub Actions workflow as the source of validation. Local execution of
build, test, formatting, and vetting is deliberately excluded from the
conformance procedure so the published artifact records CI measurements.
