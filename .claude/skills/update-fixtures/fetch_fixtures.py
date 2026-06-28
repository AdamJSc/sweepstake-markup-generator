#!/usr/bin/env python3
"""
Resolve knockout-stage placeholder slots in matches.csv (e.g. "Winner Group F",
"3rd Place: Loser SF1") into real team IDs.

Usage:
    python3 fetch_fixtures.py <matches_csv_path> [--tournament <name>]              # compute + diff -> JSON to stdout
    python3 fetch_fixtures.py <matches_csv_path> [--tournament <name>] --write      # read JSON (stdin) -> apply to CSV
    python3 fetch_fixtures.py <matches_csv_path> [--tournament <name>] --validate   # re-fetch + print comparison table

Two independent resolution strategies are used, depending on what a slot
references:

1. Group-stage slots ("Winner Group C", "Runner-up Group F", "Best 3rd
   (A/B/C/D/F)") only appear in the Round of 32. These are resolved by
   fetching BBC Sport's knockout fixture pages (current month + next month)
   and matching CSV rows to BBC events via the *slot text itself* — not by
   kickoff time, since this dataset's synthetic KO kickoff times do not
   correspond to BBC's real-world times. A BBC fixture exposes a slot's
   resolved team only once decided; until then it shows the same kind of
   placeholder text ("2nd Group F"), which is matched verbatim against the
   slot parsed from the CSV row's NOTES field.

2. Bracket-internal slots ("Winner R32_M73", "Loser SF1") appear from the
   Round of 16 onward and reference another row in this same matches.csv by
   MATCH_ID. These need no network access at all — once the referenced match
   has COMPLETED=Y, its winner/loser is read directly from the CSV.
"""

import csv
import json
import re
import subprocess
import sys
from datetime import datetime, timezone

BBC_UA = "Mozilla/5.0"

TEAM_ALIASES = {
    "ARG": ["argentina"], "AUS": ["australia"], "AUT": ["austria"],
    "BEL": ["belgium"], "BIH": ["bosnia"], "BRA": ["brazil"],
    "CAN": ["canada"], "CHE": ["switzerland"], "CIV": ["ivory coast", "côte d"],
    "COD": ["congo dr", "dr congo", "congo"], "COL": ["colombia"],
    "CPV": ["cape verde"], "CUW": ["curaçao", "curacao"],
    "CZE": ["czech"], "DEU": ["germany"], "DZA": ["algeria"],
    "ECU": ["ecuador"], "EGY": ["egypt"], "ENG": ["england"],
    "ESP": ["spain"], "FRA": ["france"], "GHA": ["ghana"],
    "HRV": ["croatia"], "HTI": ["haiti"], "IRN": ["iran"],
    "IRQ": ["iraq"], "JOR": ["jordan"], "JPN": ["japan"],
    "KOR": ["south korea", "korea"], "MAR": ["morocco"],
    "MEX": ["mexico"], "NLD": ["netherlands"], "NOR": ["norway"],
    "NZL": ["new zealand"], "PAN": ["panama"], "PRT": ["portugal"],
    "PRY": ["paraguay"], "QAT": ["qatar"], "SAU": ["saudi"],
    "SCO": ["scotland"], "SEN": ["senegal"], "SWE": ["sweden"],
    "TUN": ["tunisia"], "TUR": ["turkey", "türkiye"],
    "URY": ["uruguay"], "USA": ["united states", "usa"], "UZB": ["uzbekistan"],
    "ZAF": ["south africa"],
}

TOURNAMENT_CONFIG = {
    "fifa-world-cup": {
        "bbc_path": "football/world-cup",
        "event_filter": "FIFA World Cup",
        "ko_labels": ["last 32", "round of 32"],
        "group_pattern": r"^Winner Group ([A-L])$",
        "runner_up_pattern": r"^Runner-up Group ([A-L])$",
        "best_3rd_pattern": r"^Best 3rd \(([A-Z/]+)\)$",
    },
    "uefa-euro": {
        "bbc_path": "football/euro-2024",
        "event_filter": "UEFA Euro",
        "ko_labels": ["round of 16", "last 16"],
        "group_pattern": r"^Winner Group ([A-Z])$",
        "runner_up_pattern": r"^Runner-up Group ([A-Z])$",
        "best_3rd_pattern": None,
    },
}


