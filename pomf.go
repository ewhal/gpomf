// Package pomf provides a simple file hosting pomf compatible web application
packagep pomf 

import (
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// random string generation package
	"github.com/dchest/uniuri"
	// mysql database package
	_ "github.com/go-sql-driver/mysql"
)

const (
	// LENGTH is used to specify filename length
	LENGTH = 6
	// PORT is the port pomf will listen on
	PORT = ":8080"
	// UPDIRECTORY specifies what directory pomf will upload to
	UPDIRECTORY = "/tmp/"
	// GRILLDIRECTORY directory for Kawaii anime grills
	GRILLDIRECTORY = "pomf/build/img/"
	// POMFDIRECTORY is the directory for static pomf files
	POMFDIRECTORY = "pomf/build"
	// UPADDRESS Domain to serve static files from
	UPADDRESS = "http://localhost"
	// dbUSERNAME Database username
	dbUSERNAME = ""
	// dbNAME database name
	dbNAME = ""
	// dbPASSWORD database password
	dbPASSWORD = ""
	// DATABASE connection constant
	DATABASE = dbUSERNAME + ":" + dbPASSWORD + "@/" + dbNAME + "?charset=utf8"
	// MAXSIZE in bytes
	MAXSIZE = 10 * 1024 * 1024
)

// Result information struct
type Result struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// Response Pomf compatible Response struct
type Response struct {
	Success     bool     `json:"success"`
	ErrorCode   int      `json:"errorcode,omitempty"`
	Description string   `json:"description,omitempty"`
	Files       []Result `json:"files,omitempty"`
}

// generateName returns a random string that isn't in the database
func generateName() (string, error) {
	// generate a random string
	name := uniuri.NewLen(LENGTH)
	// open database connection
	db, err := sql.Open("mysql", DATABASE)
	if err != nil {
		// return error and empty string
		return "", err
	}

	var id string
	// Query database for randomly generated string
	err = db.QueryRow("select id from files where id=?", name).Scan(&id)
	// if string doesn't exist call generateName again
	if err != sql.ErrNoRows {
		generateName()
	}
	// close database connection
	db.Close()

	// return randomly generated string and no error
	return name, nil
}

// respond outputs Response struct in user specified formats
// supported formats are xml, json, text and html
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

// grillHandler randomly selects a kawaii grill
func grillHandler(w http.ResponseWriter, r *http.Request) {
	// read kawaii grill directory
	kawaii, err := ioutil.ReadDir(GRILLDIRECTORY)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// redirect to a randomly selected kawaii grill
	http.Redirect(w, r, "/img/"+kawaii[rand.Intn(len(kawaii))].Name(), http.StatusFound)
}

// uploadHandler constructs the response struct
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	output := r.FormValue("output")
	reader, err := r.MultipartReader()
	resp := Response{Files: []Result{}}
	if err != nil {
		resp.ErrorCode = http.StatusInternalServerError
		resp.Description = err.Error()
		respond(w, output, resp)
		return
	}

	db, err := sql.Open("mysql", DATABASE)
	if err != nil {
		resp.ErrorCode = http.StatusInternalServerError
		resp.Description = err.Error()
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

		// generate random filename
		s, err := generateName()
		if err != nil {
			resp.ErrorCode = http.StatusInternalServerError
			resp.Description = err.Error()
			break
		}
		// get file extension
		extName := filepath.Ext(part.FileName())
		// create new filename with random name and extension
		filename := s + extName
		// create a new file ready for user to upload to
		dst, err := os.Create(UPDIRECTORY + filename)
		if err != nil {
			resp.ErrorCode = http.StatusInternalServerError
			resp.Description = err.Error()
			respond(w, output, resp)
			break
		}
		defer dst.Close()

		// Prepare sha1 hash
		h := sha1.New()

		// async copy uploaded data to be hashed
		t := io.TeeReader(part, h)

		// save uploaded data to created file
		_, err = io.Copy(dst, t)
		if err != nil {
			resp.ErrorCode = http.StatusInternalServerError
			resp.Description = err.Error()
			respond(w, output, resp)
			break
		}
		// hash data
		hash := h.Sum(nil)
		// convert data to human readable format
		sha1 := base64.URLEncoding.EncodeToString(hash)
		stat, _ := dst.Stat()
		// get filesize
		size := stat.Size()

		// check to see if filesize is larger than MAXSIZE
		if size > MAXSIZE {
			resp.ErrorCode = http.StatusRequestEntityTooLarge
			resp.Description = err.Error()
			break
		}

		// save original name
		originalname := part.FileName()
		// query database to see if file exists
		err = db.QueryRow("select originalname, filename, size from files where hash=?", sha1).Scan(&originalname, &filename, &size)
		// prepare Result struct
		res := Result{
			URL:  UPADDRESS + "/" + filename,
			Name: originalname,
			Hash: sha1,
			Size: size,
		}
		// if file does not exist insert data into the files table
		if err == sql.ErrNoRows {
			// prepare statement
			query, err := db.Prepare("INSERT into files(hash, originalname, filename, size, date) values(?, ?, ?, ?, ?)")
			if err != nil {
				resp.ErrorCode = http.StatusInternalServerError
				resp.Description = err.Error()
				break
			}
			// execute statement with all necessary variables
			_, err = query.Exec(res.Hash, res.Name, filename, res.Size, time.Now().Format("2016-01-02"))
			if err != nil {
				resp.ErrorCode = http.StatusInternalServerError
				resp.Description = err.Error()
				break
			}
		}
		// append file to response struct
		resp.Files = append(resp.Files, res)
	}
	// call repond function
	respond(w, output, resp)
}

func main() {
	http.HandleFunc("/upload.php", uploadHandler)
	http.HandleFunc("/grill.php", grillHandler)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(POMFDIRECTORY))))
	err := http.ListenAndServe(PORT, nil)
	if err != nil {
		panic(err)
	}

}