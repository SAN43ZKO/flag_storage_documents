package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"storage_files/internal/model"
	"storage_files/internal/repository"
)

type FileService struct {
	repo      *repository.FileRepo
	uploadDir string
	baseURL   string
}

func NewFileService(repo *repository.FileRepo, uploadDir, baseURL string) *FileService {
	return &FileService{repo: repo, uploadDir: uploadDir, baseURL: baseURL}
}

func (s *FileService) SaveFile(ctx context.Context, originalName string, reader io.Reader) (model.File, error) {
	ext := filepath.Ext(originalName)
	base := originalName[:len(originalName)-len(ext)]
	uniqueName := fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
	filePath := filepath.Join(s.uploadDir, uniqueName)

	dst, err := os.Create(filePath)
	if err != nil {
		return model.File{}, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	size, err := io.Copy(dst, reader)
	if err != nil {
		os.Remove(filePath)
		return model.File{}, fmt.Errorf("copy file: %w", err)
	}

	mimeType := "application/octet-stream"
	switch ext {
	case ".doc":
		mimeType = "application/msword"
	case ".docx":
		mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	// В БД сохраняем оригинальное имя
	return s.repo.Create(ctx, originalName, filePath, size, mimeType)
}

func (s *FileService) GetFile(ctx context.Context, id int64) (model.File, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *FileService) ListFiles(ctx context.Context, search string) ([]model.File, error) {
	return s.repo.List(ctx, search)
}

func (s *FileService) DeleteFile(ctx context.Context, id int64) error {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := os.Remove(f.Path); err != nil {
		return fmt.Errorf("remove file: %w", err)
	}
	return s.repo.Delete(ctx, id)
}

func (s *FileService) GetDownloadURL(id int64) string {
	return fmt.Sprintf("%s/api/files/%d/download", s.baseURL, id)
}

func (s *FileService) GetCallbackURL(id int64) string {
	return fmt.Sprintf("%s/api/files/%d/callback", s.baseURL, id)
}

func (s *FileService) UpdateAfterEdit(ctx context.Context, id int64, newPath string, newSize int64) error {
	return s.repo.UpdateAfterEdit(ctx, id, newPath, newSize)
}
