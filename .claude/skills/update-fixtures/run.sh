#!/bin/bash
# Wrapper for /update-fixtures skill
# Fetches BBC fixtures, shows updates, and applies them to matches.csv

set -e

# Find the active tournament (look for any tournament directory)
tournament_dir=$(find domain/data/tournaments -maxdepth 1 -type d ! -name "tournaments" | head -1)
if [ -z "$tournament_dir" ]; then
    echo "Error: No tournament found in domain/data/tournaments" >&2
    exit 1
fi

csv="$tournament_dir/matches.csv"
if [ ! -f "$csv" ]; then
    echo "Error: $csv not found" >&2
    exit 1
fi

# Extract tournament name from path
tournament=$(basename "$tournament_dir")

skill_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
py="$skill_dir/fetch_fixtures.py"

# Step 1: Fetch & compute updates
echo "Fetching BBC Sport fixtures for $tournament..."
output=$(python3 "$py" "$csv" --tournament "$tournament")

# Step 2: Parse and display
updates=$(echo "$output" | jq -r '.updates | length')
conflicts=$(echo "$output" | jq -r '.conflicts | length')
ambiguous=$(echo "$output" | jq -r '.ambiguous | length')

if [ "$updates" -eq 0 ] && [ "$conflicts" -eq 0 ] && [ "$ambiguous" -eq 0 ]; then
    echo "No updates found. matches.csv is up to date."
    exit 0
fi

if [ "$updates" -gt 0 ]; then
    echo ""
    echo "Found $updates update(s):"
    echo "$output" | jq -r '.updates[] | "  [\(.match_id)] \(.field) = \(.value) (\(.team_name))"'
fi

if [ "$conflicts" -gt 0 ]; then
    echo ""
    echo "Found $conflicts conflict(s) (CSV differs from BBC):"
    echo "$output" | jq -r '.conflicts[] | "  [\(.match_id)] \(.field): \(.existing) → \(.bbc_value) (\(.bbc_team_name))"'
fi

if [ "$ambiguous" -gt 0 ]; then
    echo ""
    echo "Found $ambiguous ambiguous case(s) (skipped for safety):"
    echo "$output" | jq -r '.ambiguous[] | "  BBC: \(.bbc_home) vs \(.bbc_away) — \(.reason)"'
fi

# Step 3: Apply updates
if [ "$updates" -gt 0 ]; then
    echo ""
    echo "Applying updates to $csv..."
    echo "$output" | python3 "$py" "$csv" --tournament "$tournament" --write
fi

# Step 4: Validate
echo ""
echo "Cross-reference validation:"
python3 "$py" "$csv" --tournament "$tournament" --validate
