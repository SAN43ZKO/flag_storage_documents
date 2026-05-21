package office

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"
)

type Document struct {
	FileType string `json:"fileType"`
	Key      string `json:"key"`
	Title    string `json:"title"`
	URL      string `json:"url"`
}

type EditorConfig struct {
	CallbackURL string `json:"callbackUrl"`
	Lang        string `json:"lang"`
	Mode        string `json:"mode"` // "edit" или "view"
	User        User   `json:"user"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Config struct {
	Document     Document     `json:"document"`
	EditorConfig EditorConfig `json:"editorConfig"`
	Height       string       `json:"height"`
	Width        string       `json:"width"`
	Type         string       `json:"type"` // "desktop" или "embedded"
}

// GenerateConfig создаёт конфигурацию для редактора
func GenerateConfig(fileID int64, filename, downloadURL, callbackURL string, mode string) Config {
	// Генерируем уникальный ключ документа (меняется при изменении файла)
	key := generateKey(fileID, filename)
	return Config{
		Document: Document{
			FileType: getFileType(filename),
			Key:      key,
			Title:    filename,
			URL:      downloadURL,
		},
		EditorConfig: EditorConfig{
			CallbackURL: callbackURL,
			Lang:        "ru",
			Mode:        mode,
			User: User{
				ID:   "user1",
				Name: "User",
			},
		},
		Height: "100%",
		Width:  "100%",
		Type:   "desktop",
	}
}

func getFileType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".doc":
		return "doc"
	case ".docx":
		return "docx"
	default:
		return "docx" // по умолчанию
	}
}

func generateKey(id int64, filename string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d_%s_%d", id, filename, time.Now().Unix())))
	return hex.EncodeToString(h[:])
}
