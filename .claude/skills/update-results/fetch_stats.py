#!/usr/bin/env python3
"""
Fetch BBC Sport match stats for incomplete, finished WC matches.

Usage:
    python3 fetch_stats.py <matches_csv_path>

Outputs JSON to stdout:
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
                "away_red_cards": "",
                "source_url": "https://www.bbc.co.uk/sport/football/live/..."
            }
        ],
        "unmatched": ["MATCH_ID", ...]
    }

OG/red card field format:  "N;Player:Minute" e.g. "2;C.Montes:90+2;T.Smith:45"
  - HOME_OG  = own goals scored by home team players (benefiting away team)
  - AWAY_OG  = own goals scored by away team players (benefiting home team)
  - In BBC data: OGs listed under HOME team actions = AWAY_OG (away player scored it)
                 OGs listed under AWAY team actions = HOME_OG (home player scored it)
"""

import csv
import json
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timedelta, timezone

CUTOFF_MINUTES = 115
BBC_UA = "Mozilla/5.0"
MAX_DATES = 7  # cap on how many scores-fixtures pages to fetch

# Map CSV TEAM_ID → keywords present in BBC team fullName (lowercase)
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


# ---------------------------------------------------------------------------
# CSV helpers
# ---------------------------------------------------------------------------

def load_csv(path):
    with open(path, newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))


def find_candidates(rows):
    now = datetime.now(timezone.utc)
    out = []
    for r in rows:
        if r.get("COMPLETED") == "Y":
            continue
        try:
            dt = datetime.strptime(
                f"{r['DATE']} {r['TIME']}", "%d/%m/%Y %H:%M"
            ).replace(tzinfo=timezone.utc)
            if (now - dt).total_seconds() / 60 >= CUTOFF_MINUTES:
                out.append(r)
        except (ValueError, KeyError):
            pass
    return out


# ---------------------------------------------------------------------------
# Scores-fixtures page parsing
# ---------------------------------------------------------------------------

def get_wc_events(html):
    """Return all FIFA World Cup events from a scores-fixtures page JSON."""
    data = parse_bbc_json(html)
    if not data:
        return []
    key = next((k for k in data["data"] if k.startswith("sport-data-scores-fixtures")), None)
    if not key:
        return []
    events = []
    for grp in data["data"][key]["data"].get("eventGroups", []):
        if "FIFA World Cup" not in grp.get("displayLabel", ""):
            continue
        for sg in grp.get("secondaryGroups", []):
            events.extend(sg.get("events", []))
    return events


def team_matches(team_id, bbc_name):
    aliases = TEAM_ALIASES.get(team_id, [team_id.lower()])
    name = bbc_name.lower()
    return any(a in name for a in aliases)


def find_event(events, row):
    """Match a CSV row to a BBC Sport event (must be PostEvent)."""
    home_id, away_id = row["HOME_TEAM_ID"], row["AWAY_TEAM_ID"]
    for e in events:
        if (e.get("status") == "PostEvent"
                and team_matches(home_id, e["home"]["fullName"])
                and team_matches(away_id, e["away"]["fullName"])):
            return e
    return None


def extract_tipo_id(event, html):
    """
    Get BBC live page topic ID for an event.
    Tries JSON fields first; falls back to searching the already-fetched HTML
    for a live link near the team names (no extra network call).
    """
    # 1. Direct JSON fields
    if event.get("tipoTopicId"):
        return event["tipoTopicId"]
    link = event.get("onwardJourneyLink", "")
    m = re.search(r"/live/([a-z0-9]+)$", link)
    if m:
        return m.group(1)

    # 2. HTML fallback: find <a href="/sport/football/live/XXX"> near team names
    home = event["home"]["fullName"]
    away = event["away"]["fullName"]
    for match in re.finditer(r"/sport/football/live/([a-z0-9]+)", html):
        ctx = html[max(0, match.start() - 100): match.end() + 1500]
        if home in ctx and away in ctx:
            return match.group(1)

    return None


