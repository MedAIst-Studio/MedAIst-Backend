package file

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type FileProcessor interface {
	Process(ctx context.Context, file *domain.File, content []byte) error
}
