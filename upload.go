package main

import (
	"crypto/sha1"
	//	"encoding/json"
	"fmt"
	"github.com/dchest/uniuri"
	"io"
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
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("files[]")
		check(err)
		defer file.Close()
		fmt.Fprintf(w, "%v", handler.Header)
		f, err := os.OpenFile("/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		check(err)
		defer f.Close()
		io.Copy(f, file)
	}

}

func main() {
	http.HandleFunc("/upload.php", uploadHandler)
	err := http.ListenAndServe(PORT, nil)
	if err != nil {
		panic(err)
	}

}
