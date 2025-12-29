package file

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type PdfProcessor struct {
	Repo      Repository
	Fl        FileService
	Vb        VectorDB
	Ems       EmbeddingService
	Chunker   Chunker
	BatchSize int
	Normalize bool
}

func NewPdfProcessor(repo Repository, fl FileService, vb VectorDB, ems EmbeddingService, chunker Chunker, batchSize int, normalize bool) *PdfProcessor {
	return &PdfProcessor{
		Repo:      repo,
		Fl:        fl,
		Vb:        vb,
		Ems:       ems,
		Chunker:   chunker,
		BatchSize: batchSize,
		Normalize: normalize,
	}
}

func (p *PdfProcessor) Process(ctx context.Context, f *domain.File, content []byte) error {
	chunks, err := p.Chunker.ChunkPDF(ctx, content, f.Id.String())
	if err != nil {
		p.recover(ctx, f.Id)
		if !errors.Is(err, domain.ErrInternalServer) {
			return err
		}
		return err
	}

	for start := 0; start < len(chunks); start += p.BatchSize {
		end := int(math.Min(float64(start+p.BatchSize), float64(len(chunks))))

		batch := chunks[start:end]

		var embeddings []Embedding
		for _, chunk := range batch {
			var embedding Embedding
			embedding.Id = uuid.New()
			embedding.PageNumbers = chunk.Metadata.PageNumbers
			embedding.Text = chunk.Text
			embeddings = append(embeddings, embedding)
		}

		var texts []Text
		for _, emb := range embeddings {
			texts = append(texts, Text{
				Id:   emb.Id,
				Text: emb.Text,
			})
		}

		embeddingReq := &EmbeddingBatchRequest{
			Texts:     texts,
			Normalize: p.Normalize,
			BatchSize: p.BatchSize,
		}

		embeddingRes, err := p.Ems.GenerateEmbeddingBatch(ctx, embeddingReq)
		if err != nil {
			p.recover(ctx, f.Id)
			return err
		}

		embeddingMap := make(map[uuid.UUID]*Embedding)
		for i := range embeddings {
			embeddingMap[embeddings[i].Id] = &embeddings[i]
		}

		for _, resultEmb := range embeddingRes.Results {
			if emb, exists := embeddingMap[resultEmb.Id]; exists {
				emb.Vector = resultEmb.Vector
			} else {
				p.recover(ctx, f.Id)
				return fmt.Errorf("embedding ID %s not found in batch", resultEmb.Id)
			}
		}

		for _, emb := range embeddings {
			if len(emb.Vector) == 0 {
				p.recover(ctx, f.Id)
				return fmt.Errorf("embedding %s has no vector", emb.Id)
			}
		}

		var fileMetadatas []*FileMetadata
		for _, emb := range embeddings {
			fm := &FileMetadata{
				Id:          uuid.New(),
				Text:        emb.Text,
				PageNumbers: emb.PageNumbers,
				Vector:      emb.Vector,
			}
			fileMetadatas = append(fileMetadatas, fm)
		}

		if err := p.Vb.InsertEmbeddings(ctx, f.Id, fileMetadatas); err != nil {
			p.recover(ctx, f.Id)
			return err
		}

	}

	return nil
}

func (p *PdfProcessor) recover(ctx context.Context, fileId uuid.UUID) {
	p.Repo.DeleteFileById(ctx, fileId)
	p.Fl.DeleteFile(ctx, formatFilePath(fileId))
	p.Vb.DeleteEmbeddingsByFileId(ctx, fileId)
}
