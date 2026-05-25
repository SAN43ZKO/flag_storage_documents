package wopi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"storage_files/internal/service"
)

type FileInfo struct {
	BaseFileName      string `json:"BaseFileName"`
	OwnerId           string `json:"OwnerId"`
	Size              int64  `json:"Size"`
	UserId            string `json:"UserId"`
	UserFriendlyName  string `json:"UserFriendlyName"`
	UserCanWrite      bool   `json:"UserCanWrite"`
	LastModifiedTime  string `json:"LastModifiedTime"`
	PostMessageOrigin string `json:"PostMessageOrigin"`
	EnableShare       bool   `json:"EnableShare"`
}

type Handler struct {
	svc       *service.FileService
	publicURL string
}

func NewHandler(svc *service.FileService, publicURL string) *Handler {
	return &Handler{svc: svc, publicURL: publicURL}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /wopi/files/{id}", h.CheckFileInfo)
	mux.HandleFunc("GET /wopi/files/{id}/contents", h.GetFile)
	mux.HandleFunc("POST /wopi/files/{id}/contents", h.PutFile)
}

// CheckFileInfo: возвращает метаданные файла в формате WOPI
func (h *Handler) CheckFileInfo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	f, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	info := FileInfo{
		BaseFileName:      f.Filename,
		OwnerId:           "owner",
		Size:              f.Size,
		UserId:            "user",
		UserFriendlyName:  "Пользователь",
		UserCanWrite:      true,
		LastModifiedTime:  f.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
		PostMessageOrigin: h.publicURL,
		EnableShare:       false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// GetFile: отдаёт содержимое файла
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	f, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.MimeType)
	http.ServeFile(w, r, f.Path)
}

// PutFile: сохраняет изменённый файл
func (h *Handler) PutFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	f, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	// Перезаписываем файл
	dst, err := os.Create(f.Path)
	if err != nil {
		log.Printf("put file: %v", err)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, r.Body)
	if err != nil {
		log.Printf("put file: %v", err)
		http.Error(w, "copy failed", http.StatusInternalServerError)
		return
	}

	// Обновляем размер в БД
	if err := h.svc.UpdateAfterEdit(r.Context(), id, f.Path, size); err != nil {
		log.Printf("update after edit: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}
