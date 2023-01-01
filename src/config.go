package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Server   Server    `json:"server"`
	Database string    `json:"database"`
	Captcha  Recaptcha `json:"captcha"`
}

type Server struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Domain  string `json:"domain"`
	GetKey  string `json:"getkey"`
	PostKey string `json:"postkey"`
}

func (c *Config) Init() {
	c = &Config{
		Server: Server{
			Name:    serverName,
			Port:    3001,
			Domain:  "secretservice.au",
			GetKey:  uuid.New().String(),
			PostKey: uuid.New().String(),
		},
		Database: "secrets.db",
	}
	c.Save()
}

func (c *Config) Load() {
	trypath := []string{path.Join("/etc/", serverName, serverName) + ".yaml", path.Join(ownPath, serverName) + ".yaml"}
	for i, location := range trypath {
		yamlFile, err := os.Open(location)
		if err != nil {
			log.Print(err)
			if i == len(trypath)-1 {
				log.Print("There are no configuration files. I will try and create one for you with the default settings here: " + location)
				configLocation = location
				config.Init()
				break
			}
			continue
		} else {
			log.Print("Opened config: " + location)
			defer yamlFile.Close()
			byteValue, _ := io.ReadAll(yamlFile)
			yaml.Unmarshal(byteValue, &c)
		}
	}
}

func (c *Config) Save() {
	yamlString, _ := yaml.Marshal(config)
	err := os.WriteFile(configLocation, yamlString, 0600)
	if err != nil {
		log.Print(err)
	}
}

func (c *Config) Hook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonbytes, err := json.Marshal(&c)
		if err != nil {
			log.Print(err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(jsonbytes)
	case http.MethodPost:
		err := json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			log.Print(err)
			break
		}
		config.Save()
	}
}
