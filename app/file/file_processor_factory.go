package file

import (
	"fmt"
	"path/filepath"
	"strings"
)

type FileProcessorFactory struct {
	PdfProcessor   FileProcessor
	ImageProcessor FileProcessor
}

func NewFileProcessorFactory(pdfProcessor FileProcessor, imageProcessor FileProcessor) *FileProcessorFactory {
	return &FileProcessorFactory{
		PdfProcessor:   pdfProcessor,
		ImageProcessor: imageProcessor,
	}
}

func (f *FileProcessorFactory) GetProcessor(filename string) (FileProcessor, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	fmt.Printf("GetProcessor: filename=%s, ext=%s\n", filename, ext)
	switch ext {
	case ".pdf":
		fmt.Println("GetProcessor: Selected PdfProcessor")
		return f.PdfProcessor, nil
	case ".jpg", ".jpeg", ".png", ".webp":
		fmt.Println("GetProcessor: Selected ImageProcessor")
		return f.ImageProcessor, nil
	default:
		fmt.Printf("GetProcessor: Unsupported file type: %s\n", ext)
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}
