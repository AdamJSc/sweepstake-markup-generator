package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sweepstake.adamjs.net/domain"
)

var (
	dataBasePath      = filepath.Join("domain", "data")
	defaultFilesystem = os.DirFS(dataBasePath)
	publicPath        = "public"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// load tournaments from filesystem
	tournaments := make(domain.TournamentCollection, 0)
	if err := fs.WalkDir(defaultFilesystem, "tournaments", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() || path == "tournaments" {
			return nil
		}
		tournaments = append(tournaments, mustLoadTournamentFromPath(ctx, path))
		return err
	}); err != nil {
		log.Fatal(err)
	}

	bytesFn := domain.BytesFromFileSystem(defaultFilesystem, "sweepstakes.json")

	// load sweepstakes
	sweepstakes, err := (&domain.SweepstakesJSONLoader{}).
		WithSource(bytesFn).
		WithTournamentCollection(tournaments).
		LoadSweepstakes(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// write markup for each sweepstake
	var skipped int
	for _, sweepstake := range sweepstakes {
		if !sweepstake.Build {
			skipped++
			continue
		}
		mustWriteSweepstakeMarkup(sweepstake)
	}

	// print status message
	generated := len(sweepstakes) - skipped
	log.Printf("success! %d generated (%d skipped)", generated, skipped)
}

func mustLoadTournamentFromPath(ctx context.Context, path string) *domain.Tournament {
	teamsLoader := (&domain.TeamsJSONLoader{}).
		WithFileSystem(defaultFilesystem).
		WithPath(filepath.Join(path, "teams.json"))

	matchesLoader := (&domain.MatchesCSVLoader{}).
		WithFileSystem(defaultFilesystem).
		WithPath(filepath.Join(path, "matches.csv"))

	tournament, err := (&domain.TournamentFSLoader{}).
		WithFileSystem(defaultFilesystem).
		WithTeamsLoader(teamsLoader).
		WithMatchesLoader(matchesLoader).
		WithConfigPath(filepath.Join(path, "tournament.json")).
		WithMarkupPath(filepath.Join(path, "markup.gohtml")).
		LoadTournament(ctx)
	if err != nil {
		log.Fatalf("failed to load tournament from path '%s': %s", path, err.Error())
	}

	return tournament
}

func mustWriteSweepstakeMarkup(sweepstake *domain.Sweepstake) {
	b, err := sweepstake.GenerateMarkup()
	if err != nil {
		log.Fatalf("cannot generate markup for sweepstake '%s': %s", sweepstake.ID, err.Error())
	}

	sweepstakePath := filepath.Join(publicPath, sweepstake.ID)
	if err := os.MkdirAll(sweepstakePath, 0755); err != nil {
		log.Fatalf("cannot create directory '%s': %s", sweepstakePath, err.Error())
	}

	markupPath := filepath.Join(sweepstakePath, "index.html")
	if err := os.WriteFile(markupPath, b, 0644); err != nil {
		log.Fatalf("cannot write markup for sweepstake '%s': %s", sweepstake.ID, err.Error())
	}
}

func dadJoke() {
	req, err := http.NewRequest(http.MethodGet, "https://icanhazdadjoke.com", nil)
	if err != nil {
		panic(any(err))
	}

	req.Header.Set("User-Agent", "curl/7.64.1")
	req.Header.Set("Accept", "*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(any(err))
	}

	if resp.StatusCode != http.StatusOK {
		panic(any(fmt.Sprintf("status code %d", resp.StatusCode)))
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(any(err))
	}

	os.Mkdir("public", 0755)
	filename := filepath.Join("public", "index.html")
	os.Create(filename)

	f, err := os.OpenFile(filename, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(any(err))
	}

	if _, err := fmt.Fprintf(f, "<p>icanhazdadjoke.com says...</p>\n<h1>%s</h1>\n", string(b)); err != nil {
		panic(any(err))
	}

	log.Println("completed!")
}
