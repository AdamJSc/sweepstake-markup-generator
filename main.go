package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	req, err := http.NewRequest(http.MethodGet, "https://icanhazdadjoke.com", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "curl/7.64.1")
	req.Header.Set("Accept", "*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("status code %d", resp.StatusCode))
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	os.Mkdir("public", 0755)
	filename := filepath.Join("public", "index.html")
	os.Create(filename)

	f, err := os.OpenFile(filename, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}

	if _, err := fmt.Fprintf(f, "<p>icanhazdadjoke.com says...</p>\n<h1>%s</h1>\n", string(b)); err != nil {
		panic(err)
	}

	log.Println("completed!")
}
