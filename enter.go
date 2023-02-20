package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Secret struct {
	Id        int64
	Type      string `json:"type"`
	ShortCode string `json:"shortcode"`
	Code      string
	Code2     string
	Secret    string `json:"secret"`
	Download  bool   `json:"download"`
	Hidden    bool   `json:"hidden"`
	Short     bool   `json:"short"`
	Life      int    `json:"life"`
	Token     string `json:"token"`
	Expiry    int64
}

type Response struct {
	State bool   `json:"state"`
	Url   string `json:"url"`
}

type Request struct {
	Type   string `json:"type"`
	Secret string `json:"secret"`
	Blob   any    `json:"blob"`
}

type Secrets []Secret

var (
	ownPath        string
	configLocation string
	serverName     = "secretservice"
	mu             sync.Mutex
	config         Config
	smatch         *regexp.Regexp
	//go:embed web
	webfs embed.FS
)

func init() {
	//Get EXE location
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	//Get Folder of EXE
	ownPath = filepath.Dir(ex)
	//Create blob file
	_ = os.Mkdir(path.Join(ownPath, "blobs"), 0700)
	//Load the config
	config.Load()
	//init the db
	dbinit()
	//compile the regex
	smatch, _ = regexp.Compile("[a-zA-Z0-9]{6,}$")
}

func main() {
	//start the cleanup clock
	go clock()
	//init the server
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(serve))
	mux.Handle("/secret/", http.HandlerFunc(secret))
	mux.Handle("/blob/", http.HandlerFunc(blob))
	mux.Handle("/service", http.HandlerFunc(service))
	//start the server!
	err := http.ListenAndServe(":"+strconv.Itoa(config.Server.Port), mux)
	if err != nil {
		log.Print(err)
	}
}

func clock() {
	for range time.Tick(time.Minute * 1) {
		dbClean()
	}
}

func blob(w http.ResponseWriter, r *http.Request) {
	var new Secret
	if new.toCode(r.RequestURI) {
		if new.Get() {
			data, err := os.ReadFile(path.Join(ownPath, "blobs", new.Code, new.Code2))
			if err != nil {
				log.Print("Error getting file: " + err.Error())
				w.WriteHeader(400)
			} else {
				w.Write(data)
				new.Delete()
			}
		}
	}
}

func secret(w http.ResponseWriter, r *http.Request) {
	var new Secret
	if new.toCode(r.RequestURI) {
		if new.Get() {
			agent := r.Header.Get("User-Agent")
			if strings.Contains(agent, "facebook") {
				log.Print("Its facebook.")
				w.WriteHeader(400)
				return
			}
			if new.Hidden {
				r.ParseForm()
				if !r.Form.Has("show") {
					new.Type = "show"
				}
			}
			tmpl, err := template.ParseFS(webfs, "web/templates/"+new.Type+".html")
			if err != nil {
				log.Print(err)
			}
			tmpl.Execute(w, new)
			return
		}
	}
	rpath, err := url.JoinPath("https://", config.Server.Domain)
	if err != nil {
		log.Print(err)
	}
	http.Redirect(w, r, rpath, http.StatusFound)
}

func service(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		new := Secret{}
		if new.toCode(r.RequestURI) {
			if new.Get() {
				res := Request{
					Type:   new.Type,
					Secret: new.Secret,
				}
				if new.Type != "string" {
					data, err := os.ReadFile(path.Join(ownPath, "blobs", new.Code, new.Code2))
					if err != nil {
						w.WriteHeader(400)
					} else {
						res.Blob = data
					}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(res)
				new.Delete()
			}
		}
	case http.MethodPost:
		r.ParseMultipartForm(10000000)
		new := Secret{
			Secret:   r.Form.Get("secret"),
			Code:     strings.Join(strings.Split(uuid.New().String(), "-"), ""),
			Code2:    strings.Join(strings.Split(uuid.New().String(), "-"), ""),
			Token:    r.Form.Get("token"),
			Type:     r.Form.Get("type"),
			Hidden:   r.Form.Get("hidden") == "on",
			Download: r.Form.Get("download") == "on",
		}
		new.Life, _ = strconv.Atoi(r.Form.Get("life"))
		if new.Type != "string" {
			f, h, err := r.FormFile("file")
			new.Secret = h.Filename
			if err != nil {
				log.Print(err)
			} else {
				defer f.Close()
				err = os.Mkdir(path.Join(ownPath, "blobs", new.Code), 0700)
				if err != nil {
					log.Print(err)
				}
				file, err := os.OpenFile(path.Join(ownPath, "blobs", new.Code, new.Code2), os.O_WRONLY|os.O_CREATE, 0700)
				if err != nil {
					log.Print(err)
					return
				}
				defer file.Close()
				_, err = io.Copy(file, f)
				if err != nil {
					log.Print(err)
					return
				}
			}
		}
		new.Expiry = time.Now().Local().Add(time.Hour * time.Duration(new.Life)).UnixMilli()
		if !(recaptcha(new.Token, "") || (r.Header.Get("X-Api-Key") == config.Server.ApiKey)) {
			log.Print("Failed reCaptcha or api key check!")
			w.WriteHeader(400)
			return
		}
		res := Response{}
		if new.Add() {
			var secreturl string
			if new.Short {
				secreturl, _ = url.JoinPath("https://", config.Server.Domain, new.ShortCode)
			} else {
				secreturl, _ = url.JoinPath("https://", config.Server.Domain, "secret", new.Code, new.Code2)
			}
			res = Response{
				State: true,
				Url:   secreturl,
			}
		} else {
			res = Response{
				State: false,
				Url:   "",
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

func serve(w http.ResponseWriter, r *http.Request) {
	if smatch.MatchString(r.RequestURI) {
		secret := Secret{
			ShortCode: smatch.FindString(r.RequestURI),
		}
		if !secret.Get() {
			rurl, _ := url.JoinPath("https://", config.Server.Domain, "secret", secret.Code, secret.Code2)
			http.Redirect(w, r, rurl, http.StatusFound)
			return
		}
	}
	if r.RequestURI == "/" {
		tmpl, err := template.ParseFS(webfs, "web/index.html")
		if err != nil {
			log.Print(err)
		}
		tmpl.Execute(w, config.Captcha)
	} else {
		filebytes, err := webfs.ReadFile("web" + r.RequestURI)
		if err != nil {
			log.Print(err)
		}
		file := strings.Split(r.RequestURI, ".")
		switch file[len(file)-1] {
		case "html":
			w.Header().Add("Content-Type", "text/html; charset=utf-8")
		case "css":
			w.Header().Add("Content-Type", "text/css; charset=utf-8")
		case "js":
			w.Header().Add("Content-Type", "text/javascript; charset=utf-8")
		case "svg":
			w.Header().Add("Content-Type", "image/svg+xml; charset=utf-8")
		}
		w.Write(filebytes)
	}
}

func (n *Secret) toCode(u string) bool {
	strip := strings.Split(u, "?")[0]
	codes := strings.Split(strip, "/")
	if len(codes) == 4 {
		n.Code = codes[2]
		n.Code2 = codes[3]
		return true
	}
	return false
}
