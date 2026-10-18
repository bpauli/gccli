#!/usr/bin/env bash
# Compares the gccli binary built from a base ref with the one built from
# the working tree: --help for the root, every group and every leaf, the
# four completion scripts, and a set of bad inputs. stdout, stderr and the
# exit code must match byte for byte. Exits 1 when any of them differ.
#
# usage: tools/parity.sh [base-ref]   (default origin/main)
set -euo pipefail

base_ref=${1:-origin/main}
root=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

ldflags="-X main.version=test -X main.commit=none -X main.date=never"
mkdir -p "$tmp/base" "$tmp/home"
git -C "$root" archive "$base_ref" | tar -x -C "$tmp/base"
(cd "$tmp/base" && go build -ldflags "$ldflags" -o "$tmp/gccli-base" ./cmd/gccli)
(cd "$root" && go build -ldflags "$ldflags" -o "$tmp/gccli-head" ./cmd/gccli)
echo "base: $(git -C "$root" rev-parse --short "$base_ref")  head: working tree"

export HOME="$tmp/home" XDG_CONFIG_HOME="$tmp/home"
unset GCCLI_ACCOUNT GCCLI_DOMAIN GCCLI_JSON GCCLI_PLAIN GCCLI_HEADER GCCLI_COLOR GCCLI_UNITS GCCLI_KEYRING_BACKEND

capture() {
	local out=$1
	shift
	"$@" >"$out.stdout" 2>"$out.stderr" && echo 0 >"$out.code" || echo $? >"$out.code"
}

compared=0
differences=0
compare() {
	compared=$((compared + 1))
	capture "$tmp/b" "$tmp/gccli-base" "$@"
	capture "$tmp/h" "$tmp/gccli-head" "$@"
	for part in stdout stderr code; do
		if ! cmp -s "$tmp/b.$part" "$tmp/h.$part"; then
			differences=$((differences + 1))
			echo "DIFF gccli $* ($part)"
			diff "$tmp/b.$part" "$tmp/h.$part" || true
		fi
	done
}

leaves=$("$tmp/gccli-base" --help | sed -n '/^Commands:/,$p' | sed -nE 's/^  ([a-z0-9-]+( [a-z0-9-]+)*)( {2,}.*)?$/\1/p')
paths=$(printf '%s\n' "$leaves" | awk '{ p = ""; for (i = 1; i <= NF; i++) { p = p (i > 1 ? " " : "") $i; print p } }' | sort -u)
leaf_count=$(printf '%s\n' "$leaves" | wc -l)
path_count=$(printf '%s\n' "$paths" | wc -l)

compare --help
while IFS= read -r path; do
	read -ra argv <<<"$path"
	compare "${argv[@]}" --help
done <<<"$paths"
for shell in bash zsh fish powershell; do
	compare completion "$shell"
done

bad_inputs=(
	"--version"
	"nope"
	"health nope"
	"health steps"
	"activity"
	"devices settings"
	"completion nope"
	"health sleep --nope"
	"devices solar 1 --start bad"
	"activities list --limit x"
	"--json reload 2026-13-45"
	"--plain --header activities"
)
for args in "${bad_inputs[@]}"; do
	read -ra argv <<<"$args"
	compare "${argv[@]}"
done

echo "leaves: $leaf_count  paths with --help: $path_count (plus root)  completion scripts: 4  bad inputs: ${#bad_inputs[@]}"
echo "compared: $compared  differences: $differences"
test "$differences" -eq 0
