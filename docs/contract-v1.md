# Actuation contract v1

This repository is a sandbox boundary, not a self-authorizing updater.

## Inputs

Each run receives four independent inputs: an immutable compiler-input root, a
typed `.gooo` candidate, the authoritative `.gooo` actuation contract, and a
fixed fixture corpus. The input root contains the stage-N compiler phase and a
source `.gooo` program. Its complete file snapshot is compared after the run.

## Candidate lifecycle

1. Parse the candidate as a typed rewrite and bind its target phase/activity.
2. Reject forbidden effects and undeclared capabilities before actuation.
3. Copy the input root to a caller-owned temporary directory.
4. Apply the candidate only inside that copy.
5. Lower the copied phase and source into semantic IR, generate `generated.go`,
   and execute that generated artifact.
6. Compare stage N and stage N+1 according to the case policy.

Valid candidates therefore produce a real next-generation artifact, while an
incomplete binding remains `UNKNOWN` and a forbidden effect remains
`REFUTED`. No branch of the runner has promotion authority.

## Acceptance and improvement

Semantic acceptance requires canonical semantic IR equality. Explanation
acceptance additionally requires terminal-record equality. Replay acceptance
requires byte equality for semantic IR, generated Go, and terminal record. A
changed terminal reason and every forbidden effect are explicit
counterexamples. Exact before/after improvement evidence requires equality of
scenario, source digest, contract digest, and toolchain identity.
