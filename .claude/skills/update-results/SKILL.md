---
name: update-results
description: Fetch completed match stats from BBC Sport and update matches.csv for the active tournament. Use when the user wants to update results, pull stats, or sync match data.
user-invocable: true
allowed-tools:
  - Read
  - Edit
  - Bash(curl *)
  - Bash(date *)
  - Bash(find *)
  - Bash(python3 *)
---

# /update-results — Fetch match stats and update matches.csv

Fetches completed match stats from BBC Sport and writes them into the active tournament's `matches.csv`.

Arguments passed: `$ARGUMENTS`

---

## Data formats

### Own goals (`HOME_OG`, `AWAY_OG`)

Format: `N;PLAYER:MINUTE` — total count, then one `PLAYER:MINUTE` pair per own goal.

- `MINUTE` is an integer or `+`-offset string for added time (e.g. `90+2`).
- Player name format: first initial + last name, no space — e.g. `C.Montes`.
- Leave the field **empty** if there are no own goals.
- `HOME_OG` = own goals scored **by home team players** (benefiting the away team).
- `AWAY_OG` = own goals scored **by away team players** (benefiting the home team).

### Red cards (`HOME_RED_CARDS`, `AWAY_RED_CARDS`)

Same format as own goals. Covers straight red cards and second yellows.

---

## Step 1 — Identify the active tournament

1. List directories in `domain/data/tournaments/`.
2. The active tournament is the one whose `matches.csv` date range spans today.
3. If unclear, ask: _"Which tournament? (e.g. 2026-fifa-world-cup)"_

---

## Step 2 — Run the stats fetcher

```bash
python3 .claude/skills/update-results/fetch_stats.py domain/data/tournaments/<tournament>/matches.csv
```

The script:
- Finds rows where `COMPLETED ≠ Y` and kick-off was ≥ 115 minutes ago.
- Fetches BBC Sport scores-fixtures pages (parallel) and parses `window.__INITIAL_DATA__` JSON for scores, goals, own goals, and winner.
- Fetches individual BBC Sport live pages (parallel) and parses `match-lineups` JSON for yellow and red cards.
- Outputs a JSON object to stdout (see below).

**If the output `results` array is empty** and `unmatched` is also empty, tell the user no matches need updating and stop.

### Output schema

```json
{
  "results": [
    {
      "match_id": "K1",
      "home_team_id": "PRT",
      "away_team_id": "COD",
      "home_goals": 1,
      "away_goals": 1,
      "winner_team_id": "",
      "home_yellow_cards": 3,
      "away_yellow_cards": 1,
      "home_og": "",
      "away_og": "",
      "home_red_cards": "",
      "away_red_cards": "1;T.Muharemović:80",
      "source_url": "https://www.bbc.co.uk/sport/football/live/..."
    }
  ],
  "unmatched": ["MATCH_ID"]
}
```

---

## Step 3 — Present a summary for review

Print every result in this format:

```
[MATCH_ID] HOME_TEAM_ID vs AWAY_TEAM_ID
  Score:         HOME_GOALS – AWAY_GOALS
  Winner:        WINNER_TEAM_ID  (or "draw")
  Yellow cards:  HOME_YELLOW_CARDS (home) / AWAY_YELLOW_CARDS (away)
  Own goals:     HOME_OG | AWAY_OG  (or "none")
  Red cards:     HOME_RED_CARDS | AWAY_RED_CARDS  (or "none")
  Source:        SOURCE_URL
```

List any `unmatched` IDs at the end, noting they will not be updated.

---

## Step 4 — Confirm with the user

Ask: _"Do you want me to write these stats to matches.csv? (yes/no)"_

Wait for the response. If no, stop.

---

## Step 5 — Update matches.csv

Pipe the JSON output from Step 2 back into the script with `--write`:

```bash
python3 .claude/skills/update-results/fetch_stats.py domain/data/tournaments/<tournament>/matches.csv --write <<'EOF'
<paste the exact JSON output from Step 2>
EOF
```

The script updates every matched row (`COMPLETED`, `WINNER_TEAM_ID`, goals, cards, OGs) and prints how many rows were changed. `NOTES` and all other fields are preserved.