def parse_event_actions(event):
    """
    Extract goals, OGs, winner, AET, and penalties from a scores-fixtures event.

    OG attribution:
      - OG listed under HOME team's actions → scored by AWAY player → AWAY_OG
      - OG listed under AWAY team's actions → scored by HOME player → HOME_OG
    """
    def parse_ogs(actions):
        ogs = []
        for ag in actions:
            for act in ag.get("actions", []):
                if "Own Goal" in act.get("type", ""):
                    t = act.get("timeLabel", {}).get("value", "").replace("'", "").replace(" ", "")
                    ogs.append({"urn": ag.get("playerUrn", ""), "name": ag["playerName"], "time": t})
        return ogs

    # Detect AET and penalties
    aet = False
    penalties_home = None
    penalties_away = None

    # Check for AET via periodLabel/statusComment
    if (event.get("periodLabel", {}).get("value") == "AET"
            or event.get("statusComment", {}).get("value") == "AET"):
        aet = True

    # Check for penalty shootout info
    if event.get("shootOut"):
        shootout = event["shootOut"]
        penalties_home = int(shootout.get("homeScore", 0) or 0)
        penalties_away = int(shootout.get("awayScore", 0) or 0)

    return {
        "home_goals": int(event["home"].get("score", 0) or 0),
        "away_goals": int(event["away"].get("score", 0) or 0),
        "winner": event.get("winner", ""),
        # home_og = OGs by HOME team players = listed under AWAY actions
        "home_og": parse_ogs(event["away"].get("actions", [])),
        # away_og = OGs by AWAY team players = listed under HOME actions
        "away_og": parse_ogs(event["home"].get("actions", [])),
        "aet": aet,
        "penalties_home": penalties_home,
        "penalties_away": penalties_away,
    }


# ---------------------------------------------------------------------------
# Live page parsing (yellow + red cards)
# ---------------------------------------------------------------------------

def fmt_name(short):
    """'N. Lastname' or 'Firstname Lastname' → 'N.Lastname'"""
    s = short.strip()
    m = re.match(r"^([A-Z])\.\s+(.+)$", s)
    if m:
        return f"{m.group(1)}.{m.group(2)}"
    parts = s.split(" ", 1)
    if len(parts) == 2:
        return f"{parts[0][0]}.{parts[1]}"
    return s


def fetch_lineups(tipo_id):
    """
    Fetch the live match page and extract from match-lineups JSON:
      - urn → short name map (for resolving OG player names)
      - yellow card counts per team
      - red card entries per team (formatted as 'N.Name:Minute')

    Returns (urn_map, home_yellows, away_yellows, home_reds, away_reds, url)
    """
    url = f"https://www.bbc.co.uk/sport/football/live/{tipo_id}"
    data = parse_bbc_json(fetch(url))

    if not data:
        return {}, 0, 0, [], [], url

    key = next((k for k in data["data"] if k.startswith("match-lineups")), None)
    if not key:
        return {}, 0, 0, [], [], url

    lineups = data["data"][key]["data"]
    urn_map = {}
    home_y = away_y = 0
    home_r, away_r = [], []

    for side, team_key in [("home", "homeTeam"), ("away", "awayTeam")]:
        for ptype in ["starters", "substitutes"]:
            for p in lineups.get(team_key, {}).get("players", {}).get(ptype, []):
                urn_map[p.get("urn", "")] = p["name"]["short"]
                for card in p.get("cards", []):
                    ct = card.get("type", "")
                    ctime = card.get("timeLabel", {}).get("value", "").replace("'", "").replace(" ", "")
                    entry = f"{fmt_name(p['name']['short'])}:{ctime}"
                    if ct == "Yellow Card":
                        if side == "home":
                            home_y += 1
                        else:
                            away_y += 1
                    elif ct in ("Red Card", "Second Yellow Card", "Two Yellow Cards"):
                        if side == "home":
                            home_r.append(entry)
                        else:
                            away_r.append(entry)

    return urn_map, home_y, away_y, home_r, away_r, url


def resolve_og(og_list, urn_map):
    """Format OG entries as 'N.Name:Minute' using urn_map for proper short names."""
    result = []
    for og in og_list:
        name = urn_map.get(og["urn"], og["name"])
        result.append(f"{fmt_name(name)}:{og['time']}")
    return result


def fmt_field(lst):
    return "" if not lst else f"{len(lst)};" + ";".join(lst)


