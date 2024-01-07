package main

import (
	"flag"
	"fmt"
	"fserver/deps/way"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	url        string
	address    string
	namelength int
	secretkey  string
	path       string
	index      string
}

type onlyFilesFS struct {
	fs http.FileSystem
}

func (of onlyFilesFS) Open(path string) (http.File, error) {
	f, err := of.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		return nil, os.ErrNotExist
	}

	return f, nil
}

func (c *config) serveindex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, c.index)
}

func (c *config) readConfig() {
	flag.StringVar(&c.url, "url", "localhost", "url at which files are served")
	flag.StringVar(&c.address, "address", "0.0.0.0:9090", "address to listen on")
	flag.StringVar(&c.path, "path", "uploads", "path for uploaded files")
	flag.IntVar(&c.namelength, "namelen", 5, "length of filename")
	flag.StringVar(&c.secretkey, "secretkey", "secret", "secret key for restricting access")
	flag.StringVar(&c.index, "index", "public/index.html", "path to index html file")
	flag.Parse()
}

func getKey(r *http.Request) string {
	key := r.FormValue("key")
	if len(key) == 0 {
		header := r.Header.Get("Authorization")
		split := strings.Split(header, " ")
		if len(split) == 2 {
			key = split[1]
		}
	}
	return key
}

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (c *config) upload(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	if key != c.secretkey {
		fmt.Fprintf(w, "unauthorized key")
		log.Printf("unauthorized key: %+v", key)
		return
	}
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	log.Printf("file: %+v\t%+v bytes", handler.Filename, handler.Size)

	extension := filepath.Ext(handler.Filename)
	fileBytes, err := io.ReadAll(file)

	newFile := randSeq(c.namelength) + extension
	diskFile := filepath.Join(c.path, newFile)
	// create path if does not exist
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		os.MkdirAll(c.path, 0700)
	}

	os.WriteFile(diskFile, fileBytes, 0644)
	log.Printf("wrote: %+v", diskFile)
	if err != nil {
		log.Println(err)
	}

	fileURL := c.address + "/" + newFile
	fmt.Fprintf(w, "%v", fileURL)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	mux := way.NewRouter()

	// todo
	// helpstr(config) print
	cfg := config{}
	cfg.readConfig()

	mux.HandleFunc("GET", "/", cfg.serveindex)
	mux.HandleFunc("POST", "/", cfg.upload)
	mux.Handle("GET", "/...", http.FileServer(onlyFilesFS{http.Dir(cfg.path)}))

	log.Println("listening on " + cfg.address)
	log.Fatalln(http.ListenAndServe(cfg.address, mux))
}
