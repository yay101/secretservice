package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"hash"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	// Salt is a random string used to generate the challenge.
	// The minimum length is 10 characters.
	Salt string `json:"salt"`
	// Number is the secret number which the client must solve for.
	Number int `json:"number,omitempty"`
	// Challenge is the hash which the client must solve for.
	// The minimum length is 40 characters.
	Challenge string `json:"challenge"`
	// Signature is the signature of the challenge.
	Signature string `json:"signature"`
}

const (
	defaultBanSliceSize = 10
)

type captcha struct {
	lock     sync.RWMutex
	salt     string
	current  captchasecret
	previous captchasecret
}

type captchasecret struct {
	secret string
	banned []string
}

var cap captcha

func captchaClock() {
	rot := time.NewTicker(5 * time.Minute)
	for {
		<-rot.C
		rotateSecrets()
	}
}

func rotateSecrets() {
	cap.lock.Lock()
	defer cap.lock.Unlock()
	cap.previous = cap.current
	cap.current = captchasecret{secret: randomString(32), banned: []string{}}
}

func randomNumber() int {
	return rand.Intn(*conf.Captcha.DefaultComplexity-*conf.Captcha.MinimumComplexity) + *conf.Captcha.MinimumComplexity
}

func generateHash(number int) string {
	hasher := sha512.New()
	hasher.Write([]byte(cap.salt))
	hasher.Write([]byte(strconv.Itoa(number)))
	return hex.EncodeToString(hasher.Sum(nil))
}

func newMessage() *Message {
	msg := Message{
		Salt:   cap.salt,
		Number: randomNumber(),
		// Number is a secret and must not be exposed to the client.
	}
	msg.Challenge = generateHash(msg.Number)
	msg.Signature = sign(msg.Challenge, cap.current.secret)
	return &msg
}

func (m *Message) Encode() []byte {
	jsonBytes, _ := json.Marshal(m)
	return jsonBytes
}

func (m *Message) Validate() bool {
	if m.Number <= 0 {
		return false
	}
	if m.Challenge != generateHash(m.Number) {
		return false
	}
	if !VerifySignature(m.Challenge, m.Signature) {
		return false
	}
	if IsSignatureBanned(m.Signature) {
		return false
	}
	BanSignature(m.Signature)
	return true // Success!
}

// BanSignature adds the given signature to the list of banned signatures.
func BanSignature(signature string) {
	if len(signature) == 0 {
		return
	}
	cap.lock.Lock()
	defer cap.lock.Unlock()
	sigbyte := []byte(signature)
	cap.current.banned = append(cap.current.banned, string(sigbyte[:10]))
	return
}

// IsSignatureBanned checks if the given signature is banned.
func IsSignatureBanned(signature string) bool {
	if len(signature) == 0 {
		return true // blank signature
	}
	cap.lock.Lock()
	defer cap.lock.Unlock()
	for _, entry := range cap.current.banned {
		if entry == signature {
			return true
		}
	}
	return false
}

func sign(text, secret string) string {
	newHasher := func() (hasher hash.Hash) {
		return sha512.New()
	}
	signer := hmac.New(newHasher, []byte(secret))
	signer.Write([]byte(text))
	return base64.RawURLEncoding.EncodeToString(signer.Sum(nil))
}

// VerifySignature checks if the given signature is valid for the given text.
func VerifySignature(text, signature string) (valid bool) {
	if len(signature) == 0 {
		return false
	}
	validSignature := sign(text, cap.current.secret)
	if signature == validSignature {
		return true
	}
	validSignature = sign(text, cap.previous.secret)
	if signature == validSignature {
		return true
	}
	return false
}
