#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
work=${2:?CI work directory is required}
output=${3:?metrics output is required}

go_files=$(git -C "$root" ls-files '*.go' | wc -l | tr -d ' ')
gooo_files=$(git -C "$root" ls-files '*.gooo' | wc -l | tr -d ' ')
go_lines=$(git -C "$root" ls-files '*.go' -z | xargs -0 awk '{count += 1} END {print count + 0}')
gooo_lines=$(git -C "$root" ls-files '*.gooo' -z | xargs -0 awk '{count += 1} END {print count + 0}')
regular_files=$(git -C "$root" ls-files | awk '$0 != "README.md" {count += 1} END {print count + 0}')
subdirectories=$(git -C "$root" ls-files | awk -F/ 'NF > 1 {for (i=1; i<NF; i++) {path=""; for (j=1; j<=i; j++) path = path (j == 1 ? "" : "/") $j; seen[path]=1}} END {count=0; for (path in seen) count++; print count + 0}')
report="$work/conformance-report.json"
build_wall_ms=$(jq -r '.wall_ms' "$work/timing/build.json")
test_wall_ms=$(jq -r '.wall_ms' "$work/timing/test.json")
conformance_wall_ms=$(jq -r '.wall_ms' "$work/timing/conformance.json")
peak_rss_kib=$(jq -n --argjson build "$(jq -r '.peak_rss_kib' "$work/timing/build.json")" --argjson test "$(jq -r '.peak_rss_kib' "$work/timing/test.json")" --argjson conformance "$(jq -r '.peak_rss_kib' "$work/timing/conformance.json")" '$build | if $test > . then $test else . end | if $conformance > . then $conformance else . end')
test_total=$(jq -s '[.[] | select(.Action == "run" and .Test != null)] | length' "$work/test.json")
test_failed=$(jq -s '[.[] | select(.Action == "fail" and .Test != null)] | length' "$work/test.json")
test_unknown=$(jq -r '[.cases[] | select(.decision == "UNKNOWN")] | length' "$report")
generated_count=$(jq -r '.generated_artifacts_count' "$report")
generated_bytes=$(jq -r '.generated_artifacts_bytes' "$report")

jq -n \
	--arg schema "gooo/self-rewrite-ci-metrics/v1" \
	--argjson go_files "$go_files" --argjson gooo_files "$gooo_files" \
	--argjson go_physical_lines "$go_lines" --argjson gooo_physical_lines "$gooo_lines" \
	--argjson regular_files "$regular_files" --argjson subdirectories "$subdirectories" \
	--argjson generated_files "$generated_count" --argjson generated_bytes "$generated_bytes" \
	--argjson wall_build "$build_wall_ms" --argjson wall_test "$test_wall_ms" --argjson wall_conformance "$conformance_wall_ms" \
	--argjson peak_rss_kib "$peak_rss_kib" \
	--argjson total "$test_total" --argjson failed "$test_failed" --argjson unknown "$test_unknown" \
	--argjson repository_writes "$(jq -r '.repository_writes' "$report")" \
	'{schema:$schema,inventory:{go_files:$go_files,gooo_files:$gooo_files,go_physical_lines:$go_physical_lines,gooo_physical_lines:$gooo_physical_lines,regular_files:$regular_files,files_total:$regular_files,subdirectories:$subdirectories,root_readme_inventory_excluded:1},generated_artifacts:{files:$generated_files,bytes:$generated_bytes},wall_ms:{compile:$wall_build,build:$wall_build,test:$wall_test,conformance:$wall_conformance},peak_rss_kib:$peak_rss_kib,tests:{total:$total,selected:$total,executed:($total - $failed),reused:0,failed:$failed,unknown:$unknown},repository_writes:$repository_writes,automatic_commit_push_merge:0}' > "$output"
