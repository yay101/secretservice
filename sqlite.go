package main

import (
	"database/sql"
	"log"
	"os"
	"path"
	"time"

	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
)

func dbinit() {
	db, err := sql.Open("sqlite3", path.Join(ownPath, config.Database.Name)+"?_crypto_key="+config.Database.Key)
	if err != nil {
		log.Println("Error in creating db")
		return
	} else {
		log.Println("Successfully connected to db!")
		os.Chmod(path.Join(ownPath, config.Database.Name), 0700)
	}
	defer db.Close()
	//prepare secrets table
	secretdb, err := db.Prepare("CREATE TABLE IF NOT EXISTS secrets (id INTEGER PRIMARY KEY, type TEXT, shortcode TEXT, code TEXT, code2 TEXT, secret TEXT, download BOOL, hidden BOOL, short BOOL, life INTEGER, expiry INTEGER)")
	if err != nil {
		log.Println("Error in creating table")
	} else {
		log.Println("Successfully created table secrets!")
	}
	secretdb.Exec()
}

func (s *Secret) Add() bool {
	mu.Lock()
	defer mu.Unlock()
	db, err := sql.Open("sqlite3", path.Join(ownPath, config.Database.Name)+"?_crypto_key="+config.Database.Key)
	if err != nil {
		log.Println("Error in connecting db")
		return false
	}
	defer db.Close()
	res, err := db.Exec("INSERT INTO secrets (type, shortcode, code, code2, secret, download, hidden, short, life, expiry) VALUES(?,?,?,?,?,?,?,?,?,?)", s.Type, s.ShortCode, s.Code, s.Code2, s.Secret, s.Download, s.Hidden, s.Short, s.Life, s.Expiry)
	if err != nil {
		log.Print(err)
		return false
	} else {
		log.Print("Keeping a secret!")
		log.Print(res)
	}
	return true
}

func (s *Secret) Get() bool {
	mu.Lock()
	defer mu.Unlock()
	db, err := sql.Open("sqlite3", path.Join(ownPath, config.Database.Name)+"?_crypto_key="+config.Database.Key)
	if err != nil {
		log.Println("Error in connecting db")
		return false
	}
	defer db.Close()
	var row *sql.Row
	if s.ShortCode != "" {
		row = db.QueryRow("SELECT * FROM secrets WHERE shortcode=? AND short=?", s.ShortCode, true)
	} else {
		row = db.QueryRow("SELECT * FROM secrets WHERE code=? AND code2=?", s.Code, s.Code2)
	}
	err = row.Scan(&s.Id, &s.Type, &s.ShortCode, &s.Code, &s.Code2, &s.Secret, &s.Download, &s.Hidden, &s.Short, &s.Life, &s.Expiry)
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func (s *Secret) Delete() {
	mu.Lock()
	defer mu.Unlock()
	db, err := sql.Open("sqlite3", path.Join(ownPath, config.Database.Name)+"?_crypto_key="+config.Database.Key)
	if err != nil {
		log.Println("Error in connecting db")
		return
	}
	defer db.Close()
	res, err := db.Exec("delete from secrets where code = ?", s.Code)
	if err != nil {
		log.Print(res)
		log.Print(err)
	}
	if s.Type != "string" {
		err := os.RemoveAll(path.Join(ownPath, "blobs", s.Code))
		if err != nil {
			log.Print()
		}
	}
}

func dbClean() {
	mu.Lock()
	db, err := sql.Open("sqlite3", path.Join(ownPath, config.Database.Name)+"?_crypto_key="+config.Database.Key)
	if err != nil {
		log.Println("Error in connecting db: " + err.Error())
		return
	}
	rows, err := db.Query("SELECT * FROM secrets WHERE expiry < ?", time.Now().Local().UnixMilli())
	db.Close()
	mu.Unlock()
	if err != nil {
		log.Print(err)
	}
	defer rows.Close()
	var oldSecrets Secrets
	for rows.Next() {
		var tmp Secret
		err = rows.Scan(&tmp.Id, &tmp.Type, &tmp.ShortCode, &tmp.Code, &tmp.Code2, &tmp.Secret, &tmp.Download, &tmp.Hidden, &tmp.Short, &tmp.Life, &tmp.Expiry)
		if err != nil {
			log.Print(err)
		} else {
			oldSecrets = append(oldSecrets, tmp)
		}
	}
	for _, s := range oldSecrets {
		s.Delete()
	}
}
