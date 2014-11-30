package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"

	"encoding/json"
	"fmt"

	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
)

type BinaryFileResponse struct {
	Name    string
	File    http.File
	ModTime time.Time
}

func PortfolioIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("Hit portfolio")

	content, err := ioutil.ReadFile("./views/portfolio.md")

	if err != nil {
		return
		// Do something here
	}

	// Render the template
	template := pongo2.Must(pongo2.FromFile("./views/portfolio.tpl"))
	template.ExecuteWriter(pongo2.Context{"body_content": string(content)}, w)
}

// Fixes the ID3 tag info for a remote audio file
// GET /id3/fix
func Id3FixSong(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	url := query.Get("url")
	title := query.Get("title")
	artist := query.Get("artist")

	// Check the response header to make sure the file is actually an audio file
	resp, err := http.Head(url)
	if err := checkResponse(resp, err); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Ensure the file size is not larger than 20 MB
	if resp.ContentLength > 0x1400000 {
		writeJsonError(w, http.StatusBadRequest, errors.New("File too large"))
		return
	}

	// Get the file
	resp, err = http.Get(url)
	if err = checkResponse(resp, err); err != nil {
		writeJsonError(w, http.StatusBadRequest, err)
		return
	}

	defer resp.Body.Close()
	// Make sure we were given an audio file by checking the content type
	if match, _ := regexp.MatchString(`audio\.+`, resp.Header.Get("Content-Type")); match {
		writeJsonError(w, http.StatusBadRequest, errors.New("Not an audio file"))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "audio/mpeg")

	response, _ := fixSong(artist, title, body)
	defer response.File.Close()

	http.ServeContent(w, r, response.Name, response.ModTime, response.File)
}

func writeJsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)

	data, _ := json.Marshal(map[string]string{
		"error": fmt.Sprintf("%s", err),
	})

	w.Write(data)
}
