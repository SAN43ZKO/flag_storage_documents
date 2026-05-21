package model

import "time"

type File struct {
	ID        int64     `json:"id"`
	Filename  string    `json:"filename"`
	Path      string    `json:"path,omitempty"` // не отдаём наружу
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
