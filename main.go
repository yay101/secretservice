package main

import (
	"context"
	"crypto/tls"
	"embed"
	"flag"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"encoding/json"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
)

type SecretType string

const (
	Text SecretType = "text"
	File SecretType = "file"
)

var (
	conf  config
	match *regexp.Regexp
	//go:embed www
	webfs     embed.FS
	itemplate *template.Template
	vtemplate *template.Template
	upgrader  = websocket.Upgrader{}
	done      = make(chan struct{})
)

type config struct {
	Domain    string
	Proxy     bool
	Filepath  string
	Key       string
	Ssl       int
	Http      int
	Chunksize int
}

func init() {
	log.SetFlags(log.Lshortfile)
	conf.Filepath = "/mnt/cache"
	conf.Domain = "secretservice.au"
	conf.Http = 80
	conf.Ssl = 443
	conf.Chunksize = 1048576
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exPath := filepath.Dir(ex)
	for _, v := range os.Environ() {
		kv := strings.Split(v, "=")
		switch kv[0] {
		case "SS_DOMAIN":
			conf.Domain = kv[1]
		case "SS_PROXY":
			proxy, err := strconv.ParseBool(kv[1])
			if err != nil {
				log.Fatal(err)
			}
			conf.Proxy = proxy
		case "SS_PATH":
			conf.Filepath = kv[1]
			if conf.Filepath == exPath {
				log.Fatal("path should be different than executable eg. ./data")
			}
		case "SS_PORT":
			port, err := strconv.Atoi(kv[1])
			if err != nil {
				log.Fatal(err)
			}
			conf.Http = port
		case "SS_SSL":
			port, err := strconv.Atoi(kv[1])
			if err != nil {
				log.Fatal(err)
			}
			conf.Ssl = port
		case "SS_CHUNK":
			chunksize, err := strconv.Atoi(kv[1])
			if err != nil {
				log.Fatal(err)
			}
			conf.Chunksize = chunksize
		}
	}
	err = os.MkdirAll(conf.Filepath, 0600)
	if err != nil {
		log.Fatal(err)
	}
	err = os.Chown(conf.Filepath, os.Getuid(), os.Getgid())
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	err = initdb()
	if err != nil {
		log.Fatal(err)
	}
	match = regexp.MustCompile(`\/([a-zA-Z0-9]{64}|[a-zA-Z0-9]{6})\b`)
	itemplate, err = template.New("index.html").ParseFS(webfs, "www/index.html")
	if err != nil {
		log.Fatal(err)
	}
	vtemplate = template.New("view.html").Funcs(template.FuncMap{
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	})
	_, err = vtemplate.ParseFS(webfs, "www/view.html")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	//start the cleanup clock
	go clock()
	//init the server
	mux := http.NewServeMux()
	mux.Handle("/www/", http.HandlerFunc(wwwrouter))
	mux.Handle("/", http.HandlerFunc(serve))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	if conf.Proxy {
		httpServer := &http.Server{Addr: ":" + strconv.Itoa(conf.Http), Handler: mux}
		go func() {
			err := httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
		<-quit
		log.Print("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		close(done)
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		if err := db.Close(); err != nil {
			log.Printf("Database close error: %v", err)
		}
		log.Print("Server stopped")
	} else {
		cpath := path.Join(conf.Filepath, "certs")
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(conf.Domain),
			Cache:      autocert.DirCache(cpath),
		}
		httpsServer := &http.Server{
			Addr:    ":" + strconv.Itoa(conf.Ssl),
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}
		httpServer := &http.Server{
			Addr:    ":" + strconv.Itoa(conf.Http),
			Handler: certManager.HTTPHandler(nil),
		}
		go func() {
			err := httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}()
		go func() {
			err := httpsServer.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
		<-quit
		log.Print("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		close(done)
		if err := httpsServer.Shutdown(ctx); err != nil {
			log.Printf("HTTPS server shutdown error: %v", err)
		}
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		if err := db.Close(); err != nil {
			log.Printf("Database close error: %v", err)
		}
		log.Print("Server stopped")
	}
}

func clock() {
	//start the clock
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			err := ss.Expire()
			if err != nil {
				log.Print(err)
			}
		case <-done:
			ticker.Stop()
			return
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
	if r.Method == http.MethodPost {
		newSecret(w, r)
		return
	}

	code := match.FindStringSubmatch(r.RequestURI)
	if len(code) > 1 {
		r.ParseForm()

		se, err := ss.GetByParentID(code[1])
		if err != nil || len(se) == 0 {
			seByShort, err := ss.GetByShortCode(code[1])
			if err == nil && len(seByShort) > 0 {
				parentID := seByShort[0].ParentID
				iv := seByShort[0].Iv
				redirectURL := "https://" + conf.Domain + "/" + parentID + "/" + iv
				http.Redirect(w, r, redirectURL, http.StatusFound)
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		isWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
		if isWebSocket {
			pathPart := r.URL.Path
			if idx := strings.Index(pathPart, "?"); idx != -1 {
				pathPart = pathPart[:idx]
			}
			parts := strings.Split(pathPart, "/")
			if len(parts) >= 3 {
				parentID := parts[1]
				secretID := parts[2]
				handleWebSocket(w, r, parentID, secretID)
				return
			}
		}

		if r.Form.Has("view") {
			viewSecret(w, r, code[1])
			return
		}

		w.Header().Set("Content-Type", "text/html")
		data, _ := webfs.ReadFile("www/preview.html")
		w.Write(data)
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
}