# ---------------------------------------------------------------------------
# Tournament detection and config
# ---------------------------------------------------------------------------

def detect_tournament(csv_path):
    """Extract tournament name from path like domain/data/tournaments/<tournament>/matches.csv"""
    parts = csv_path.split("/")
    if "tournaments" in parts:
        idx = parts.index("tournaments")
        if idx + 1 < len(parts):
            return parts[idx + 1]
    return None


def get_config(tournament):
    """Get tournament config or raise error if not found."""
    if tournament not in TOURNAMENT_CONFIG:
        available = ", ".join(TOURNAMENT_CONFIG.keys())
        raise ValueError(f"Unknown tournament '{tournament}'. Available: {available}")
    return TOURNAMENT_CONFIG[tournament]


# ---------------------------------------------------------------------------
# Network
# ---------------------------------------------------------------------------

def fetch(url):
    r = subprocess.run(["curl", "-s", "-A", BBC_UA, url], capture_output=True, text=True)
    return r.stdout


def parse_bbc_json(html):
    m = re.search(r'window\.__INITIAL_DATA__=(".*?");\s*</script>', html, re.DOTALL)
    if not m:
        return None
    return json.loads(json.loads(m.group(1)))


def current_and_next_month():
    now = datetime.now(timezone.utc)
    months = [now.strftime("%Y-%m")]
    nxt_year, nxt_month = (now.year + 1, 1) if now.month == 12 else (now.year, now.month + 1)
    months.append(f"{nxt_year:04d}-{nxt_month:02d}")
    return months


def fetch_r32_events(config):
    """Fetch current + next month BBC pages and return all knockout events, deduped."""
    seen = {}
    for ym in current_and_next_month():
        url = f"https://www.bbc.co.uk/sport/{config['bbc_path']}/scores-fixtures/{ym}?filter=fixtures"
        data = parse_bbc_json(fetch(url))
        if not data:
            continue
        key = next((k for k in data["data"] if k.startswith("sport-data-scores-fixtures")), None)
        if not key:
            continue
        for grp in data["data"][key]["data"].get("eventGroups", []):
            for sg in grp.get("secondaryGroups", []):
                if sg.get("displayLabel", "").lower() not in config["ko_labels"]:
                    continue
                for ev in sg.get("events", []):
                    seen[ev["id"]] = ev
    return list(seen.values())


def resolve_team(full_name):
    low = full_name.lower()
    for team_id, aliases in TEAM_ALIASES.items():
        if any(a in low for a in aliases):
            return team_id
    return None


def bbc_slot(full_name):
    """Parse a BBC placeholder name into a normalized slot key, or None if it's a real team."""
    m = re.match(r"^(1st|2nd|3rd)\s+Group\s+([A-Z/]+)$", full_name.strip())
    if not m:
        return None
    ordinal, letters = m.group(1), m.group(2)
    return (ordinal, "/".join(sorted(letters.split("/"))))


def parse_csv_datetime(date_str, time_str):
    """Parse CSV date (DD/MM/YYYY) and time (HH:MM) into a normalized comparable tuple."""
    try:
        dt = datetime.strptime(f"{date_str} {time_str}", "%d/%m/%Y %H:%M")
        return (dt.year, dt.month, dt.day, dt.hour, dt.minute)
    except (ValueError, TypeError):
        return None


def get_bbc_datetime(event):
    """Extract date/time from BBC event, return normalized tuple or None."""
    try:
        # BBC events store date in date.isoDate (YYYY-MM-DD) and time in date.time (HH:MM)
        date_obj = event.get("date", {})
        iso_date = date_obj.get("isoDate", "")
        time_str = date_obj.get("time", "")
        if not iso_date or not time_str:
            return None
        dt = datetime.strptime(f"{iso_date} {time_str}", "%Y-%m-%d %H:%M")
        return (dt.year, dt.month, dt.day, dt.hour, dt.minute)
    except (ValueError, TypeError, AttributeError):
        return None


# ---------------------------------------------------------------------------
# CSV helpers
# ---------------------------------------------------------------------------

def load_csv(path):
    with open(path, newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))


