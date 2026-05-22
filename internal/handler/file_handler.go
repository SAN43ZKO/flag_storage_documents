package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"storage_files/internal/converter"
	"storage_files/internal/office"
	"storage_files/internal/service"
)

type FileHandler struct {
	svc              *service.FileService
	onlyOfficeAPIURL string
}

func NewFileHandler(svc *service.FileService, onlyOfficeAPIURL string) *FileHandler {
	return &FileHandler{svc: svc, onlyOfficeAPIURL: onlyOfficeAPIURL}
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
	// Конфигурация для OnlyOffice
	mux.HandleFunc("GET /api/files/{id}/edit", h.GetEditorConfig)
	// Callback от OnlyOffice после редактирования
	mux.HandleFunc("POST /api/files/{id}/callback", h.Callback)
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

// GetEditorConfig возвращает конфигурацию для OnlyOffice
func (h *FileHandler) GetEditorConfig(w http.ResponseWriter, r *http.Request) {
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

	downloadURL := h.svc.GetDownloadURL(id)
	callbackURL := h.svc.GetCallbackURL(id)

	config := office.GenerateConfig(id, f.Filename, downloadURL, callbackURL, "edit")
	writeJSON(w, http.StatusOK, config)
}

// Callback принимает сохранённый файл от OnlyOffice
func replaceHost(originalURL, newBase string) (string, error) {
	u, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(newBase)
	if err != nil {
		return "", err
	}
	u.Scheme = base.Scheme
	u.Host = base.Host
	return u.String(), nil
}

func (h *FileHandler) Callback(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	status, _ := body["status"].(float64)
	if status != 2 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"error": 0})
		return
	}

	fileURL, ok := body["url"].(string)
	if !ok {
		writeError(w, http.StatusBadRequest, "no url")
		return
	}

	// Заменяем хост в URL на адрес OnlyOffice из конфига
	fixedURL, err := replaceHost(fileURL, h.onlyOfficeAPIURL)
	if err != nil {
		log.Printf("replaceHost: %v", err)
		writeError(w, http.StatusInternalServerError, "bad url")
		return
	}

	resp, err := http.Get(fixedURL)
	if err != nil {
		log.Printf("download edited file: %v", err)
		writeError(w, http.StatusInternalServerError, "download failed")
		return
	}
	defer resp.Body.Close()

	f, err := h.svc.GetFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	dst, err := os.Create(f.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, resp.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "copy failed")
		return
	}

	if err := h.svc.UpdateAfterEdit(r.Context(), id, f.Path, size); err != nil {
		log.Printf("update after edit: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"error": 0})
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
