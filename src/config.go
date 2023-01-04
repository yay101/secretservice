package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Server   ServerSettings   `json:"server"`
	Database DatabaseSettings `json:"database"`
	Captcha  Recaptcha        `json:"captcha"`
}

type ServerSettings struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Domain string `json:"domain"`
	ApiKey string `json:"apikey"`
}

type DatabaseSettings struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

func (c *Config) Init() {
	c = &Config{
		Server: ServerSettings{
			Name:   serverName,
			Port:   3001,
			Domain: "secretservice.au",
			ApiKey: strings.Join(strings.Split(uuid.New().String(), "-"), ""),
		},
		Database: DatabaseSettings{
			Name: "secrets.db",
			Key:  strings.Join(strings.Split(uuid.New().String(), "-"), ""),
		},
	}
	c.Save()
}

func (c *Config) Load() {
	var env bool
	if c.Server.Name, env = os.LookupEnv("server_name"); env {
		log.Print("Found environmental variable.")
		port, err := strconv.Atoi(os.Getenv("server_port"))
		if err != nil {
			log.Print("Error getting port from env: " + err.Error())
		}
		c = &Config{
			Server: ServerSettings{
				Name:   os.Getenv("server_name"),
				Port:   port,
				Domain: os.Getenv("server_domain"),
				ApiKey: os.Getenv("server_apikey"),
			},
			Database: DatabaseSettings{
				Name: os.Getenv("database_name"),
				Key:  os.Getenv("database_key"),
			},
		}
		return
	}
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
