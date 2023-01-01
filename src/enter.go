package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Secret struct {
	Id       int64
	Type     string `json:"type"`
	Code     string
	Code2    string
	Secret   string `json:"secret"`
	Download bool   `json:"download"`
	Hidden   bool   `json:"hidden"`
	Life     int    `json:"life"`
	Token    string `json:"token"`
	Expiry   int64
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
		log.Print("Looking for old secrets!")
		dbClean()
	}
}

func blob(w http.ResponseWriter, r *http.Request) {
	strip := strings.Replace(r.RequestURI, "?show=true", "", 1)
	codes := strings.Split(strip, "/")
	new := Secret{
		Code:  codes[2],
		Code2: codes[3],
	}
	if new.Get() {
		data, err := os.ReadFile(path.Join(ownPath, "blobs", new.Code, new.Secret))
		if err != nil {
			log.Print("error getting file")
			w.WriteHeader(400)
		} else {
			w.Write(data)
			new.Delete()
		}
	}
}

func secret(w http.ResponseWriter, r *http.Request) {
	tp := path.Join(ownPath, "/web", "/templates")
	strip := strings.Replace(r.RequestURI, "?show=true", "", 1)
	codes := strings.Split(strip, "/")
	if len(codes) == 4 {
		new := Secret{
			Code:  codes[2],
			Code2: codes[3],
		}
		if new.Get() {
			if new.Hidden {
				r.ParseForm()
				if !r.Form.Has("show") {
					sp := path.Join(tp, "show.html")
					http.ServeFile(w, r, sp)
					return
				}
			}
			var tmpl *template.Template
			switch new.Type {
			case "string":
				tmpl = template.Must(template.ParseFiles(path.Join(tp, "string.html")))
				tmpl.Execute(w, new)
				new.Delete()
			default:
				dp := path.Join(tp, new.Type+".html")
				log.Print(dp)
				http.ServeFile(w, r, dp)
			}
			return
		} else {
			rpath, err := url.JoinPath("https://", config.Server.Domain)
			if err != nil {
				log.Print(err)
			}
			http.Redirect(w, r, rpath, http.StatusFound)
			return
		}
	}
}

func service(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		strip := strings.Replace(r.RequestURI, "?show=true", "", 1)
		codes := strings.Split(strip, "/")
		new := Secret{
			Code:  codes[2],
			Code2: codes[3],
		}
		if new.Get() {
			res := Request{
				Type:   new.Type,
				Secret: new.Secret,
			}
			if new.Type != "string" {
				data, err := os.ReadFile(path.Join(ownPath, "blobs", new.Code, new.Secret))
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
	case http.MethodPost:
		r.ParseMultipartForm(32 << 20)
		lint, _ := strconv.Atoi(r.Form.Get("life"))
		var hid, dow bool
		if r.Form.Get("hidden") == "on" {
			hid = true
		}
		if r.Form.Get("download") == "on" {
			dow = true
		}
		new := Secret{
			Secret:   r.Form.Get("secret"),
			Code:     uuid.New().String(),
			Code2:    uuid.New().String(),
			Token:    r.Form.Get("token"),
			Type:     r.Form.Get("type"),
			Hidden:   hid,
			Download: dow,
			Life:     lint,
		}

		log.Print("HAS FILE")
		f, h, err := r.FormFile("file")
		if err != nil {
			log.Print(err)
		} else {
			defer f.Close()
			new.Secret = h.Filename
			err = os.Mkdir(path.Join(ownPath, "blobs", new.Code), 0700)
			if err != nil {
				log.Print(err)
			}
			file, err := os.OpenFile(path.Join(ownPath, "blobs", new.Code, h.Filename), os.O_WRONLY|os.O_CREATE, 0700)
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
		new.Expiry = time.Now().Local().Add(time.Hour * time.Duration(new.Life)).UnixMilli()
		if !recaptcha(new.Token, "") || apipost(r.Header.Get("X-Api-Key")) {
			log.Print("Failed reCaptcha or api key check!")
			return
		}
		res := Response{}
		if new.Add() {
			newurl, _ := url.JoinPath("https://", config.Server.Domain, "secret", new.Code, new.Code2)
			res = Response{
				State: true,
				Url:   newurl,
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
	wp := path.Join(ownPath, "/web")
	if r.RequestURI == "/" {
		tmpl := template.Must(template.ParseFiles(wp + "/index.html"))
		tmpl.Execute(w, config.Captcha)
	} else {
		dp := path.Join(wp, r.RequestURI)
		http.ServeFile(w, r, dp)
	}
}

func apipost(remotekey string) bool {
	if remotekey == config.Server.PostKey {
		return true
	} else {
		return false
	}
}
