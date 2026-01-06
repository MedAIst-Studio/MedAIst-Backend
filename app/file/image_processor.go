package file

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

func (p *ImageProcessor) Process(ctx context.Context, file *domain.File, content []byte) error {
	return nil
}
