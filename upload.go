package main

import (

	//	"encoding/json"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"

	"github.com/dchest/uniuri"
	//	"mime/multipart"
	"net/http"
	"os"
)

const (
	LENGTH    = 6
	PORT      = ":8080"
	DIRECTORY = "/tmp/"
	UPADDRESS = "http://localhost/"
)

type Result struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `jason:"size"`
}

type Response struct {
	SUCCESS     bool     `json:"success"`
	ERRORCODE   int      `json:"errorcode,omitempty"`
	DESCRIPTION string   `json:"description,omitempty"`
	FILES       []Result `json:"files,omitempty"`
}

func generateName() string {
	name := uniuri.NewLen(LENGTH)
	return name
}
func check(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		io.WriteString(w, "Error")
	case "POST":

		reader, err := r.MultipartReader()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			if part.FileName() == "" {
				continue
			}
			s := generateName()
			extName := filepath.Ext(part.FileName())
			filename := s + extName
			dst, err := os.Create(DIRECTORY + filename)
			defer dst.Close()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			h := sha1.New()
			t := io.TeeReader(part, h)
			_, err = io.Copy(dst, t)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			hash := h.Sum(nil)
			sha1 := base64.URLEncoding.EncodeToString(hash)
			size := dst.Stat()
			res := Result{
				URL:  UPADDRESS + "/" + s,
				Name: part.FileName(),
				Hash: sha1,
				Size: size.Size(),
			}

			io.WriteString(w, filename+"\n")
		}
	}

}

func main() {
	http.HandleFunc("/upload.php", uploadHandler)
	err := http.ListenAndServe(PORT, nil)
	if err != nil {
		panic(err)
	}

}
