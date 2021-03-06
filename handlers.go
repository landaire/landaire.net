package main

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"encoding/json"
	"fmt"

	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	xv "github.com/landaire/xval"
	//	"github.com/bradfitz/gomemcache/memcache"
	"github.com/julienschmidt/httprouter"
)

var (
	quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

//	mc *memcache.Client
)

type BinaryFileResponse struct {
	Name    string
	File    http.File
	ModTime time.Time
}

type XvalResult struct {
	DesKey, DecryptedData []byte
	XValueFlags           []string
	Error                 string
}

func init() {
	//	mc = memcache.New("127.0.0.1:11211")
}

func PortfolioIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger.Println("Hit portfolio")

	//	renderedContent, err := mc.Get("portfolio")
	//
	//	if err != nil && renderedContent != nil {
	//		logger.Println("Got cached content")
	//		w.Write(renderedContent.Value)
	//		return
	//	} else {
	//		logger.Error(err)
	//	}

	content, err := ioutil.ReadFile("./views/portfolio.md")

	if err != nil {
		return
		// Do something here
	}

	// Render the template
	template := pongo2.Must(pongo2.FromFile("./views/portfolio.html"))

	context := pongo2.Context{
		"show_back_link": false,
		"body_content":   string(content),
	}

	content, err = template.ExecuteBytes(context)

	if err != nil {
		return
	}

	//	logger.Println("Setting cache")
	//
	//	err = mc.Set(&memcache.Item{
	//		Key: "portfolio",
	//		Value: content,
	//		Expiration: int32((24 * time.Hour * 7) / time.Millisecond),
	//	})

	if err != nil {
		logger.Error(err)
	}

	w.Write(content)
}

func XvalIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	query := r.URL.Query()
	serial := query.Get("serial")
	xval := strings.Replace(query.Get("xval"), "-", "", -1)

	var decryptionError string
	var decryptionResultData *XvalResult
	validationErrors := make(map[string][]string)
	hasErrors := false

	if serial != "" && xval != "" {
		serialConstraints := []Constraint{
			ExactLength{
				Value:        serial,
				Length:       12,
				ErrorMessage: "Serial number must be 12 characters",
			},
			Match{
				Value:        serial,
				Regex:        regexp.MustCompile(`\d{12}`),
				ErrorMessage: "Serial number contains invalid characters",
			},
		}

		xvalConstraints := []Constraint{
			ExactLength{
				Value:        xval,
				Length:       16,
				ErrorMessage: "X Value should be 16 characters without dashes.",
			},
		}

		for key, constraintArr := range map[string][]Constraint{"serial": serialConstraints, "xval": xvalConstraints} {
			for _, val := range constraintArr {
				if !val.Validate() {
					newArray := append(validationErrors[key], val.GetErrorMessage())
					logger.Info(key)
					validationErrors[key] = newArray
					hasErrors = true
				}
			}
		}

		if !hasErrors {
			fmt.Println("does not have errors")
			desKey, data, err := xv.Decrypt(serial, xval)

			if err != nil {
				decryptionError = err.Error()
			} else {
				decryptionResultData = &XvalResult{desKey, data, xv.TextResult(data), fmt.Sprint(err)}
			}

		}
	}

	// Render the template
	template := pongo2.Must(pongo2.FromFile("./views/xval.html"))
	template.ExecuteWriter(pongo2.Context{
		"title":             "Xbox 360 X Value Checker",
		"serial":            serial,
		"xval":              xval,
		"has_errors":        hasErrors,
		"validation_errors": validationErrors,
		"decryption_error":  decryptionError,
		"decryption_result": decryptionResultData,
	}, w)
}

// Fixes the ID3 tag info for a remote audio file
// GET /id3/fix
func Id3FixSong(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	query := r.URL.Query()
	songUrl := query.Get("url")
	title := query.Get("title")
	artist := query.Get("artist")

	// Check the response header to make sure the file is actually an audio file
	resp, err := http.Head(songUrl)
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
	resp, err = http.Get(songUrl)
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

	response, err := fixSong(artist, title, body)
	if err != nil {
		writeJsonError(w, http.StatusInternalServerError, err)
		return
	}

	defer response.File.Close()

	escapedName := quoteEscaper.Replace(response.Name)
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\";", escapedName))
	io.Copy(w, response.File)
}

func writeJsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)

	logger.Error(err)
	data, _ := json.Marshal(map[string]string{
		"error": fmt.Sprintf("%s", err),
	})

	w.Write(data)
}
