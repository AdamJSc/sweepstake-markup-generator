# sweepstake-markup-generator

## About

This project uses a number of config files to generate a HTML-based results portal for a series of Sweepstakes (based on Tournaments such as the football World Cup or European Championships).

Each portal's markup is defined on a per-Sweepstake basis and is intended to be used for exhibiting the winner of (or current leaderboard for) each of the Sweepstake's prizes.

These static HTML files are written to a single output directory (default `public/`) so they can be served via a web server.

## Requirements

- Go 1.19
- Docker
- GNU Make (Makefile)

## Running locally

```bash
make run
```

This will build the markup for the Sweepstakes defined in the local `domain/data/sweepstakes.json` manifest.

It will also spin up an nginx Docker container that serves the project's `public/` directory (default build location) at `http://localhost:8080`

To access the example/demo Sweepstake, visit `http://localhost:8080/example-wc2022/` in your browser.

To run the build process again while the server is running, in a separate terminal:

```bash
make build
```

Refresh your browser to access the latest build content (no need to restart the web server).

## Run tests

```bash
go test ./...
```

## Configuring the Sweepstakes

The entrypoint for the whole build process is the Sweepstakes manifest file.

By default, the `domain/data/sweepstakes.json` file will be used.

Alternatively, you can define and host your own manifest file elsewhere, away from the project repo.

To acquire this manifest via HTTP, set the environment variable `SWEEPSTAKES_URL` to the URL of the manifest file. If this location requires Basic Auth, please also set `SWEEPSTAKES_BASICAUTH` in the format `username:password`.

The manifest must be a JSON file that contains an array of objects with the following schema.

(The `domain/data/sweepstakes.json` file can be used as a guide)

* `id` _(string | required)_ - e.g. _"example-wc2022"_ - ID portion of the Sweepstake's URL.
* `name` _(string | required)_ - e.g. _"Example World Cup 2022"_ - rendered as the title/heading of the results portal.
* `tournament_id` _(string | required)_ - e.g. _example-2022-fifa-world-cup"_ - ID of the Tournament to use as a basis for the Sweepstake.
* `prizes.winner` _(bool | optional)_ - if `true`, include the _Tournament Winner_ prize winner.
* `prizes.runner_up` _(bool | optional)_ - if `true`, include the _Tournament Runner-up_ prize winner.
* `prizes.most_goals_conceded` _(bool | optional)_ - if `true`, include the _Most Goals Conceded_ prize leaderboard.
* `prizes.most_yellow_card` _(bool | optional)_ - if `true`, include the _Most Yellow Cards_ prize leaderboard.
* `prizes.quickest_own_goal` _(bool | optional)_ - if `true`, include the _Quickest Own Goal_ prize leaderboard.
* `prizes.quickest_red_card` _(bool | optional)_ - if `true`, include the _Quickest Red Card_ prize leaderboard.
* `build` _(bool | optional)_ - skips the build if omitted or `false`.
* `participants` _(array | required)_
    * `team_id` _(string | required)_ - e.g. _"ARG"_ - ID of one of the Tournament's Teams (must be a valid Team ID for the specified `tournament_id`, Team IDs cannot be repeated and each Team ID must be included once within the array).
    * `participant_name` _(string | required)_ - e.g. _"Paul McCartney"_ - name of the participant representing the associated Team ID.

## Tournament source files

Each Sweepstake must reference a Tournament by ID.

Tournaments are parsed automatically from the `domain/data/tournaments` directory.

This directory is a collection of folders, with each one representing a separate Tournament and must comprise the following files.

(The existing `2022-fifa-world-cup` folder can be used as a guide).

### markup.gohtml

This is a standard GO template file that contains the markup used to generate the results portal for all Sweepstakes that are based on the current Tournament.

For the full data payload that is passed to the template executor, see `domain.Sweepstake.GenerateMarkup()`.

### matches.csv

This is a CSV file that drives the actual results of each Sweepstake. It must assume the following row format:

