package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	taglib "github.com/landaire/go-taglib"
)

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
	file, err := ioutil.TempFile("", "id3_")

	if err != nil {
		Log.Logger.Error("Error creating file:", err)
		return nil, err
	}

	file.Write(data)
	file.Close()

	// Now that the data is written to a file, do some taglib stuff
	// this line may cause a race condition on systems where TMPTIME is > 0 and this
	// file was written at 23:59:59 or something like that.
	parsedFile, err := taglib.Read(file.Name())
	defer parsedFile.Close()

	if err != nil {
		Log.Logger.Errorf("Error reading file %s: %s\n", file.Name(), err)
		return nil, err
	}

	if parsedFile == nil {
		Log.Logger.Error("File was not able to be parsed")
		return nil, errors.New("Could not parse file")
	}

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
	file, _ = os.Open(file.Name())
	// The info is required for the mod time
	info, err := file.Stat()
	if err != nil {
		file.Close()
		Log.Logger.Error("There was a problem when calling file.Stat()")
		return nil, err
	}

	// Return the revel BinaryResult
	return &BinaryFileResponse{
		File:    file,
		Name:    fmt.Sprintf("\"%s - %s.mp3\"", artist, title),
		ModTime: info.ModTime(),
	}, nil
}
