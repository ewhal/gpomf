package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dchest/uniuri"
	_ "github.com/go-sql-driver/mysql"
)

const (
	LENGTH         = 6
	PORT           = ":8080"
	DIRECTORY      = "/tmp/"
	GRILLDIRECTORY = ""
	UPADDRESS      = "http://localhost"
	dbUSERNAME     = ""
	dbNAME         = ""
	dbPASSWORD     = ""
	DATABASE       = dbUSERNAME + ":" + dbPASSWORD + "@/" + dbNAME + "?charset=utf8"
	MAXSIZE        = 10 * 1024 * 1024
)

type Result struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
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
	query, err := db.Query("select id from files where id=?", name)
	if err != sql.ErrNoRows {
		for query.Next() {
			generateName()
		}
	}
	db.Close()

	return name
}
func respond(w http.ResponseWriter, output string, resp Response) {
	if resp.ErrorCode != 0 {
		resp.Files = []Result{}
		resp.Success = false
	} else {
		resp.Success = true
	}

	switch output {
	case "json":

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "xml":
		x, err := xml.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		w.Write(x)

	case "html":
		w.Header().Set("Content-Type", "text/html")
		for _, file := range resp.Files {
			io.WriteString(w, "<a href='"+file.URL+"'>"+file.URL+"</a><br />")
		}

	case "gyazo", "text":
		w.Header().Set("Content-Type", "plain/text")
		for _, file := range resp.Files {
			io.WriteString(w, file.URL+"\n")
		}

	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		io.WriteString(w, "name, url, hash, size\n")
		for _, file := range resp.Files {
			io.WriteString(w, file.Name+","+file.URL+","+file.Hash+","+strconv.FormatInt(file.Size, 10)+"\n")
		}

	default:
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}

}
func grillHandler(w http.ResponseWriter, r *http.Request) {
	kawaii, err := ioutil.ReadDir(GRILLDIRECTORY)
	check(err)
	http.Redirect(w, r, GRILLDIRECTORY+kawaii[rand.Intn(len(kawaii))].Name(), 301)
}
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	reader, err := r.MultipartReader()
	resp := Response{Files: []Result{}}
	output := r.FormValue("output")
	if err != nil {
		resp.ErrorCode = http.StatusInternalServerError
		resp.Description = err.Error()
		respond(w, output, resp)
		return
	}

	db, err := sql.Open("mysql", DATABASE)
	check(err)

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
			respond(w, output, resp)
			break
		}

		h := sha1.New()
		t := io.TeeReader(part, h)
		_, err = io.Copy(dst, t)

		if err != nil {
			resp.ErrorCode = http.StatusInternalServerError
			resp.Description = err.Error()
			respond(w, output, resp)
			break
		}
		hash := h.Sum(nil)
		sha1 := base64.URLEncoding.EncodeToString(hash)
		stat, _ := dst.Stat()
		size := stat.Size()
		if size > MAXSIZE {
			resp.ErrorCode = http.StatusRequestEntityTooLarge
			resp.Description = err.Error()
			break
		}

		originalname := part.FileName()
		err = db.QueryRow("select originalname, filename, size from files where hash=?", sha1).Scan(&originalname, &filename, &size)
		res := Result{
			URL:  UPADDRESS + "/" + filename,
			Name: originalname,
			Hash: sha1,
			Size: size,
		}
		if err == sql.ErrNoRows {
			query, err := db.Prepare("INSERT into files(hash, originalname, filename, size, date) values(?, ?, ?, ?, ?)")
			check(err)
			_, err = query.Exec(res.Hash, res.Name, filename, res.Size, time.Now().Format("2016-01-02"))
			check(err)
		}
		resp.Files = append(resp.Files, res)
	}
	respond(w, output, resp)
}

func main() {
	http.HandleFunc("/upload.php", uploadHandler)
	http.HandleFunc("/grill.php", grillHandler)
	err := http.ListenAndServe(PORT, nil)
	if err != nil {
		panic(err)
	}

}
