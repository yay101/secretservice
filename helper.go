package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	mrand "math/rand"
)

func randomString(n int) string {
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[mrand.Intn(len(runes))]
	}
	return string(b)
}

func randomStringExtra(n int) string {
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789$-_.+!*'(),")
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[mrand.Intn(len(runes))]
	}
	return string(b)
}

// Encrypt method is to encrypt or hide any classified text
func encryptData(data []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	buf := make([]byte, block.BlockSize())
	_, err = rand.Read(buf)
	if err != nil {
		return nil, err
	}

	cfb := cipher.NewCFBEncrypter(block, buf)
	encrypted := make([]byte, len(data))
	cfb.XORKeyStream(encrypted, data)
	return append(buf, encrypted...), nil
}

// Decrypt method is to extract back the encrypted text
func decryptData(data []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	iv := data[:block.BlockSize()]
	data = data[len(iv):]
	cfb := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(data))
	cfb.XORKeyStream(decrypted, data)
	return decrypted, nil
}