def parse_match_text(notes):
    """Strip trailing '(Venue)' and any leading label, split into (home_text, away_text)."""
    notes = re.sub(r"\s*\([^)]*\)\s*$", "", notes)
    notes = re.sub(r"^[^:]*:\s*", "", notes) if ":" in notes.split(" vs ")[0] else notes
    home_text, away_text = notes.split(" vs ", 1)
    return home_text.strip(), away_text.strip()


def csv_slot(text, config):
    """Parse a CSV NOTES side into a normalized slot key matching bbc_slot's format, or None."""
    m = re.match(config["group_pattern"], text)
    if m:
        return ("1st", m.group(1))
    m = re.match(config["runner_up_pattern"], text)
    if m:
        return ("2nd", m.group(1))
    if config["best_3rd_pattern"]:
        m = re.match(config["best_3rd_pattern"], text)
        if m:
            return ("3rd", "/".join(sorted(m.group(1).split("/"))))
    return None


def internal_ref(text):
    """Parse a 'Winner <MATCH_ID>' / 'Loser <MATCH_ID>' reference, or None."""
    m = re.search(r"(Winner|Loser)\s+(\S+)$", text)
    if not m:
        return None
    return m.group(1), m.group(2)


# ---------------------------------------------------------------------------
# Resolution: group-stage slots (Round of 32) via BBC
# ---------------------------------------------------------------------------

def build_slot_to_row(r32_rows, config):
    slot_to_row = {}
    for row in r32_rows:
        home_text, away_text = parse_match_text(row["NOTES"])
        for text, field in ((home_text, "HOME_TEAM_ID"), (away_text, "AWAY_TEAM_ID")):
            slot = csv_slot(text, config)
            if slot:
                slot_to_row[slot] = (row, field)
    return slot_to_row


def resolve_r32_slots(r32_rows, bbc_events, config):
    slot_to_row = build_slot_to_row(r32_rows, config)
    updates, conflicts, ambiguous = [], [], []
    bbc_dt = {ev["id"]: get_bbc_datetime(ev) for ev in bbc_events}

    for ev in bbc_events:
        sides = [(ev["home"]["fullName"], "home"), (ev["away"]["fullName"], "away")]
        slots = [(bbc_slot(name), name, side) for name, side in sides]
        placeholders = [s for s in slots if s[0] is not None]
        resolved = [(resolve_team(name), name, side) for slot, name, side in slots if slot is None]

        if len(placeholders) == 1 and len(resolved) == 1:
            slot, _, _ = placeholders[0]
            team_id, team_name, _ = resolved[0]
            if not team_id or slot not in slot_to_row:
                continue
            row, placeholder_field = slot_to_row[slot]
            target_field = "AWAY_TEAM_ID" if placeholder_field == "HOME_TEAM_ID" else "HOME_TEAM_ID"
            existing = row.get(target_field, "")
            if not existing:
                updates.append({"match_id": row["MATCH_ID"], "field": target_field,
                                 "value": team_id, "team_name": team_name, "matched_by": "placeholder_slot"})
            elif existing != team_id:
                conflicts.append({"match_id": row["MATCH_ID"], "field": target_field,
                                   "existing": existing, "bbc_value": team_id, "bbc_team_name": team_name})

        elif len(placeholders) == 0 and len(resolved) == 2:
            (id1, name1, _), (id2, name2, _) = resolved
            if not id1 or not id2:
                continue
            already_matched = any(
                {r.get("HOME_TEAM_ID"), r.get("AWAY_TEAM_ID")} == {id1, id2} for r in r32_rows
            )
            if already_matched:
                continue
            anchored = [r for r in r32_rows
                        if {r.get("HOME_TEAM_ID"), r.get("AWAY_TEAM_ID")} & {id1, id2}
                        and "" in (r.get("HOME_TEAM_ID"), r.get("AWAY_TEAM_ID"))]
            if len(anchored) == 1:
                row = anchored[0]
                known_field = "HOME_TEAM_ID" if row.get("HOME_TEAM_ID") else "AWAY_TEAM_ID"
                blank_field = "AWAY_TEAM_ID" if known_field == "HOME_TEAM_ID" else "HOME_TEAM_ID"
                other_id = id2 if row[known_field] == id1 else id1
                other_name = name2 if row[known_field] == id1 else name1
                if not row.get(blank_field):
                    updates.append({"match_id": row["MATCH_ID"], "field": blank_field,
                                     "value": other_id, "team_name": other_name, "matched_by": "team_anchor"})
            elif not anchored:
                # Fallback: try to match by date/time
                bbc_dt_val = bbc_dt.get(ev["id"])
                matched_by_datetime = None
                if bbc_dt_val:
                    for row in r32_rows:
                        csv_dt_val = parse_csv_datetime(row.get("DATE", ""), row.get("TIME", ""))
                        if csv_dt_val and csv_dt_val == bbc_dt_val:
                            # Found a row with matching date/time; check if we can fill in the teams
                            home_blank = not row.get("HOME_TEAM_ID")
                            away_blank = not row.get("AWAY_TEAM_ID")
                            if home_blank or away_blank:
                                matched_by_datetime = row
                                break

                if matched_by_datetime:
                    row = matched_by_datetime
                    home_blank = not row.get("HOME_TEAM_ID")
                    away_blank = not row.get("AWAY_TEAM_ID")
                    if home_blank and away_blank:
                        # Both blank: fill home=id1, away=id2
                        updates.append({"match_id": row["MATCH_ID"], "field": "HOME_TEAM_ID",
                                        "value": id1, "team_name": name1, "matched_by": "date_time"})
                        updates.append({"match_id": row["MATCH_ID"], "field": "AWAY_TEAM_ID",
                                        "value": id2, "team_name": name2, "matched_by": "date_time"})
                    elif home_blank:
                        # Only home blank: guess based on order
                        other_id = id2 if row["AWAY_TEAM_ID"] == id1 else id1
                        other_name = name2 if row["AWAY_TEAM_ID"] == id1 else name1
                        updates.append({"match_id": row["MATCH_ID"], "field": "HOME_TEAM_ID",
                                        "value": other_id, "team_name": other_name, "matched_by": "date_time"})
                    elif away_blank:
                        # Only away blank: guess based on order
                        other_id = id2 if row["HOME_TEAM_ID"] == id1 else id1
                        other_name = name2 if row["HOME_TEAM_ID"] == id1 else name1
                        updates.append({"match_id": row["MATCH_ID"], "field": "AWAY_TEAM_ID",
                                        "value": other_id, "team_name": other_name, "matched_by": "date_time"})
                else:
                    ambiguous.append({"bbc_home": name1, "bbc_away": name2,
                                      "reason": "both sides resolved on BBC but no matching CSV row found (by team anchor or date/time)"})

    return updates, conflicts, ambiguous


