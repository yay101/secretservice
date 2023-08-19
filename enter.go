package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Secret struct {
	Id        int64
	Type      string `json:"type"`
	Code      string
	ShortCode string
	Secret    string `json:"secret"`
	Blob      []byte `json:"blob"`
	Download  bool   `json:"download"`
	Hidden    bool   `json:"hidden"`
	Short     bool   `json:"short"`
	Life      int    `json:"life"`
	Expiry    time.Time
}

type Response struct {
	State bool   `json:"state"`
	Url   string `json:"url"`
}

type Request struct {
	Type   string `json:"type"`
	Secret string `json:"secret"`
}

type Secrets []Secret

var (
	config Config
	smatch *regexp.Regexp
	lmatch *regexp.Regexp
	//go:embed web
	webfs embed.FS
)

func init() {
	//Load the config
	config.Load()
	//init the db
	dbinit()
	//compile the regex
	smatch, _ = regexp.Compile("(?:/)([a-zA-Z0-9]{6,6}$)")
	lmatch, _ = regexp.Compile("(?:/)([a-zA-Z0-9]{256,256})")
}

func main() {
	//start the cleanup clock
	go clock()
	//init the server
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(serve))
	mux.Handle("/blob/", http.HandlerFunc(blob))
	mux.Handle("/service/", http.HandlerFunc(service))
	//start the server!
	err := http.ListenAndServe(":"+strconv.Itoa(config.Server.Port), mux)
	if err != nil {
		log.Print(err)
	}
}

func random(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func clock() {
	for range time.Tick(time.Minute * 1) {
		dbClean()
	}
}

func blob(w http.ResponseWriter, r *http.Request) {
	var new Secret
	new.Code = lmatch.FindStringSubmatch(r.RequestURI)[1]
	if new.Get() {
		w.Write(new.Blob)
		new.Delete()
	} else {
		w.WriteHeader(404)
	}
}

func service(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		new := Secret{}
		new.Code = lmatch.FindStringSubmatch(r.RequestURI)[1]
		if new.Get() {
			res := Request{
				Type:   new.Type,
				Secret: new.Secret,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)
		}
	case http.MethodPost:
		err := r.ParseMultipartForm(64000)
		if err != nil {
			log.Print(err)
			w.WriteHeader(400)
			return
		}
		new := Secret{
			Type:     r.Form.Get("type"),
			Secret:   r.Form.Get("secret"),
			Download: r.Form.Get("download") == "on",
			Hidden:   r.Form.Get("hidden") == "on",
			Short:    r.Form.Get("short") == "on",
		}
		new.Life, _ = strconv.Atoi(r.Form.Get("life"))
		if new.Type != "string" {
			f, h, err := r.FormFile("file")
			if err != nil {
				log.Print(err)
				w.WriteHeader(400)
				return
			}
			new.Secret = h.Filename
			buf := bytes.NewBuffer(nil)
			_, err = io.Copy(buf, f)
			if err != nil {
				log.Print(err)
				w.WriteHeader(400)
				return
			}
			new.Blob = buf.Bytes()
		}
		defer r.Body.Close()
		if new.Short {
			new.ShortCode = random(6)
		}
		new.Code = random(256)
		new.Expiry = time.Now().Local().Add(time.Hour * time.Duration(new.Life))
		if !(recaptcha(r.Header.Get("X-Captcha-Token"), "") || (r.Header.Get("X-Api-Key") == config.Server.ApiKey)) {
			log.Print("Failed reCaptcha or api key check!")
			w.WriteHeader(403)
			return
		}
		res := Response{}
		if new.Add() {
			var secreturl string
			if new.Short {
				new.ShortCode = random(6)
				secreturl, _ = url.JoinPath("https://", config.Server.Domain, new.ShortCode)
			} else {
				secreturl, _ = url.JoinPath("https://", config.Server.Domain, new.Code)
			}
			res.State = true
			res.Url = secreturl

		} else {
			res.State = false
			res.Url = ""
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

func serve(w http.ResponseWriter, r *http.Request) {
	agent := r.Header.Get("User-Agent")
	if strings.Contains(agent, "facebook") {
		w.WriteHeader(403)
		return
	}
	secret := Secret{}
	switch true {
	case smatch.MatchString(r.RequestURI):
		secret.ShortCode = smatch.FindStringSubmatch(r.RequestURI)[1]
		if secret.Get() {
			lPath, err := url.JoinPath("https://", config.Server.Domain, secret.Code)
			if err != nil {
				log.Print(err)
			}
			http.Redirect(w, r, lPath, http.StatusFound)
			return
		}
	case lmatch.MatchString(r.RequestURI):
		secret.Code = lmatch.FindStringSubmatch(r.RequestURI)[1]
		if secret.Get() {
			if secret.Hidden {
				r.ParseForm()
				if !r.Form.Has("show") {
					secret.Type = "show"
				}
			}
			if secret.Type == "string" {
				secret.Delete()
			}
			tmpl, err := template.ParseFS(webfs, "web/templates/"+secret.Type+".html")
			if err != nil {
				log.Print(err)
			}
			tmpl.Execute(w, secret)
			return
		}
	default:
		if r.RequestURI == "/" {
			tmpl, err := template.ParseFS(webfs, "web/index.html")
			if err != nil {
				log.Print(err)
			}
			tmpl.Execute(w, config.Captcha)
			return
		}
		filebytes, err := webfs.ReadFile("web" + r.RequestURI)
		if err != nil {
			log.Print(err)
			w.WriteHeader(404)
		} else {
			w.Header().Add("Content-Type", mime.TypeByExtension("."+strings.Split(r.RequestURI, ".")[1]))
			w.Write(filebytes)
			return
		}
	}
	rpath, err := url.JoinPath("https://", config.Server.Domain)
	if err != nil {
		log.Print(err)
	}
	http.Redirect(w, r, rpath, http.StatusFound)
}
