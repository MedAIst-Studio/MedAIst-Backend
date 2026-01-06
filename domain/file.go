package domain

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Name      string
	Size      int64
	Path      string
	CreatedAt time.Time
}

type FileRequires struct {
	UserId    uuid.UUID
	Name      string
	Size      int64
	Extension string
}

func NewFile(fr *FileRequires) (*File, error) {

	if !isSupportedFile(fr.Extension) {
		return nil, ErrNotPDFFile // Note: We might want to rename this error later globally, but keeping for now to minimize ripple or assuming generic name.
	}

	if !isSizeValid(fr.Size) {
		return nil, ErrFileSizeExceeded
	}
	id := uuid.New()
	return &File{
		Id:        id,
		UserId:    fr.UserId,
		Name:      fr.Name,
		Size:      fr.Size,
		Path:      "uploads/" + id.String() + cleanExtension(fr.Extension),
		CreatedAt: time.Now(),
	}, nil
}

func isSupportedFile(extension string) bool {
	ext := cleanExtension(extension)
	switch ext {
	case ".pdf", ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func cleanExtension(extension string) string {
	if len(extension) > 0 && extension[0] != '.' {
		return "." + extension
	}
	return extension
}

func isSizeValid(size int64) bool {
	return size > 0 && size <= 10*1024*1024
}