# ---------------------------------------------------------------------------
# Resolution: bracket-internal slots (R16 onward) via matches.csv itself
# ---------------------------------------------------------------------------

def resolve_internal_slots(rows):
    by_id = {r["MATCH_ID"]: r for r in rows}
    updates = []

    for row in rows:
        if row.get("STAGE") != "KO" or row["MATCH_ID"].startswith("R32_"):
            continue
        home_text, away_text = parse_match_text(row["NOTES"])
        for text, field in ((home_text, "HOME_TEAM_ID"), (away_text, "AWAY_TEAM_ID")):
            if row.get(field):
                continue
            ref = internal_ref(text)
            if not ref:
                continue
            kind, ref_id = ref
            ref_row = by_id.get(ref_id)
            if not ref_row or ref_row.get("COMPLETED") != "Y":
                continue
            winner = ref_row["WINNER_TEAM_ID"]
            if not winner:
                continue
            team_id = winner if kind == "Winner" else (
                ref_row["AWAY_TEAM_ID"] if ref_row["HOME_TEAM_ID"] == winner else ref_row["HOME_TEAM_ID"]
            )
            updates.append({"match_id": row["MATCH_ID"], "field": field,
                             "value": team_id, "team_name": team_id, "matched_by": "bracket_reference"})
    return updates


# ---------------------------------------------------------------------------
# Write
# ---------------------------------------------------------------------------

def write_updates(csv_path, updates):
    with open(csv_path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        fieldnames = reader.fieldnames
        rows = list(reader)

    by_match = {}
    for u in updates:
        by_match.setdefault(u["match_id"], []).append(u)

    rows_touched = set()
    fields_written = 0
    for row in rows:
        for u in by_match.get(row["MATCH_ID"], []):
            row[u["field"]] = u["value"]
            rows_touched.add(row["MATCH_ID"])
            fields_written += 1

    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames, lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)

    print(f"Updated {fields_written} field(s) across {len(rows_touched)} row(s).")