def fmt_notes(actions, home_id, away_id):
    """Format notes text for AET and penalties."""
    home_goals = actions["home_goals"]
    away_goals = actions["away_goals"]
    aet = actions["aet"]
    pen_h = actions["penalties_home"]
    pen_a = actions["penalties_away"]

    if pen_h is not None and pen_a is not None:
        # Penalties shootout
        winner_name = home_id if pen_h > pen_a else away_id
        return f"{home_goals}-{away_goals} AET, {winner_name} win {pen_h}-{pen_a} on penalties"
    elif aet:
        # Extra time but no penalties
        return f"{home_goals}-{away_goals} AET"
    else:
        # Regular time
        return ""


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def write_results(csv_path, results):
    """Apply fetched results to the CSV file in-place."""
    with open(csv_path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        fieldnames = reader.fieldnames
        rows = list(reader)

    result_map = {r["match_id"]: r for r in results}
    updated = 0

    for row in rows:
        r = result_map.get(row["MATCH_ID"])
        if not r:
            continue
        row["COMPLETED"] = "Y"
        row["WINNER_TEAM_ID"] = r["winner_team_id"]
        row["HOME_GOALS"] = str(r["home_goals"])
        row["AWAY_GOALS"] = str(r["away_goals"])
        row["HOME_YELLOW_CARDS"] = str(r["home_yellow_cards"])
        row["AWAY_YELLOW_CARDS"] = str(r["away_yellow_cards"])
        row["HOME_OG"] = r["home_og"]
        row["AWAY_OG"] = r["away_og"]
        row["HOME_RED_CARDS"] = r["home_red_cards"]
        row["AWAY_RED_CARDS"] = r["away_red_cards"]
        # Append notes_text to existing NOTES field
        if r.get("notes_text"):
            existing = row.get("NOTES", "").strip()
            row["NOTES"] = f"{existing} {r['notes_text']}".strip()
        updated += 1

    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames, lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)

    print(f"Updated {updated} row(s).")


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 fetch_stats.py <matches_csv_path> [--write]", file=sys.stderr)
        sys.exit(1)

    csv_path = sys.argv[1]

    if len(sys.argv) >= 3 and sys.argv[2] == "--write":
        data = json.load(sys.stdin)
        write_results(csv_path, data["results"])
        return
    rows = load_csv(csv_path)
    candidates = find_candidates(rows)

    if not candidates:
        print(json.dumps({"results": [], "unmatched": []}, ensure_ascii=False))
        return

    # Calculate date range: from 1 day before earliest candidate to today+1, capped
    dates_dt = [datetime.strptime(c["DATE"], "%d/%m/%Y") for c in candidates]
    start = min(dates_dt).date() - timedelta(days=1)
    end = datetime.now(timezone.utc).date() + timedelta(days=1)
    dates = []
    d = start
    while d <= end and len(dates) < MAX_DATES:
        dates.append(d.strftime("%Y-%m-%d"))
        d += timedelta(days=1)

    # Parallel fetch of scores-fixtures pages + parse WC events
    sf_pages = {}  # date → (html, events)
    sf_urls = {date: f"https://www.bbc.co.uk/sport/football/scores-fixtures/{date}" for date in dates}

    def fetch_sf(date):
        html = fetch(sf_urls[date])
        return date, html, get_wc_events(html)

    with ThreadPoolExecutor(max_workers=len(dates)) as ex:
        for date, html, events in ex.map(fetch_sf, dates):
            sf_pages[date] = (html, events)

    # Flatten all finished events
    all_events = [e for _, (_, evts) in sf_pages.items() for e in evts if e.get("status") == "PostEvent"]

    # Match events to candidates; extract tipo_id using the page HTML that had the event
    matched, unmatched = [], []
    for row in candidates:
        event = find_event(all_events, row)
        if not event:
            unmatched.append(row["MATCH_ID"])
            continue

        # Find which page contained this event (to use its HTML for tipo_id fallback)
        source_html = ""
        for _, (html, evts) in sf_pages.items():
            if any(e.get("id") == event.get("id") for e in evts):
                source_html = html
                break

        tipo_id = extract_tipo_id(event, source_html)
        actions = parse_event_actions(event)
        matched.append({"row": row, "actions": actions, "tipo_id": tipo_id})

    # Parallel fetch of live pages for yellow/red cards + OG name resolution
    def process(item):
        tipo_id = item["tipo_id"]
        if not tipo_id:
            return item, {}, 0, 0, [], [], None
        urn_map, hy, ay, hr, ar, url = fetch_lineups(tipo_id)
        return item, urn_map, hy, ay, hr, ar, url

    results = []
    with ThreadPoolExecutor(max_workers=8) as ex:
        for item, urn_map, hy, ay, hr, ar, url in ex.map(process, matched):
            row = item["row"]
            actions = item["actions"]
            home_id, away_id = row["HOME_TEAM_ID"], row["AWAY_TEAM_ID"]

            winner_id = (
                home_id if actions["winner"] == "home" else
                away_id if actions["winner"] == "away" else
                ""
            )

            results.append({
                "match_id": row["MATCH_ID"],
                "home_team_id": home_id,
                "away_team_id": away_id,
                "home_goals": actions["home_goals"],
                "away_goals": actions["away_goals"],
                "winner_team_id": winner_id,
                "home_yellow_cards": hy,
                "away_yellow_cards": ay,
                "home_og": fmt_field(resolve_og(actions["home_og"], urn_map)),
                "away_og": fmt_field(resolve_og(actions["away_og"], urn_map)),
                "home_red_cards": fmt_field(hr),
                "away_red_cards": fmt_field(ar),
                "notes_text": fmt_notes(actions, home_id, away_id),
                "source_url": url or (
                    f"https://www.bbc.co.uk/sport/football/live/{item['tipo_id']}"
                    if item["tipo_id"] else ""
                ),
            })

    results.sort(key=lambda r: r["match_id"])
    print(json.dumps({"results": results, "unmatched": unmatched}, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
