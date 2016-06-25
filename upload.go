package main

import (
	"crypto/sha1"
	//	"encoding/json"
	"fmt"
	"io"

	"github.com/dchest/uniuri"
	//	"mime/multipart"
	"net/http"
	"os"
)

const (
	LENGTH    = 6
	PORT      = ":8080"
	DIRECTORY = "/tmp/"
)

func startup() {

}

func DB() {

}

func exists(location string) bool {
	if _, err := os.Stat(DIRECTORY + location); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true

}

func getHash(file []byte) string {
	h := sha1.New()
	io.WriteString(h, string(file))
	return string(h.Sum(nil))

}

func generateName() string {
	name := uniuri.NewLen(LENGTH)
	if exists(name) == true {
		generateName()
	}
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

		//copy each part to destination.
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			//if part.FileName() is empty, skip this iteration.
			if part.FileName() == "" {
				continue
			}
			dst, err := os.Create(DIRECTORY + part.FileName())
			defer dst.Close()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, err := io.Copy(dst, part); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			io.WriteString(w, part.FileName()+"\n")
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