# ---------------------------------------------------------------------------
# Validate (cross-reference table)
# ---------------------------------------------------------------------------

def run_validate(csv_path, config):
    rows = load_csv(csv_path)
    r32_rows = [r for r in rows if r["MATCH_ID"].startswith("R32_")]
    bbc_events = fetch_r32_events(config)
    slot_to_row = build_slot_to_row(r32_rows, config)
    row_to_bbc = {}

    for ev in bbc_events:
        sides = [(ev["home"]["fullName"], "home"), (ev["away"]["fullName"], "away")]
        for name, side in sides:
            slot = bbc_slot(name)
            if slot and slot in slot_to_row:
                row, field = slot_to_row[slot]
                row_to_bbc.setdefault(row["MATCH_ID"], {})[field] = name
                other_field = "AWAY_TEAM_ID" if field == "HOME_TEAM_ID" else "HOME_TEAM_ID"
                other_name = ev["away"]["fullName"] if side == "home" else ev["home"]["fullName"]
                row_to_bbc[row["MATCH_ID"]][other_field] = other_name

    print(f"{'MATCH_ID':10} {'HOME (csv vs bbc)':55} {'AWAY (csv vs bbc)':55} STATUS")
    print("-" * 135)
    for row in r32_rows:
        if row.get("COMPLETED") == "Y":
            continue
        bbc = row_to_bbc.get(row["MATCH_ID"])
        if not bbc:
            print(f"{row['MATCH_ID']:10} {'(no matching BBC fixture found)':55}")
            continue
        statuses = []
        cells = []
        for field, label in (("HOME_TEAM_ID", "HOME"), ("AWAY_TEAM_ID", "AWAY")):
            csv_val = row.get(field, "")
            bbc_name = bbc.get(field, "?")
            bbc_id = resolve_team(bbc_name)
            if csv_val and bbc_id and csv_val == bbc_id:
                cells.append(f"{csv_val} == {bbc_name}")
                statuses.append("OK")
            elif csv_val and bbc_id and csv_val != bbc_id:
                cells.append(f"{csv_val} != {bbc_name} ({bbc_id})")
                statuses.append("MISMATCH")
            elif csv_val and not bbc_id:
                cells.append(f"{csv_val} (BBC: {bbc_name})")
                statuses.append("OK")
            elif not csv_val and bbc_id:
                cells.append(f"(blank) -> BBC: {bbc_name} -- rerun without --validate")
                statuses.append("PENDING")
            else:
                cells.append(f"(blank) -- BBC: {bbc_name}")
                statuses.append("OK")
        status = "OK" if all(s == "OK" for s in statuses) else "/".join(s for s in statuses if s != "OK")
        print(f"{row['MATCH_ID']:10} {cells[0]:55} {cells[1]:55} {status}")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 fetch_fixtures.py <matches_csv_path> [--tournament <name>] [--write|--validate]", file=sys.stderr)
        sys.exit(1)

    csv_path = sys.argv[1]
    tournament = None
    mode = None

    # Parse optional arguments
    i = 2
    while i < len(sys.argv):
        if sys.argv[i] == "--tournament" and i + 1 < len(sys.argv):
            tournament = sys.argv[i + 1]
            i += 2
        elif sys.argv[i] in ("--write", "--validate"):
            mode = sys.argv[i]
            i += 1
        else:
            i += 1

    # Detect tournament from CSV path if not provided
    if not tournament:
        tournament = detect_tournament(csv_path)
    if not tournament:
        print("Error: Could not detect tournament from CSV path. Use --tournament flag.", file=sys.stderr)
        sys.exit(1)

    config = get_config(tournament)

    if mode == "--write":
        data = json.load(sys.stdin)
        write_updates(csv_path, data["updates"])
        return

    if mode == "--validate":
        run_validate(csv_path, config)
        return

    rows = load_csv(csv_path)
    r32_rows = [r for r in rows if r["MATCH_ID"].startswith("R32_")]
    bbc_events = fetch_r32_events(config)

    r32_updates, conflicts, ambiguous = resolve_r32_slots(r32_rows, bbc_events, config)
    internal_updates = resolve_internal_slots(rows)

    print(json.dumps({
        "updates": r32_updates + internal_updates,
        "conflicts": conflicts,
        "ambiguous": ambiguous,
    }, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
