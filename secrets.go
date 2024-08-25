package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type secret struct {
	Id        string
	Type      SecretType
	Code      string
	ShortCode string
	Data      string
	Length    uint64
	Expiry    time.Time
	Short     bool
	Recv      bool
	Key       string
	Iv        string
}

type response struct {
	Result bool   `json:"result"`
	Url    string `json:"url"`
}

func getSecret(w http.ResponseWriter, r *http.Request, code string) {
	w.Header().Set("Content-Type", "text/html")
	secret, err := ss.ByCode(code)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	switch secret.Type {
	case String:
		err := stemplate.Execute(w, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ss.Del(secret)
		return
	case File:
		err := ftemplate.Execute(w, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		return
	case Chat:
		err := ctemplate.Execute(w, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		return
	}
}

func newSecret(w http.ResponseWriter, r *http.Request) {
	new := incoming{}
	err := json.NewDecoder(r.Body).Decode(&new)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scrt := secret{
		Id:        uuid.New().String(),
		Code:      randomString(64),
		ShortCode: randomString(6),
		Expiry:    time.Now().Add(time.Duration(new.Expiry) * time.Hour),
		Data:      new.Data,
		Length:    new.Length,
		Type:      new.Type,
		Short:     new.Short,
		Key:       new.Key,
		Iv:        new.Iv,
	}
	err = ss.Add(&scrt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if scrt.Short {
		json.NewEncoder(w).Encode(&response{Result: true, Url: "https://" + *conf.Server.Domain + "/" + scrt.ShortCode})
	} else {
		json.NewEncoder(w).Encode(&response{Result: true, Url: "https://" + *conf.Server.Domain + "/" + scrt.Code})
	}
	return
}

func getFile(w http.ResponseWriter, r *http.Request, code string) {
	secret, err := ss.ByCode(code)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	fp := path.Join(*conf.Server.Filepath, secret.Id)
	file, err := os.OpenFile(fp, os.O_RDONLY, 0600)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	defer file.Close()
	buff := make([]byte, *conf.Server.Chunksize+16)
	i, err := strconv.Atoi(r.Header.Get("Current-Chunk"))
	l, err := file.ReadAt(buff, int64(i*(*conf.Server.Chunksize+16)))
	if err == io.EOF {
		defer ss.Del(secret)
	}
	if l == *conf.Server.Chunksize+16 || err == io.EOF {
		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Length", strconv.Itoa(l))
		w.Write(buff[:l])
		return
	}
}

func saveFile(w http.ResponseWriter, r *http.Request, code string) {
	secret, err := ss.ByCode(code)
	if err != nil || secret.Recv {
		http.Error(w, "not found or expired", http.StatusNotFound)
		return
	}
	length := r.ContentLength
	cd := strings.Split(r.Header.Get("Content-Range"), "/")
	chunk, err := strconv.Atoi(cd[0])
	if err != nil {
		http.Error(w, "Invalid Range Header", http.StatusBadRequest)
		return
	}
	total, err := strconv.Atoi(cd[1])
	if err != nil {
		http.Error(w, "Invalid Range Header", http.StatusBadRequest)
		return
	}
	fp := path.Join(*conf.Server.Filepath, secret.Id)
	file, err := os.OpenFile(fp, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		http.Error(w, "couldnt open file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "couldnt read chunk", http.StatusBadRequest)
		return
	}
	wl, err := file.WriteAt(b, int64(chunk)*int64(*conf.Server.Chunksize+16))
	if int64(wl) != length || err != nil {
		http.Error(w, "couldnt write chunk", http.StatusBadRequest)
		return
	}
	if chunk == total {
		secret.Lock()
	}
	return
}