* `MATCH_ID` _(string | required)_ - e.g. _"SF1"_ - arbitrary Match ID - can be any value but must be unique - the Match considered to be the Final must have the ID "F" (content inside `[]` is ignored).
* `DATE` _(string | required)_ - e.g. _"20/11/2022"_ - kick-off date in the format _dd/mm/yyyy_
* `TIME` _(string | required)_ - e.g. _"19:00"_ - kick-off time in the format _hh:mm_
* `STAGE` _(string | required)_ - e.g. _"GROUP"_ - must be either `GROUP` (group stage) or `KO` (knockout)
* `COMPLETED` _(string | optional)_ - e.g. _"Y"_ - must be `Y` to denote that the Match has been completed, otherwise leave empty
* `WINNER_TEAM_ID` _(string | optional)_ - e.g. _"ARG"_ - Team who is considered to have won the fixture - must be the same as either Home or Away Team ID - if Match is a draw at the group stage, leave this field blank - if Match is a draw at full-time during knockout stage, this field should be the winner after extra-time or penalties.
* `HOME_TEAM_ID` _(string | optional)_ - e.g. _"ARG"_ - ID of Home Team - can be blank if still TBC (i.e. a knockout round that hasn't been reached yet) - if not empty, must be a valid Tournament Team ID and not the same as Away Team ID.
* `AWAY_TEAM_ID` _(string | optional)_ - e.g. _"BRA"_ - ID of Away Team - can be blank if still TBC (i.e. a knockout round that hasn't been reached yet) - if not empty, must be a valid Tournament Team ID and not the same as Home Team ID.
* `HOME_GOALS` _(int | optional)_ - e.g. _3_ - number of goals scored by the Home Team - considered to be 0 if left blank.
* `AWAY_GOALS` _(int | optional)_ - e.g. _2_ - number of goals scored by the Away Team - considered to be 0 if left blank.
* `HOME_YELLOW_CARDS` _(int | optional)_ - e.g. _4_ - number of yellow cards received by the Home Team - considered to be 0 if left blank.
* `AWAY_YELLOW_CARDS` _(int | optional)_ - e.g. _1_ - number of yellow cards received by the Away Team - considered to be 0 if left blank.
* `HOME_OG` _(string | optional)_ - string representing the own goals scored by the Home Team in the format `N;<PLAYER>:<MINUTE>` - where `N` is the number of own goals to follow, `<PLAYER>` is the player's name as a string and `<MINUTE>` is an integer representing the Match minute (or string if in added time with a `+` offset). The number of instances of `<PLAYER>:<MINUTE>` must equal `N` and each one be separated by a `;`. The `<MINUTE>` values do not necessarily need to be in ascending order within the field. Here are some examples:
    * ✅ (empty field)
    * ✅ `1;Johnson:32` (one player)
    * ✅ `2;Smith:12;Jones:67` (two players)
    * ✅ `3;Doe:53;Reed:45+6;Benson:88` (multiple players, minutes in any order, including an offset)
    * ❌ `1:Johnson:32` (invalid delimiter format)
    * ❌ `2;Johnson:32` (preceeding number does not equal subsequent count of player/minutes)
    * ❌ `2;Smith:12';Jones:67` (invalid minute format)
* `AWAY_OG` _(string | optional)_ - same as above but for own goals scored by the Away Team
* `HOME_RED_CARDS` _(string | optional)_ - same as above but for players sent off for the Home Team (either two yellow cards, or a straight red card)
* `AWAY_RED_CARDS` _(string | optional)_ - same as above but for players sent off for the Away Team (either two yellow cards, or a straight red card)
* `NOTES` _(string | optional)_ - e.g. _"Brazil win 4-2 on penalties"_ - any additional notes - rendered alongside Match result within the results portal (content inside `[]` is ignored).

### teams.json

A JSON representation of Teams competing in the Tournament. Must be an array containing objects of the following schema:

* `id` _(string | required)_ - e.g. _"ARG"_ - Team ID referenced by Tournament Matches and associated Sweepstakes.
* `name` _(string | required)_ - e.g. _"Argentina"_ - Team name which can/should be rendered within results portal markup.
* `image_url` _(string | required)_ - e.g. _http://argentina.jpg"_ - URL to image file representing the associated Team.

### tournament.json

Tournament-level config settings, including:

* `id` _(string | required)_ - e.g. _"2022-fifa-world-cup"_ - Tournament ID referenced by Sweepstakes - must be unique across all Tournaments - it's recommended for simplicity that this value is the same as the Tournament's folder name (although this isn't enforced).
* `name` _(string | required)_ - e.g. _"2022 FIFA World Cup"_ - name of Tournament.
* `image_url` _(string | required)_ - e.g. _http://2022-fifa-world-cup.jpg"_ - URL to image file representing the associated Tournament.
* `with_last_updated` _(bool | optional)_ - if `true`, includes the timestamp of the build within the data payload that is passed to the template executor, so that this can be rendered as part of the results portal markup - omit this value or set to `false` if the Tournament has already elapsed - this will prevent the "last updated" date from being re-rendered and displayed for elapsed Tournaments when the build process is run for future Tournaments.

## Sweepstake Prizes

The following football-specific prizes are currently supported by the Sweepstake generator (hence why only football Tournaments are currently supported).

Only Matches that are flagged as `Completed` will be included in the calculations for each prize.

* **Tournament Winner** - Participant/Team specified as the winner of the Match that has the ID `F` (the final).
* **Tournament Runner-up** - The other Participant/Team that is competing in the Match with ID `F`, but is not specified as the winner.
* **Most Goals Conceded** - Leaderboard of the Participants/Teams that have conceded the most goals throughout the Tournament. Driven primarily by the `HOME_GOALS` and `AWAY_GOALS` fields in `matches.csv`.
* **Most Yellow Cards** - Leaderboard of the Participants/Teams that have received the most yellow cards throughout the Tournament. Driven primarily by the `HOME_YELLOW_CARDS` and `AWAY_YELLOW_CARDS` fields in `matches.csv`.
* **Quickest Own Goal** - Leaderboard of the Participants/Teams that have scored an own goal during the Tournament, ordered quickest first by Match minute. Driven primarily by the `HOME_OG` and `AWAY_OG` fields in `matches.csv`.
* **Quickest Red Card** - Leaderboard of the Participants/Teams who have had a player sent off (either straight red card, or second yellow) during the Tournament, ordered quickest first by Match minute. Driven primarily by the `HOME_RED_CARDS` and `AWAY_RED_CARDS` fields in `matches.csv`.