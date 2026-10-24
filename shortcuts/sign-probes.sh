#!/bin/bash
# Sign the probe shortcuts (probes.py) to find which action Apple rejects.
# Run on a Mac (macOS 12 or later) signed in to iCloud:   ./sign-all.sh
# Works with the stock /bin/bash 3.2.

here="$(cd "$(dirname "$0")" && pwd)"
in_dir="$here/probes/unsigned"
out_dir="$here/probes/signed"

if [ "$(uname -s)" != "Darwin" ]; then
    echo "error: this has to run on a Mac (the 'shortcuts' command only exists on macOS)." >&2
    exit 1
fi
if ! command -v shortcuts >/dev/null 2>&1; then
    echo "error: the 'shortcuts' command was not found. It ships with macOS 12 (Monterey) or later." >&2
    exit 1
fi

shopt -s nullglob
files=("$in_dir"/*.shortcut)
if [ ${#files[@]} -eq 0 ]; then
    echo "error: no .shortcut files in $in_dir" >&2
    echo "       build them first:  python3 -I \"$here/build.py\"" >&2
    exit 1
fi

mkdir -p "$out_dir" || { echo "error: cannot create $out_dir" >&2; exit 1; }

ok=0
failed=0
for f in "${files[@]}"; do
    name="$(basename "$f")"
    out="$out_dir/$name"
    rm -f "$out"
    if log="$(shortcuts sign --mode anyone --input "$f" --output "$out" 2>&1)" && [ -s "$out" ]; then
        echo "signed   $name"
        ok=$((ok + 1))
    else
        echo "FAILED   $name" >&2
        if [ -n "$log" ]; then
            printf '%s\n' "$log" | sed 's/^/         /' >&2
        else
            echo "         (no output file and no message from 'shortcuts sign')" >&2
        fi
        rm -f "$out"
        failed=$((failed + 1))
    fi
done

echo
echo "$ok signed, $failed failed. Signed files are in: $out_dir"
if [ "$failed" -gt 0 ]; then
    echo "If every file failed: make sure this Mac is signed in to iCloud (System Settings > Apple Account)" >&2
    echo "and that you can open the Shortcuts app. Signing needs Apple's servers, so it needs internet." >&2
    exit 1
fi
echo "Next: AirDrop the files in signed/ to the iPhone (see README.md)."
