package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type VerifyResponse struct {
	Result    bool      `json:"success"`
	Score     float64   `json:"score"`
	Timestamp time.Time `json:"challenge_ts"`
	Hostname  string    `json:"hostname"`
	Errors    []string  `json:"error-codes"`
}

type Recaptcha struct {
	Enabled   bool    `json:"enabled"`
	SiteKey   string  `json:"sitekey"`
	SecretKey string  `json:"secretkey"`
	Score     float64 `json:"score"`
}

func recaptcha(response string, remoteip string) bool {
	if !config.Captcha.Enabled {
		log.Print("Bypassing captcha.")
		return true
	}
	data := url.Values{
		"secret":   {config.Captcha.SecretKey},
		"response": {response},
	}
	req, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", data)
	if err != nil {
		log.Print(err)
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Print(err)
		return false
	}
	Response := VerifyResponse{}
	json.Unmarshal(body, &Response)
	if Response.Score > config.Captcha.Score {
		return true
	} else {
		return false
	}
}
