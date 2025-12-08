package embeddingapi

import (
	"context"
	"fmt"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/message"
	"github.com/sashabaranov/go-openai"
)

type EmbeddingService struct {
	client *openai.Client
}

func NewEmbeddingService(apiKey string) *EmbeddingService {
	client := openai.NewClient(apiKey)

	return &EmbeddingService{
		client: client,
	}
}

func (c *EmbeddingService) GenerateEmbeddingBatch(ctx context.Context, req *file.EmbeddingBatchRequest) (*file.EmbeddingResponse, error) {

	// --- 1) []Text → []string dönüşümü ---
	inputs := make([]string, len(req.Texts))
	for i, t := range req.Texts {
		inputs[i] = t.Text
	}

	embeddingReq := openai.EmbeddingRequestStrings{
		Input: inputs,
		Model: openai.SmallEmbedding3,
	}

	// --- 2) OpenAI çağrısı ---
	resp, err := c.client.CreateEmbeddings(ctx, embeddingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// --- 3) Response mapping ---
	results := make([]file.Embedding, len(resp.Data))

	for i, data := range resp.Data {

		// float64 → float32 dönüştürülmeli
		vector := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vector[j] = float32(v)
		}

		results[i] = file.Embedding{
			Id:          req.Texts[i].Id,
			Text:        req.Texts[i].Text,
			Vector:      vector,
			PageNumbers: []int{}, // eğer page varsa burada doldurursun
		}
	}

	// --- 4) Son response ---
	return &file.EmbeddingResponse{
		Results:   results,
		Dimension: len(results[0].Vector),
		Count:     len(results),
	}, nil
}

func (c *EmbeddingService) GenerateEmbedding(ctx context.Context, req *message.EmbeddingRequest) (*message.EmbeddingResponse, error) {
	embeddingReq := openai.EmbeddingRequest{
		Input: req.Text,
		Model: openai.SmallEmbedding3,
	}

	// Yeni client formatı:
	resp, err := c.client.CreateEmbeddings(ctx, embeddingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	// ---> float64[] → float32[]
	src := resp.Data[0].Embedding
	vector := make([]float32, len(src))
	for i, v := range src {
		vector[i] = float32(v)
	}

	return &message.EmbeddingResponse{
		Vector: vector,
	}, nil
}
