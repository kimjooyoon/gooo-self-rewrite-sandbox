#!/usr/bin/env bash
set -euo pipefail

output=${1:?timing output is required}
shift
mkdir -p "$(dirname "$output")"
started=$(date +%s%3N)
set +e
/usr/bin/time -v "$@" 2> >(tee "${output}.time" >&2)
status=$?
set -e
finished=$(date +%s%3N)
wall_ms=$((finished - started))
peak_rss_kib=$(awk -F: '/Maximum resident set size/ {gsub(/ /, "", $2); print $2; exit}' "${output}.time")
peak_rss_kib=${peak_rss_kib:-0}
jq -n --argjson wall_ms "$wall_ms" --argjson peak_rss_kib "$peak_rss_kib" --argjson exit_code "$status" \
	'{wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,exit_code:$exit_code}' > "$output"
exit "$status"
