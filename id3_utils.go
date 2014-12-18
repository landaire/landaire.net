package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"math/rand"

	taglib "github.com/landaire/go-taglib"
)

var (
	letters   = []rune(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`)
	tempFiles = make(chan string)
)

func init() {
	// Delete any existing files in ./tmp
	os.RemoveAll("./tmp")

	// This other goroutine will clean up files in ./tmp after they're served
	go func() {
		for filePath := range tempFiles {
			os.Remove(filePath)
		}
	}()
}

// Checks a response for errors.
// Returns a revel.Result or nil depending on if the response is bad or good (respectively)
func checkResponse(resp *http.Response, err error) error {
	if err != nil {
		err = errors.New(fmt.Sprintf("Error occurred while getting response: %s", err))
		Log.Logger.Error(err)

		return err
	}
	if resp == nil {
		return errors.New("Invalid response")
	}
	if resp.StatusCode != 200 {
		return errors.New("Response status code was not 200 - OK. Got: " + resp.Status)
	}
	return nil
}

// Writes ID3v2 tags to the MP3 file given by the data argument
func fixSong(artist, title string, data []byte) (*BinaryFileResponse, error) {
	// Can't open a temp file twice... so create it in this dir I guess
	name, err := getTempName()
	if err != nil {
		return nil, err
	}

	file, err := os.Create(name)

	if err != nil {
		Log.Logger.Error("Error creating file:", err)
		return nil, err
	}

	defer func() { tempFiles <- name }()

	file.Write(data)
	file.Close()

	parsedFile, err := taglib.Read(file.Name())
	if err != nil {
		Log.Logger.Errorf("Taglib error reading file %s: %s\n", file.Name(), err)
		return nil, err
	}
	defer parsedFile.Close()

	songTitle := parsedFile.Title()
	songArtist := parsedFile.Artist()

	// Set the data if they don't exist
	if songTitle == "" || songArtist == "" {
		parsedFile.SetTitle(title)
		parsedFile.SetArtist(artist)
		parsedFile.Save()
	}
	// Explicitly close the file here so that it won't have a lock on it when producing
	// the BinaryResult
	parsedFile.Close()

	// file is used in the BinaryResult
	file, err = os.Open(file.Name())
	if err != nil {
		Log.Logger.Error(fmt.Sprintf("Could not open file %s", file.Name()))
	}

	// The info is required for the mod time
	info, err := file.Stat()
	if err != nil {
		file.Close()
		Log.Logger.Error("There was a problem when calling file.Stat()")
		return nil, err
	}

	// Return the BinaryFileResult
	return &BinaryFileResponse{
		File:    file,
		Name:    fmt.Sprintf("%s - %s.mp3", artist, title),
		ModTime: info.ModTime(),
	}, nil
}

// Gets a temp file path relative to the application binary
func getTempName() (string, error) {
	randPart := make([]rune, 10)

	for i := range randPart {
		randPart[i] = letters[rand.Intn(len(letters))]
	}

	if pathExists, _ := exists("./tmp"); !pathExists {
		if err := os.Mkdir("tmp", 0755); err != nil {
			Log.Logger.Error("Error creating tmp dir:", err)
			return "", err
		}
	}

	// Note that taglib requires the file extension
	return "./tmp/id3_" + string(randPart) + ".mp3", nil
}

// Checks if a file/folder exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
