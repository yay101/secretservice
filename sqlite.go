package main

import (
	"database/sql"
	"log"
	"os"
	"path"
	"strconv"
	"time"

	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
)

var db *sql.DB

func dbinit() {
	var err error
	db, err = sql.Open("sqlite3", path.Join("./", config.Database.Name)+"?_crypto_key="+config.Database.Key)
	if err != nil {
		log.Fatal("Error in creating db")
		return
	} else {
		log.Print("Successfully connected to db!")
		os.Chmod(path.Join("./", config.Database.Name), 0700)
	}
	//prepare secrets table
	secretdb, err := db.Prepare("CREATE TABLE IF NOT EXISTS secrets (id INTEGER PRIMARY KEY, type TEXT, code TEXT, shortcode TEXT, secret TEXT, download BOOL, hidden BOOL, short BOOL, life INTEGER, blob BLOB, expiry DATETIME)")
	if err != nil {
		log.Fatal("Error in creating table")
	} else {
		log.Print("Successfully created table secrets!")
	}
	secretdb.Exec()
}

func (s *Secret) Add() bool {
	res, err := db.Exec("INSERT INTO secrets (type, code, shortcode, secret, download, hidden, short, life, blob, expiry) VALUES(?,?,?,?,?,?,?,?,?,?)", s.Type, s.Code, s.ShortCode, s.Secret, s.Download, s.Hidden, s.Short, s.Life, s.Blob, s.Expiry)
	if err != nil {
		log.Print(err)
		return false
	} else {
		ins, _ := res.LastInsertId()
		log.Print("Keeping a secret!")
		log.Print("Inserted at: " + strconv.FormatInt(ins, 10))
	}
	return true
}

func (s *Secret) Get() bool {
	var row *sql.Row
	if s.ShortCode != "" {
		row = db.QueryRow("SELECT * FROM secrets WHERE shortcode = ? AND short = ?", s.ShortCode, true)
	} else {

		row = db.QueryRow("SELECT * FROM secrets WHERE code = ?", s.Code)
	}
	err := row.Scan(&s.Id, &s.Type, &s.Code, &s.ShortCode, &s.Secret, &s.Download, &s.Hidden, &s.Short, &s.Life, &s.Blob, &s.Expiry)
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func (s *Secret) Delete() {
	res, err := db.Exec("DELETE FROM secrets WHERE code=?", s.Code)
	if err != nil {
		log.Print(res)
		log.Print(err)
	}
	row, _ := res.RowsAffected()
	log.Print("Deleted: " + strconv.FormatInt(row, 10))
}

func dbClean() {
	rows, err := db.Query("SELECT * FROM secrets WHERE expiry < ?", time.Now().Local())
	if err != nil {
		log.Print(err)
	}
	defer rows.Close()
	var oldSecrets Secrets
	for rows.Next() {
		var tmp Secret
		err = rows.Scan(&tmp.Id, &tmp.Type, &tmp.Code, &tmp.ShortCode, &tmp.Secret, &tmp.Download, &tmp.Hidden, &tmp.Short, &tmp.Life, &tmp.Blob, &tmp.Expiry)
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
