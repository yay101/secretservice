package main

import (
	"crypto/tls"
	"embed"
	"flag"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type incoming struct {
	Type    SecretType `json:"type"`
	Data    string     `json:"data"`
	Captcha string     `json:"captcha"`
	Length  uint64     `json:"length"`
	Expiry  int        `json:"expiry"`
	Short   bool       `json:"short"`
	Key     string     `json:"key"`
	Iv      string     `json:"iv"`
}

type SecretType string

const (
	String SecretType = "string"
	File   SecretType = "binary"
	Chat   SecretType = "chat"
)

var (
	conf  config
	match *regexp.Regexp
	//go:embed www
	webfs     embed.FS
	itemplate *template.Template
	stemplate *template.Template
	ftemplate *template.Template
	ctemplate *template.Template
)

type config struct {
	Server struct {
		Domain    *string
		Proxy     *bool
		Filepath  *string
		Key       *string
		Ssl       *int
		Http      *int
		Chunksize *int
	}
	Captcha struct {
		Enabled           *bool
		MinimumComplexity *int
		DefaultComplexity *int
	}
}

func init() {
	log.SetFlags(log.Lshortfile)
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exPath := filepath.Dir(ex)
	conf.Captcha.Enabled = flag.Bool("captcha", true, "Enable altcha.")
	if captcha, present := os.LookupEnv("captcha"); present {
		enabled, err := strconv.ParseBool(captcha)
		if err != nil {
			log.Print(`Bad bool for captcha, accepted string values: "1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"`)
		}
		conf.Captcha.Enabled = &enabled
	}
	conf.Captcha.MinimumComplexity = flag.Int("mincomplexity", 50000, "The minimum complexity for altcha.")
	if mincomplexity, present := os.LookupEnv("mincomplexity"); present {
		mincomplexity, err := strconv.Atoi(mincomplexity)
		if err != nil {
			log.Print(`Bad integer for mincomplexity, 0-2147483647`)
		}
		conf.Captcha.MinimumComplexity = &mincomplexity
	}
	conf.Captcha.DefaultComplexity = flag.Int("defaultcomplexity", 100000, "The complexity for the altcha.")
	if defcomplexity, present := os.LookupEnv("defaultcomplexity"); present {
		defcomplexity, err := strconv.Atoi(defcomplexity)
		if err != nil {
			log.Print(`Bad integer for defcomplexity, 0-2147483647`)
		}
		conf.Captcha.DefaultComplexity = &defcomplexity
	}
	conf.Server.Domain = flag.String("domain", "localhost", "The domain name to use.")
	if domain, present := os.LookupEnv("domain"); present {
		if domain == "" {
			log.Print("Blank domain.")
		} else {
			conf.Server.Domain = &domain
		}
	}
	conf.Server.Proxy = flag.Bool("proxy", false, "Enable proxy mode.")
	if proxy, present := os.LookupEnv("proxy"); present {
		enabled, err := strconv.ParseBool(proxy)
		if err != nil {
			log.Print(`Bad bool for captcha, accepted string values: "1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"`)
		}
		conf.Server.Proxy = &enabled
	}
	conf.Server.Filepath = flag.String("path", exPath+"/data", "The full path to the data folder.")
	if path, present := os.LookupEnv("path"); present {
		if path == "" {
			log.Print("Blank path.")
		} else {
			conf.Server.Domain = &path
		}
	}
	conf.Server.Key = flag.String("key", randomString(32), "The database encoding key.")
	if key, present := os.LookupEnv("key"); present {
		if key == "" || len(key) != 32 {
			log.Print("Bad key.")
		} else {
			conf.Server.Key = &key
		}
	}
	conf.Server.Http = flag.Int("port", 3333, "The port for http on the server. (3333)")
	if port, present := os.LookupEnv("port"); present {
		if port == "" {
			log.Print("Bad port provided.")
		} else {
			iport, err := strconv.Atoi(port)
			if err != nil {
				log.Print("Cannot convert port number: ", port, err)
			}
			conf.Server.Http = &iport
		}
	}
	conf.Server.Ssl = flag.Int("sslport", 4433, "The port for https on the server. (4433)")
	if sslport, present := os.LookupEnv("sslport"); present {
		if sslport == "" {
			log.Print("Bad ssl port provided.")
		} else {
			iport, err := strconv.Atoi(sslport)
			if err != nil {
				log.Print("Cannot convert ssl port number: ", sslport, err)
			}
			conf.Server.Ssl = &iport
		}
	}
	conf.Server.Chunksize = flag.Int("chunksize", 1048576, "Chunksize")
	if chunksize, present := os.LookupEnv("chunksize"); present {
		if chunksize == "" {
			log.Print("Bad chunksize provided.")
		} else {
			isize, err := strconv.Atoi(chunksize)
			if err != nil {
				log.Print("Cannot convert chunk size: ", chunksize, err)
			}
			conf.Server.Chunksize = &isize
		}
	}
	if conf.Server.Filepath == &exPath {
		log.Fatal("path should be different than executable eg. ./data")
	}
	err = os.MkdirAll(*conf.Server.Filepath, 0600)
	if err != nil {
		log.Fatal(err)
	}
	err = os.Chown(*conf.Server.Filepath, os.Getuid(), os.Getgid())
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	err = initdb()
	if err != nil {
		log.Fatal(err)
	}
	match, _ = regexp.Compile("[a-zA-Z0-9]{64,64}|[a-zA-Z0-9]{6,6}")
	itemplate, err = template.New("index.html").ParseFS(webfs, "www/index.html")
	if err != nil {
		log.Print(err)
	}
	stemplate, err = template.New("string.html").ParseFS(webfs, "www/string.html")
	if err != nil {
		log.Print(err)
	}
	ftemplate, err = template.New("file.html").ParseFS(webfs, "www/file.html")
	if err != nil {
		log.Print(err)
	}
	ctemplate, err = template.New("chat.html").ParseFS(webfs, "www/chat.html")
	if err != nil {
		log.Print(err)
	}
	if *conf.Captcha.Enabled {
		cap.salt = randomString(16)
		go captchaClock()
	}
}

func main() {
	//start the cleanup clock
	go clock()
	//init the server
	mux := http.NewServeMux()
	mux.Handle("/www/", http.HandlerFunc(wwwrouter))
	mux.Handle("/", http.HandlerFunc(serve))
	//start the server!
	if *conf.Server.Proxy {
		err := http.ListenAndServe(":"+strconv.Itoa(*conf.Server.Http), mux)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cpath := path.Join(*conf.Server.Filepath, "certs")
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*conf.Server.Domain),
			Cache:      autocert.DirCache(cpath),
		}
		server := &http.Server{
			Addr:    ":" + strconv.Itoa(*conf.Server.Ssl),
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}
		go http.ListenAndServe(":"+strconv.Itoa(*conf.Server.Http), certManager.HTTPHandler(nil))
		err := server.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatal(err)
		}
	}
}

func clock() {
	//start the clock
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		err := ss.Expire()
		if err != nil {
			log.Print(err)
		}
	}
}

func serve(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("User-Agent"), "facebook") {
		w.WriteHeader(403)
		return
	}
	if r.RequestURI == "/" && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		itemplate.Execute(w, conf)
		return
	}
	code := match.FindStringSubmatch(r.RequestURI)
	if len(code) > 0 {
		r.ParseForm()
		if r.Header.Get("Current-Chunk") != "" {
			getFile(w, r, code[0])
			return
		}
		if r.Method == http.MethodPost {
			saveFile(w, r, code[0])
			return
		}
		if !r.Form.Has("view") {
			w.Header().Set("Content-Type", "text/html")
			data, _ := webfs.ReadFile("www/view.html")
			w.Write(data)
			return
		} else {
			getSecret(w, r, code[0])
			return
		}
	}
	if r.Method == http.MethodPost {
		newSecret(w, r)
		return
	}
}

func wwwrouter(w http.ResponseWriter, r *http.Request) {
	data, err := webfs.ReadFile(r.RequestURI[1:])
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(r.RequestURI)))
	w.Write(data)
	return
}
