package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

type secret struct {
	ID         string     `db:"id,primary"`
	ParentID   string     `db:"index"`
	Type       SecretType `db:"index"`
	Data       string
	Length     uint64
	Uploaded   bool
	Downloaded bool
	Expiry     time.Time
	Key        string
	Iv         string
	ShortCode  string `db:"index"`
}

type SecretRequest struct {
	Short   bool         `json:"short"`
	Expiry  int          `json:"expiry"`
	Secrets []SecretItem `json:"secrets"`
	Key     string       `json:"key"`
	Iv      string       `json:"iv"`
}

type SecretItem struct {
	Type   SecretType `json:"type"`
	Data   string     `json:"data"`
	Length int        `json:"length"`
}

type SecretResponse struct {
	ID       string     `json:"id"`
	Type     SecretType `json:"type"`
	Length   uint64     `json:"length"`
	Uploaded bool       `json:"uploaded,omitempty"`
}

type BatchResponse struct {
	Result  bool             `json:"result"`
	URL     string           `json:"url"`
	Short   string           `json:"short,omitempty"`
	Secrets []SecretResponse `json:"secrets"`
}

func newSecret(w http.ResponseWriter, r *http.Request) {
	var req SecretRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	parentID := randomString(64)
	shortCode := ""
	if req.Short {
		shortCode = randomString(6)
	}

	expiry := time.Now().Add(time.Duration(req.Expiry) * time.Hour)

	var secretsResp []SecretResponse

	for _, item := range req.Secrets {
		secretID := randomString(64)
		scrt := &secret{
			ID:        secretID,
			ParentID:  parentID,
			Type:      item.Type,
			Data:      item.Data,
			Length:    uint64(item.Length),
			Expiry:    expiry,
			ShortCode: shortCode,
		}

		if req.Key != "" {
			scrt.Key = req.Key
			scrt.Iv = req.Iv
		}

		if item.Type == File {
			scrt.Uploaded = false
		} else {
			scrt.Uploaded = true
		}

		err = ss.Add(scrt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		secretsResp = append(secretsResp, SecretResponse{
			ID:       secretID,
			Type:     scrt.Type,
			Length:   scrt.Length,
			Uploaded: scrt.Uploaded,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	baseURL := "https://" + conf.Domain
	if len(secretsResp) > 0 {
		urlPath := parentID
		if req.Iv != "" {
			urlPath = parentID + "/" + req.Iv
		}
		if req.Short {
			json.NewEncoder(w).Encode(&BatchResponse{
				Result:  true,
				URL:     baseURL + "/" + urlPath,
				Short:   baseURL + "/" + shortCode,
				Secrets: secretsResp,
			})
		} else {
			json.NewEncoder(w).Encode(&BatchResponse{
				Result:  true,
				URL:     baseURL + "/" + urlPath,
				Secrets: secretsResp,
			})
		}
	}
}

type WSMessage struct {
	Total   int    `json:"total"`
	Current int    `json:"current"`
	Data    string `json:"data"`
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, parentID string, secretID string) {
	se, err := ss.Get(secretID)
	if err != nil || se.Type != File {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		return
	}
	defer conn.Close()

	fp := path.Join(conf.Filepath, secretID)

	if !se.Uploaded {
		file, err := os.OpenFile(fp, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			log.Print(err)
			return
		}
		defer file.Close()

		for {
			var msg WSMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("Upload: connection closed, err=%v", err)
				break
			}

			data, err := base64.StdEncoding.DecodeString(msg.Data)
			if err != nil {
				log.Print("Failed to decode base64:", err)
				break
			}

			_, err = file.WriteAt(data, int64(msg.Current)*(int64(conf.Chunksize)+16))
			if err != nil {
				log.Print(err)
				break
			}

			conn.WriteJSON(map[string]int{"ack": msg.Current})

			if msg.Current == msg.Total-1 {
				log.Printf("Upload complete for %s", secretID)
				ss.MarkUploaded(secretID)
			}
		}
	} else {
		file, err := os.OpenFile(fp, os.O_RDONLY, 0600)
		if err != nil {
			log.Print(err)
			return
		}

		info, _ := file.Stat()
		totalSize := info.Size()
		chunkSize := int64(conf.Chunksize) + 16
		totalParts := (totalSize + chunkSize - 1) / chunkSize

		for part := int64(0); part < totalParts; part++ {
			buff := make([]byte, chunkSize)
			n, _ := file.ReadAt(buff, part*chunkSize)
			if n > 0 {
				encoded := base64.StdEncoding.EncodeToString(buff[:n])
				conn.WriteJSON(WSMessage{
					Total:   int(totalParts),
					Current: int(part),
					Data:    encoded,
				})
			}
		}
		file.Close()
		ss.MarkDownloaded(secretID)
		ss.Del(se)
		os.Remove(fp)

		all, err := ss.GetByParentID(se.ParentID)
		if err == nil {
			filesRemaining := false
			for _, s := range all {
				if s.Type == File && !s.Downloaded {
					filesRemaining = true
					break
				}
			}
			if !filesRemaining {
				ss.DelByParentID(se.ParentID)
			}
		}

		log.Printf("Download complete and cleaned up for %s", secretID)
	}
}

func viewSecret(w http.ResponseWriter, r *http.Request, parentID string) {
	secrets, err := ss.GetByParentID(parentID)
	if err != nil || len(secrets) == 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	first := secrets[0]

	data := struct {
		ParentID string
		Key      string
		Iv       string
		Secrets  []secret
	}{
		ParentID: parentID,
		Key:      first.Key,
		Iv:       first.Iv,
		Secrets:  secrets,
	}

	log.Printf("viewSecret: id=%s type=%s", first.ID, first.Type)
	err = vtemplate.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
