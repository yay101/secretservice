package main

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path"

	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
)

type SecretService struct{}

var (
	db *sql.DB
	ss SecretService
)

func initdb() (err error) {
	log.Print("Connecting to db...")
	db, err = sql.Open("sqlite3", "./secrets.db")
	if err != nil {
		return err
	}
	log.Print("Successfully connected to db!")

	_, err = db.Exec("PRAGMA key = " + *conf.Server.Key + ";")
	if err != nil {
		return err
	}
	ssdb, err := db.Prepare("CREATE TABLE IF NOT EXISTS secrets (id TEXT PRIMARY KEY UNIQUE, type TEXT, code TEXT, shortcode TEXT, data TEXT, length INTEGER, expiry DATETIME, short BOOL, recv BOOL, key TEXT, iv TEXT)")
	if err != nil {
		return errors.New("error in preparing table")
	}
	_, err = ssdb.Exec()
	if err != nil {
		return errors.New("error in creating table")
	} else {
		log.Print("Successfully created table secrets!")
	}
	ssdb.Close()
	err = os.Chmod("./secrets.db", 0600)
	if err != nil {
		return err
	}
	return nil
}

func (s *secret) Lock() error {
	stmt, err := db.Prepare("UPDATE secrets SET recv = ? WHERE id = ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(true, s.Id)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretService) ByCode(code string) (*secret, error) {
	se := secret{}
	row := db.QueryRow("SELECT * FROM secrets WHERE (shortcode = ? AND short = ?) OR (code = ?)", code, true, code)
	if row == nil {
		return nil, errors.New("code not found")
	}
	err := row.Scan(&se.Id, &se.Type, &se.Code, &se.ShortCode, &se.Data, &se.Length, &se.Expiry, &se.Short, &se.Recv, &se.Key, &se.Iv)
	if err != nil {
		return nil, err
	}
	return &se, nil
}

func (s *SecretService) Add(se *secret) error {
	_, err := db.Exec("INSERT INTO secrets (id, type, code, shortcode, data, length, expiry, short, recv, key, iv) VALUES(?,?,?,?,?,?,?,?,?,?,?)", se.Id, se.Type, se.Code, se.ShortCode, se.Data, se.Length, se.Expiry, se.Short, se.Recv, se.Key, se.Iv)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretService) Del(se *secret) error {
	_, err := db.Exec("DELETE FROM secrets WHERE id = ?", se.Id)
	if err != nil {
		return err
	}
	if se.Type == File {
		spath := path.Join(*conf.Server.Filepath, se.Id)
		err := os.Remove(spath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SecretService) Expire() error {
	rows, err := db.Query("SELECT * FROM secrets WHERE expiry < datetime('now')")
	if err != nil {
		return err
	}
	if rows == nil {
		return nil
	}
	expired := []secret{}
	for rows.Next() {
		se := secret{}
		err := rows.Scan(&se.Id, &se.Type, &se.Code, &se.ShortCode, &se.Data, &se.Length, &se.Expiry, &se.Short, &se.Recv, &se.Key, &se.Iv)
		if err != nil {
			return err
		}
		expired = append(expired, se)
	}
	for _, se := range expired {
		err := s.Del(&se)
		if err != nil {
			log.Print(err)
		}
	}
	files, err := os.ReadDir(*conf.Server.Filepath)
	if err != nil {
		return err
	}
	for _, file := range files {
		row := db.QueryRow("SELECT * FROM secrets WHERE id = ?", file.Name())
		if row == nil {
			err := os.Remove(path.Join(*conf.Server.Filepath, file.Name()))
			return err
		}
	}
	return nil
}
