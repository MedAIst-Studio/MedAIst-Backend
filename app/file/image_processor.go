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
	// No processing needed for images at this stage.
	// They are just stored, and OCR happens on-demand during chat.
	return nil
}
