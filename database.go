package main

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/yay101/embeddb"
)

type SecretService struct{}

var (
	db      *embeddb.DB
	ss      SecretService
	secrets *embeddb.Table[secret]
)

func initdb() (err error) {
	dbPath := "./secrets.db"
	db, err = embeddb.Open(dbPath)
	if err != nil {
		return err
	}
	log.Print("Successfully connected to db!")

	secrets, err = embeddb.Use[secret](db, "secrets")
	if err != nil {
		return err
	}
	log.Print("Successfully created/connected to secrets table!")

	err = os.Chmod(dbPath, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretService) Add(se *secret) error {
	_, err := secrets.Insert(se)
	return err
}

func (s *SecretService) Get(id string) (*secret, error) {
	return secrets.Get(id)
}

func (s *SecretService) GetByParentID(parentID string) ([]secret, error) {
	return secrets.Filter(func(se secret) bool {
		return se.ParentID == parentID
	})
}

func (s *SecretService) GetByShortCode(shortCode string) ([]secret, error) {
	if shortCode == "" {
		return nil, nil
	}
	return secrets.Filter(func(se secret) bool {
		return se.ShortCode == shortCode
	})
}

func (s *SecretService) MarkUploaded(id string) error {
	se, err := secrets.Get(id)
	if err != nil {
		return err
	}
	se.Uploaded = true
	return secrets.Update(id, se)
}

func (s *SecretService) MarkDownloaded(id string) error {
	se, err := secrets.Get(id)
	if err != nil {
		return err
	}
	se.Downloaded = true
	return secrets.Update(id, se)
}

func (s *SecretService) Del(se *secret) error {
	err := secrets.Delete(se.ID)
	if err != nil {
		return err
	}
	if se.Type == File {
		spath := path.Join(conf.Filepath, se.ID)
		err := os.Remove(spath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SecretService) DelByParentID(parentID string) error {
	all, err := secrets.Filter(func(se secret) bool {
		return se.ParentID == parentID
	})
	if err != nil {
		return err
	}
	for _, se := range all {
		err = s.Del(&se)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SecretService) Expire() error {
	now := time.Now()
	var expired []secret

	scanner := secrets.ScanRecords()
	defer scanner.Close()

	for scanner.Next() {
		se, _ := scanner.Record()
		if se.Expiry.Before(now) {
			expired = append(expired, *se)
		}
	}

	for _, se := range expired {
		err := s.Del(&se)
		if err != nil {
			log.Print(err)
		}
	}

	files, err := os.ReadDir(conf.Filepath)
	if err != nil {
		return err
	}
	for _, file := range files {
		result, err := secrets.Get(file.Name())
		if err != nil || result == nil {
			err := os.Remove(path.Join(conf.Filepath, file.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
