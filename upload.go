package main

import (

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

//func getHash(file io.Reader) string {
//	h := sha1.New()
//	return string(h.Sum(nil))

//}

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
			dst, err := os.Create(DIRECTORY + s + ".txt")
			defer dst.Close()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, err := io.Copy(dst, part); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			io.WriteString(w, s+"\n")
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
