package main

import (

	//	"encoding/json"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"time"

	//"mime/multipart"
	"net/http"
	"os"
	//"encoding/base64"
	//"encoding/xml"

	"github.com/dchest/uniuri"
	_ "github.com/go-sql-driver/mysql"
)

const (
	LENGTH     = 6
	PORT       = ":8080"
	DIRECTORY  = "/tmp/"
	UPADDRESS  = "http://localhost"
	dbUSERNAME = ""
	dbNAME     = ""
	dbPASSWORD = ""
	DATABASE   = dbUSERNAME + ":" + dbPASSWORD + "@/" + dbNAME + "?charset=utf8"
)

type Result struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `jason:"size"`
}

type Response struct {
	Success     bool     `json:"success"`
	ErrorCode   int      `json:"errorcode,omitempty"`
	Description string   `json:"description,omitempty"`
	Files       []Result `json:"files,omitempty"`
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func generateName() string {
	name := uniuri.NewLen(LENGTH)
	db, err := sql.Open("mysql", DATABASE)
	check(err)
	query, err := db.Query("select id from pastebin where id=?", name)
	if err != sql.ErrNoRows {
		for query.Next() {
			generateName()
		}
	}
	db.Close()

	return name
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	reader, err := r.MultipartReader()

	resp := Response{Files: []Result{}}
	if err != nil {
		resp.ErrorCode = http.StatusInternalServerError
		resp.Description = err.Error()
		resp.Success = false
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
			resp.ErrorCode = http.StatusInternalServerError
			resp.Description = err.Error()
			return
		}

		h := sha1.New()
		t := io.TeeReader(part, h)
		_, err = io.Copy(dst, t)

		if err != nil {
			resp.ErrorCode = http.StatusInternalServerError
			resp.Description = err.Error()
			return
		}
		hash := h.Sum(nil)
		sha1 := base64.URLEncoding.EncodeToString(hash)
		size, _ := dst.Stat()
		db, err := sql.Open("mysql", DATABASE)
		check(err)
		query, err := db.Prepare("INSERT into files(hash, originalname, filename, size, date) values(?, ?, ?, ?, ?)")
		res := Result{
			URL:  UPADDRESS + "/" + s + extName,
			Name: part.FileName(),
			Hash: sha1,
			Size: size.Size(),
		}
		_, err = query.Exec(res.Hash, res.Name, res.Hash, res.Size, time.Now().Format("2016-01-02 15:04:05"))
		check(err)
		resp.Files = append(resp.Files, res)

	}
	fmt.Println(resp)
}

func main() {
	http.HandleFunc("/upload.php", uploadHandler)
	err := http.ListenAndServe(PORT, nil)
	if err != nil {
		panic(err)
	}

}
