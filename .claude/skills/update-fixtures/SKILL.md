---
name: update-fixtures
description: Fetch BBC Sport knockout fixtures and auto-resolve placeholder slots in matches.csv. Tournament-agnostic with flexible CLI.
user-invocable: true
allowed-tools:
  - Read
  - Edit
  - Bash(curl *)
  - Bash(python3 *)
  - Bash(date *)
---

# /update-fixtures — Auto-resolve knockout fixture placeholders

Fetches BBC Sport knockout-stage fixtures (current and next month) and resolves placeholder slots in `matches.csv` into real team IDs where the BBC data confirms them.

Arguments passed: `$ARGUMENTS`

---

## What it does

Knockout-stage fixtures in `matches.csv` contain placeholder slot text in the NOTES field, e.g. "Winner Group C vs Runner-up Group F". Once those groups finish and BBC Sport publishes the knockout draw, the resolved team names appear on BBC's fixtures page. This skill:

1. **Fetches** BBC Sport knockout fixtures from the current and next calendar month.
2. **Matches** BBC events to CSV rows using multiple strategies (see below).
3. **Resolves** any placeholder slots where BBC has shown a confirmed team name.
4. **Validates** (optional) re-fetches and prints a comparison table showing CSV vs BBC state for each knockout fixture.

For later rounds, bracket references (e.g. "Winner R32_M73", "Loser SF1") are resolved directly from `matches.csv` itself once the referenced match completes.

No user prompts. Runs fully autonomously and outputs JSON to stdout.

### Matching strategies (priority order)

When matching BBC fixtures to CSV rows, the script tries these strategies in order:

1. **Placeholder slot matching** — Compares BBC placeholder text (e.g. "2nd Group F") to CSV NOTES field placeholders. Used when one side is a placeholder and the other is a resolved team.
2. **Team anchor matching** — When both BBC sides are resolved, looks for a CSV row with one of those teams already filled in and the other blank, then fills the blank.
3. **Date/time fallback** — If both BBC sides are resolved but no team anchor exists, tries to match by DATE and TIME columns. Fills in both teams if both CSV fields are blank, or fills the blank field if one is already set.

Each update includes a `matched_by` field indicating which strategy was used.

---

## Tournament detection

The tournament is automatically detected from the CSV path (e.g. `domain/data/tournaments/fifa-world-cup/matches.csv` → `fifa-world-cup`). To override, use `--tournament <name>`.

Supported tournaments: `fifa-world-cup`, `uefa-euro` (easily extensible in the config).

---

## Usage

### Fetch & diff (compute what could be updated)

```bash
python3 .claude/skills/update-fixtures/fetch_fixtures.py domain/data/tournaments/<tournament>/matches.csv
```

Outputs JSON with `updates` (safe to apply), `conflicts` (CSV differs from BBC), and `ambiguous` (can't safely resolve).

### Apply updates to CSV

```bash
python3 .claude/skills/update-fixtures/fetch_fixtures.py domain/data/tournaments/<tournament>/matches.csv | \
  python3 .claude/skills/update-fixtures/fetch_fixtures.py domain/data/tournaments/<tournament>/matches.csv --write
```

Or pipe the JSON from fetch mode directly:

```bash
cat updates.json | python3 .claude/skills/update-fixtures/fetch_fixtures.py domain/data/tournaments/<tournament>/matches.csv --write
```

### Validate (cross-reference against BBC)

```bash
python3 .claude/skills/update-fixtures/fetch_fixtures.py domain/data/tournaments/<tournament>/matches.csv --validate
```

Prints a side-by-side table comparing each knockout fixture row's current CSV state against BBC's live fixture page.

### Explicit tournament (optional)

If tournament detection fails, pass `--tournament` explicitly:

```bash
python3 .claude/skills/update-fixtures/fetch_fixtures.py <path> --tournament fifa-world-cup --validate
```

---

## Resolving ambiguous cases

The script correctly refuses to guess when both sides of a BBC fixture are already resolved on CSV (e.g. South Africa vs Canada in R32_M73). These are flagged in the `ambiguous` array with a reason. They should be confirmed manually once before the next run, after which the script will skip them on future invocations (they become "already matched").

---

## Output format

```json
{
  "updates": [
    {
      "match_id": "R32_M74",
      "field": "HOME_TEAM_ID",
      "value": "DEU",
      "team_name": "Germany",
      "matched_by": "placeholder_slot"
    },
    {
      "match_id": "R16_M45",
      "field": "AWAY_TEAM_ID",
      "value": "FRA",
      "team_name": "France",
      "matched_by": "date_time"
    }
  ],
  "conflicts": [
    {
      "match_id": "...",
      "field": "...",
      "existing": "...",
      "bbc_value": "...",
      "bbc_team_name": "..."
    }
  ],
  "ambiguous": [
    {
      "bbc_home": "South Africa",
      "bbc_away": "Canada",
      "reason": "both sides resolved on BBC but no matching CSV row found (by team anchor or date/time)"
    }
  ]
}
```

**`matched_by` values:**
- `placeholder_slot` — Matched via group-stage placeholder text (e.g. "2nd Group F")
- `team_anchor` — Matched via one resolved team already in CSV, other blank
- `date_time` — Matched via DATE and TIME columns
- `bracket_reference` — Matched via internal bracket reference (e.g. "Winner R32_M73")
