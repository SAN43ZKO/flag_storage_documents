package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"storage_files/internal/converter"
	"storage_files/internal/service"
)

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) RegisterRoutes(mux *http.ServeMux) {
	// Загрузка файла
	mux.HandleFunc("POST /api/files/upload", h.Upload)
	// Список файлов
	mux.HandleFunc("GET /api/files", h.List)
	// Скачивание оригинального файла
	mux.HandleFunc("GET /api/files/{id}/download", h.Download)
	// Предпросмотр PDF
	mux.HandleFunc("GET /api/files/{id}/preview", h.Preview)
	// Удаление файла
	mux.HandleFunc("DELETE /api/files/{id}", h.Delete)
}

// Upload загружает файл на сервер
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file")
		return
	}
	defer file.Close()

	f, err := h.svc.SaveFile(r.Context(), header.Filename, file) // передаём оригинальное имя
	if err != nil {
		log.Printf("upload error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// List возвращает список всех файлов
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	files, err := h.svc.ListFiles(r.Context(), search)
	if err != nil {
		log.Printf("list error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, files)
}

// Download отдаёт оригинальный файл
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	f, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", f.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, f.Filename))
	http.ServeFile(w, r, f.Path)
}

// Preview конвертирует документ в PDF и отдаёт его
func (h *FileHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	f, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	// Создаём временную папку для конвертации
	tmpDir := os.TempDir()
	pdfPath, err := converter.ConvertToPDF(f.Path, tmpDir)
	if err != nil {
		log.Printf("conversion error: %v", err)
		writeError(w, http.StatusInternalServerError, "conversion failed")
		return
	}
	defer os.Remove(pdfPath)

	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, pdfPath)
}

// Delete удаляет файл
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteFile(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- вспомогательные функции ---
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
