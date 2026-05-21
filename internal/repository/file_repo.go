package repository

import (
	"context"
	"fmt"
	"time"

	"storage_files/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileRepo struct {
	pool *pgxpool.Pool
}

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) Create(ctx context.Context, filename, path string, size int64, mimeType string) (model.File, error) {
	var f model.File
	now := time.Now().UTC()
	err := r.pool.QueryRow(ctx,
		`INSERT INTO files (filename, path, size, mime_type, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id, filename, path, size, mime_type, created_at, updated_at`,
		filename, path, size, mimeType, now, now,
	).Scan(&f.ID, &f.Filename, &f.Path, &f.Size, &f.MimeType, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return model.File{}, fmt.Errorf("insert file: %w", err)
	}
	return f, nil
}

func (r *FileRepo) GetByID(ctx context.Context, id int64) (model.File, error) {
	var f model.File
	err := r.pool.QueryRow(ctx,
		`SELECT id, filename, path, size, mime_type, created_at, updated_at
         FROM files WHERE id = $1`, id,
	).Scan(&f.ID, &f.Filename, &f.Path, &f.Size, &f.MimeType, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.File{}, fmt.Errorf("file not found")
		}
		return model.File{}, fmt.Errorf("get file: %w", err)
	}
	return f, nil
}

func (r *FileRepo) List(ctx context.Context, search string) ([]model.File, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if search == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT id, filename, path, size, mime_type, created_at, updated_at
             FROM files ORDER BY id`)
	} else {
		like := "%" + search + "%"
		rows, err = r.pool.Query(ctx,
			`SELECT id, filename, path, size, mime_type, created_at, updated_at
             FROM files
             WHERE filename ILIKE $1
             ORDER BY id`, like)
	}
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.Filename, &f.Path, &f.Size, &f.MimeType, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *FileRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM files WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("file not found")
	}
	return nil
}

func (r *FileRepo) UpdateAfterEdit(ctx context.Context, id int64, newPath string, newSize int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE files SET path=$1, size=$2, updated_at=NOW() WHERE id=$3`,
		newPath, newSize, id)
	return err
}
