package main

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Secret struct {
	Id        int64
	Type      string `json:"type"`
	Code      string
	ShortCode string
	Secret    string `json:"secret"`
	Download  bool   `json:"download"`
	Hidden    bool   `json:"hidden"`
	Short     bool   `json:"short"`
	Life      int    `json:"life"`
	Key       string `json:"key"`
	Blob      []byte `json:"blob"`
	Expiry    time.Time
}

type Response struct {
	State bool   `json:"state"`
	Url   string `json:"url"`
}

type Request struct {
	Type   string `json:"type"`
	Secret string `json:"secret"`
	Blob   []byte `json:"blob"`
}

type Secrets []Secret

var (
	mu     sync.Mutex
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
	smatch, _ = regexp.Compile("(?:/)[a-zA-Z0-9]{6,6}")
	lmatch, _ = regexp.Compile("(?:/)[a-zA-Z0-9]{256,256}")
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

func random(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}
		ret = append(ret, letters[num.Int64()])
	}
	return string(ret)
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
	}
}

func secret(w http.ResponseWriter, r *http.Request) {
	var new Secret
	new.Code = lmatch.FindStringSubmatch(r.RequestURI)[1]
	if new.Get() {
		agent := r.Header.Get("User-Agent")
		if strings.Contains(agent, "facebook") {
			w.WriteHeader(403)
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
		new.Code = lmatch.FindStringSubmatch(r.RequestURI)[1]
		if new.Get() {
			res := Request{
				Type:   new.Type,
				Secret: new.Secret,
				Blob:   new.Blob,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)
		}
	case http.MethodPost:
		new := Secret{}
		err := json.NewDecoder(r.Body).Decode(&new)
		if err != nil {
			log.Print(err)
			w.WriteHeader(400)
			return
		}
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
	secret := Secret{}
	switch true {
	case smatch.MatchString(r.RequestURI):
		secret.ShortCode = smatch.FindStringSubmatch(r.RequestURI)[1]
	case lmatch.MatchString(r.RequestURI):
		secret.Code = lmatch.FindStringSubmatch(r.RequestURI)[1]
	}
	if secret.Get() {
		rurl, _ := url.JoinPath("https://", config.Server.Domain, secret.Code)
		http.Redirect(w, r, rurl, http.StatusFound)
		return
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
		w.Header().Add("Content-Type", http.DetectContentType(filebytes))
		w.Write(filebytes)
	}
}
