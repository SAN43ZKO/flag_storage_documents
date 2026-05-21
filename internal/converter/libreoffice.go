package converter

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertToPDF конвертирует файл (doc, docx) в PDF и возвращает путь к PDF.
func ConvertToPDF(inputPath, outputDir string) (string, error) {
	cmd := exec.Command("libreoffice",
		"--headless",
		"--convert-to", "pdf",
		"--outdir", outputDir,
		inputPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		log.Printf("libreoffice failed: %s", out.String())
		return "", fmt.Errorf("libreoffice conversion failed: %w", err)
	}

	// Ищем PDF‑файл, созданный LibreOffice. Обычно имя = basename без расширения + .pdf
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	pdfName := strings.TrimSuffix(base, ext) + ".pdf"
	pdfPath := filepath.Join(outputDir, pdfName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		// Иногда LibreOffice добавляет что-то к имени, попробуем найти любой pdf в каталоге
		files, _ := filepath.Glob(filepath.Join(outputDir, "*.pdf"))
		if len(files) > 0 {
			return files[0], nil
		}
		log.Printf("expected pdf not found: %s, dir contents: %v", pdfPath, listDir(outputDir))
		return "", fmt.Errorf("pdf not generated")
	}
	return pdfPath, nil
}

func listDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}
