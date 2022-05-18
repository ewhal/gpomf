// Package pomf provides a simple file hosting pomf compatible web application
package main

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

	// random string generation package
	"github.com/dchest/uniuri"
)

type Configuration struct {
	// LENGTH is used to specify filename length
	Length int
	// PORT is the port pomf will listen on
	Port string
	// UPDIRECTORY specifies what directory pomf will upload to
	UpDirectory string
	// GRILLDIRECTORY directory for Kawaii anime grills
	GrillDirectory string
	// POMFDIRECTORY is the directory for static pomf files
	PomfDirectory string
	// UPADDRESS Domain to serve static files from
	UpAddress string
	// dbUSERNAME Database username
	Username string
	// dbNAME database name
	Name string
	// dbPASSWORD database password
	Pass string
	// MAXSIZE in bytes
	MaxSize int64
}

var configuration Configuration

// DATABASE connection constant
var DATABASE string

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
	sent        bool
}

// generateName returns a random string that isn't in the database
func generateName() (string, error) {
	// generate a random string
	name := uniuri.NewLen(configuration.Length)
	// open database connection
	db, err := sql.Open(dbDriver, DATABASE)
	if err != nil {
		// return error and empty string
		return "", err
	}

	var id string
	// Query database for randomly generated string
	err = db.QueryRow(makeQuery("select id from files where id=?"), name).Scan(&id)
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
func respond(w http.ResponseWriter, output string, resp *Response) {
	resp.sent = true
	if resp.ErrorCode != 0 {
		resp.Files = []Result{}
		resp.Success = false
		w.WriteHeader(resp.ErrorCode)
		io.WriteString(w, resp.Description)
		return
	} else {
		resp.Success = true
	}
	if resp.sent {
		// bail we already sent shit back
		return
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
	kawaii, err := ioutil.ReadDir(configuration.GrillDirectory)
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
	resp := &Response{Files: []Result{}}
	if err != nil {
		resp.ErrorCode = http.StatusInternalServerError
		resp.Description = err.Error()
		respond(w, output, resp)
		return
	}

	db, err := sql.Open(dbDriver, DATABASE)
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
		uploadedFilename := filename
		// create a new file ready for user to upload to
		dst, err := os.Create(configuration.UpDirectory + filename)
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
		sh := base64.URLEncoding.EncodeToString(hash)
		stat, _ := dst.Stat()
		// get filesize
		size := stat.Size()
		// check to see if filesize is larger than MAXSIZE
		if size > configuration.MaxSize {
			resp.ErrorCode = http.StatusRequestEntityTooLarge
			resp.Description = "File too large"
			break
		}

		// save original name
		originalname := part.FileName()
		// query database to see if file exists
		err = db.QueryRow(makeQuery("select originalname, filename, size from files where hash=?"), sh).Scan(&originalname, &filename, &size)
		// prepare Result struct
		res := Result{
			URL:  configuration.UpAddress + "/" + filename,
			Name: originalname,
			Hash: sh,
			Size: size,
		}
		// if file does not exist insert data into the files table
		if err == sql.ErrNoRows {
			// prepare statement
			query, err := db.Prepare(makeQuery("INSERT into files(id, hash, originalname, filename, size, date) values(?, ?, ?, ?, ?, ?)"))
			if err != nil {
				resp.ErrorCode = http.StatusInternalServerError
				resp.Description = err.Error()
				respond(w, output, resp)
				break
			}
			// execute statement with all necessary variables
			_, err = query.Exec(filename, res.Hash, res.Name, filename, res.Size, makeTime())
			if err != nil {
				resp.ErrorCode = http.StatusInternalServerError
				resp.Description = err.Error()
				respond(w, output, resp)
				break
			}
		} else if err == nil {
			os.Remove(configuration.UpDirectory + uploadedFilename)
		}
		// append file to response struct
		resp.Files = append(resp.Files, res)
	}
	// call repond function
	respond(w, output, resp)
}

func main() {

	file, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		panic(err)
	}

	configuration.MaxSize = configuration.MaxSize * 1024 * 1024
	DATABASE = makeURL(configuration)

	http.HandleFunc("/upload.php", uploadHandler)
	http.HandleFunc("/grill.php", grillHandler)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(configuration.PomfDirectory))))
	err = http.ListenAndServe(configuration.Port, nil)
	if err != nil {
		panic(err)
	}

}
