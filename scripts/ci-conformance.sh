#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
work=${2:?CI work directory is required}
binary=${3:?sandbox binary is required}
mkdir -p "$work"
report="$work/actuation-report.json"
input_root="$root/fixtures/input/stage-n"
before=$(find "$input_root" -type f -print0 | sort -z | xargs -0 sha256sum)
"$binary" run \
	--meta "$root/meta/self-rewrite-boundary.gooo" \
	--input-root "$input_root" \
	--phase "$input_root/phase.gooo" \
	--source "$input_root/source.gooo" \
	--corpus "$root/fixtures/corpus/corpus.json" \
	--work "$work/actuation-work" \
	--report "$report" > "$work/actuation-stdout.json"
after=$(find "$input_root" -type f -print0 | sort -z | xargs -0 sha256sum)
test "$before" = "$after"

jq -e '
	.schema == "gooo/self-rewrite-actuation-report/v1" and
	.decision == "CLOSED" and
	.repository_writes == 0 and
	.automatic_commit_push_merge == 0 and
	(.cases | length == 6) and
	([.cases[] | .match] | all) and
	([.cases[] | {id, expected, decision}] | sort_by(.id)) == [
		{id:"BYTE_IDENTICAL_REPLAY_CLOSED",expected:"CLOSED",decision:"CLOSED"},
		{id:"CHANGED_TERMINAL_REASON_REFUTED",expected:"REFUTED",decision:"REFUTED"},
		{id:"EXPLANATION_PRESERVING_CLOSED",expected:"CLOSED",decision:"CLOSED"},
		{id:"FORBIDDEN_EFFECT_REFUTED",expected:"REFUTED",decision:"REFUTED"},
		{id:"INCOMPLETE_BINDING_UNKNOWN",expected:"UNKNOWN",decision:"UNKNOWN"},
		{id:"SEMANTICS_PRESERVING_CLOSED",expected:"CLOSED",decision:"CLOSED"}
	] and
	([.cases[] | select(.id == "SEMANTICS_PRESERVING_CLOSED") | .candidate_applied] == [true]) and
	([.cases[] | select(.id == "EXPLANATION_PRESERVING_CLOSED") | .candidate_applied] == [true]) and
	([.cases[] | select(.id == "CHANGED_TERMINAL_REASON_REFUTED") | .candidate_applied] == [true]) and
	([.cases[] | select(.id == "FORBIDDEN_EFFECT_REFUTED") | .candidate_applied] == [false]) and
	([.cases[] | select(.id == "INCOMPLETE_BINDING_UNKNOWN") | .candidate_applied] == [false]) and
	([.cases[] | select(.id == "BYTE_IDENTICAL_REPLAY_CLOSED") | .candidate_applied] == [true]) and
	([.cases[] | select(.id == "INCOMPLETE_BINDING_UNKNOWN") | .unknowns[0] | keys] == [["blocked_by","next_operation","reason","stage","step","unknown_class"]]) and
	([.cases[] | select(.id == "FORBIDDEN_EFFECT_REFUTED") | .refutations[] | select(.reason == "FORBIDDEN_EFFECT") | .counterexample] == ["input-repository.write"]) and
	([.cases[] | select(.id == "CHANGED_TERMINAL_REASON_REFUTED") | .refutations[] | select(.reason == "TERMINAL_REASON_CHANGED") | .reason] == ["TERMINAL_REASON_CHANGED"]) and
	([.cases[] | select(.id == "BYTE_IDENTICAL_REPLAY_CLOSED") | .baseline.semantic_digest == .candidate.semantic_digest and .baseline.generated_digest == .candidate.generated_digest and .baseline.terminal_digest == .candidate.terminal_digest] == [true]) and
	([.cases[] | select(.id == "SEMANTICS_PRESERVING_CLOSED") | .baseline.semantic_digest == .candidate.semantic_digest] == [true]) and
	([.cases[] | select(.id == "EXPLANATION_PRESERVING_CLOSED") | .baseline.semantic_digest == .candidate.semantic_digest and .baseline.terminal_digest == .candidate.terminal_digest] == [true]) and
	([.cases[] | select(.id == "CHANGED_TERMINAL_REASON_REFUTED") | .baseline.terminal.reason != .candidate.terminal.reason] == [true])
' "$report" >/dev/null

cp "$report" "$work/conformance-report.json"
cat > "$work/step-summary.md" <<EOF
## Self-rewrite sandbox conformance

- decision: $(jq -r '.decision' "$report")
- fixed cases: $(jq -r '.cases | length' "$report")
- repository_writes: $(jq -r '.repository_writes' "$report")
- automatic_commit_push_merge: $(jq -r '.automatic_commit_push_merge' "$report")
- generated_artifacts: $(jq -r '.generated_artifacts_count' "$report") files / $(jq -r '.generated_artifacts_bytes' "$report") bytes
EOF
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
	cat "$work/step-summary.md" >> "$GITHUB_STEP_SUMMARY"
fi
