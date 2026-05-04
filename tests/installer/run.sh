#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGES=(ubuntu-bash ubuntu-zsh ubuntu-fish debian arch fedora)

# Allow caller to limit the matrix: `run.sh ubuntu-bash debian`
if [ "$#" -gt 0 ]; then
    IMAGES=("$@")
fi

failures=()
for img in "${IMAGES[@]}"; do
    dockerfile="$ROOT/tests/installer/docker/${img}.Dockerfile"
    if [ ! -f "$dockerfile" ]; then
        echo "skipping ${img}: Dockerfile not found"
        continue
    fi
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo "  matrix cell: ${img}"
    echo "════════════════════════════════════════════════════════════════"
    tag="spotnik-installer-test-${img}"
    docker build --quiet -f "$dockerfile" -t "$tag" "$ROOT" >/dev/null
    if ! docker run --rm "$tag"; then
        failures+=("$img")
    fi
done

echo ""
if [ "${#failures[@]}" -gt 0 ]; then
    echo "✗ failed cells: ${failures[*]}" >&2
    exit 1
fi
echo "✓ all cells passed"
