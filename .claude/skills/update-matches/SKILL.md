---
name: update-matches
description: Fetch completed match stats from BBC Sport and update matches.csv for the active tournament. Use when the user wants to update results, pull stats, or sync match data.
user-invocable: true
allowed-tools:
  - Read
  - Edit
  - WebFetch
  - Bash(date *)
  - Bash(find *)
---

# /update-matches — Fetch match stats and update matches.csv

Fetches completed match stats from BBC Sport and writes them into the active tournament's `matches.csv`.

Arguments passed: `$ARGUMENTS`

---

## Data formats

### Own goals (`HOME_OG`, `AWAY_OG`)

Format: `N;PLAYER:MINUTE` where `N` is the total number of own goals, followed by one `PLAYER:MINUTE` pair per own goal, all separated by `;`.

- `MINUTE` is the match minute as an integer, or a string with a `+` offset for added time (e.g. `90+2`).
- Leave the field **empty** if there are no own goals.

Examples:
- `1;Johnson:32` — one own goal by Johnson at minute 32
- `2;Smith:12;Jones:67` — two own goals
- `3;Doe:53;Reed:45+6;Benson:88` — three own goals, one in added time

### Red cards (`HOME_RED_CARDS`, `AWAY_RED_CARDS`)

Same format as own goals, but for players sent off (straight red card, or second yellow card).

- `1;C.Montes:90+2` — one red card
- `2;Y.Sithole:49;T.Zwane:84` — two red cards

---

## Step 1 — Identify the active tournament

1. Read all directories in `domain/data/tournaments/*`.
2. Identify the active tournament by reading the year from the folder name (e.g. `2026-fifa-world-cup` is 2026) and checking the `matches.csv` file.
3. The active tournament is the one whose match dates span today's date (i.e. today falls between the earliest and latest match dates in its `matches.csv`).
4. If no tournament can be confidently identified, ask the user: _"Which tournament should I update? (e.g. 2026-fifa-world-cup)"_ and use their answer as the tournament directory name.

---

## Step 2 — Identify matches to update

1. Parse the active tournament's `matches.csv`.
2. Select rows where:
   - `COMPLETED` is empty or `"N"` (i.e. not `"Y"`), **and**
   - The match kick-off date and time (columns `DATE` and `TIME`) indicate the match has plausibly finished.
3. To assess whether a match has finished:
   - Parse `DATE` as `DD/MM/YYYY` and `TIME` as `HH:MM`.
   - Times are almost always in **UTC** or **BST** (BST = UTC+1, in effect late March–late October). Treat them as local UK time.
   - A match is considered likely finished if the current time is at least **115 minutes** after the stated kick-off time (90 min match + 25 min buffer for stoppages and half-time).
4. Collect the `HOME_TEAM_ID` and `AWAY_TEAM_ID` for each candidate match — you will use these to identify the correct match on BBC Sport.

If no matches are found that meet these criteria, tell the user and stop.

---

## Step 3 — Locate individual match pages on BBC Sport

Fetch the following three BBC Sport scores/fixtures pages (substituting the actual dates):

- `https://www.bbc.co.uk/sport/football/scores-fixtures/YYYY-MM-DD` (yesterday)
- `https://www.bbc.co.uk/sport/football/scores-fixtures/YYYY-MM-DD` (today)
- `https://www.bbc.co.uk/sport/football/scores-fixtures/YYYY-MM-DD` (tomorrow)

From each page, find hyperlinks to individual match report/summary pages. Match each link to one of the candidate matches from Step 2 by identifying the two competing teams.

Record the URL for each matched individual match page. If a match cannot be found on any of the three pages, note it and continue — do not skip silently.

---

## Step 4 — Extract stats from each match page

Fetch each individual match page and extract:

| Field | What to extract |
|---|---|
| `HOME_GOALS` | Number of goals scored by the home team (integer) |
| `AWAY_GOALS` | Number of goals scored by the away team (integer) |
| `HOME_YELLOW_CARDS` | Number of yellow cards given to the home team (integer) |
| `AWAY_YELLOW_CARDS` | Number of yellow cards given to the away team (integer) |
| `HOME_OG` | Own goals scored by the home team — use the format described above; empty if none |
| `AWAY_OG` | Own goals scored by the away team — use the format described above; empty if none |
| `HOME_RED_CARDS` | Players sent off for the home team — use the same format as OG; empty if none |
| `AWAY_RED_CARDS` | Players sent off for the away team — use the same format as OG; empty if none |
| `WINNER_TEAM_ID` | The `TEAM_ID` of the winning team — use `HOME_TEAM_ID` if the home team won, `AWAY_TEAM_ID` if the away team won, or **leave blank** if the match ended in a draw. For knockout matches that were level at full-time, use the team that ultimately won (after extra time or penalties) as shown on the BBC Sport page. |

For player names in OG and red card fields, use the format shown on the BBC Sport page (typically first initial + last name, e.g. `C.Montes`). For minutes, use the exact minute shown, including any `+` added-time notation.

---

## Step 5 — Present a summary for review

Print a summary of every match you have stats for, in this format:

```
[MATCH_ID] HOME_TEAM vs AWAY_TEAM
  Score:         HOME_GOALS – AWAY_GOALS
  Winner:        WINNER_TEAM_ID  (or "draw")
  Yellow cards:  HOME_YELLOW_CARDS (home) / AWAY_YELLOW_CARDS (away)
  Own goals:     HOME_OG | AWAY_OG  (or "none")
  Red cards:     HOME_RED_CARDS | AWAY_RED_CARDS  (or "none")
  Source:        <BBC Sport URL>
```

If any match from Step 2 could not be found on BBC Sport, list it separately at the end and explain that it will not be updated.

---

## Step 6 — Confirm with the user

Ask: _"Do you want me to write these stats to matches.csv? (yes/no)"_

Wait for the user's response before proceeding. If they say no, stop.

---

## Step 7 — Update matches.csv

For each confirmed match:

1. Find the row in `matches.csv` by `MATCH_ID`.
2. Update the following fields with the extracted values:
   - `COMPLETED` → `Y`
   - `WINNER_TEAM_ID` — set to the winning team's ID, or leave blank if the match was a draw
   - `HOME_GOALS`, `AWAY_GOALS`
   - `HOME_YELLOW_CARDS`, `AWAY_YELLOW_CARDS`
   - `HOME_OG`, `AWAY_OG`
   - `HOME_RED_CARDS`, `AWAY_RED_CARDS`
3. Do **not** overwrite `NOTES` — leave that as-is for the user to manage.
4. Preserve all other fields exactly as they are.
5. Write the updated file using Edit, one row at a time — do not rewrite the entire file.

Confirm how many rows were updated once done.
