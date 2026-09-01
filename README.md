# gooo-self-rewrite-sandbox

An independent, promotion-free actuation boundary for applying a typed `.gooo`
compiler rewrite to a caller-owned temporary copy and producing a real next
generation Go artifact.

The semantic authority is [`meta/self-rewrite-boundary.gooo`](meta/self-rewrite-boundary.gooo).
Go implements only parsing/lowering, temporary-copy actuation, generated-artifact
execution, and evidence comparison. The input repository is snapshotted before
and after every run; a non-zero write count is a refutation. No code in this
repository can commit, push, merge, or promote a candidate.

## Fixed denominator

The corpus has exactly six cases:

| Case | Expected decision |
| --- | --- |
| semantics-preserving CLOSED | `CLOSED` |
| explanation-preserving CLOSED | `CLOSED` |
| changed terminal reason | `REFUTED` |
| forbidden effect | `REFUTED` |
| incomplete binding | `UNKNOWN` |
| byte-identical replay | `CLOSED` |

`UNKNOWN` always preserves `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`. Decisions reduce in the fixed order
`REFUTED > UNKNOWN > CLOSED`. Improvement evidence closes only for an exact
scenario/source/contract/toolchain before/after pair; otherwise it is
`UNKNOWN`.

## Verification boundary

GitHub Actions is the validation authority. It runs Go 1.27, formatting, vet,
build, tests, and the six-case conformance workflow, then uploads the exact
integer inventory, wall time, peak RSS, generated artifact count/bytes, and
test counts. Local test, build, vet, and conformance execution is intentionally
not part of the development protocol.

The optional oracle lock pins the immutable
`gooo-reflexive-compiler-slice v0.3.0` and
`gooo-two-generation-bootstrap v0.1.1` releases for later comparison. It does
not create a required cross-project gate.

## CI command

The workflow invokes the CLI with the immutable fixture root, the authoritative
meta phase, the fixed corpus, and a caller-owned `.ci` workspace. The report is
written only to that workspace.
