package vector

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

type Config struct {
	Host           string
	Port           int
	CollectionName string
	ApiKey         string
	MinScore       float32
	VectorSize     uint64
	ChunkLimit     uint64
}

type QdrantVectorDB struct {
	Config
	client *qdrant.Client
}

func NewQdrantVectorDB(cfg Config) *QdrantVectorDB {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   cfg.Host,
		Port:   cfg.Port,
		APIKey: cfg.ApiKey,
		UseTLS: true,
	})
	if err != nil {
		panic(fmt.Errorf("failed to connect to Qdrant DB: %w", err))
	}

	q := &QdrantVectorDB{
		client: client,
		Config: cfg,
	}

	if err := q.ensureCollection(context.Background(), cfg.VectorSize); err != nil {
		client.Close()
		fmt.Printf("Warning: failed to ensure collection: %v\n", err)
		panic(err)
	}

	return q
}

func (q *QdrantVectorDB) ensureCollection(ctx context.Context, vectorSize uint64) error {
	exists, err := q.client.CollectionExists(ctx, q.Config.CollectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	// Ensure collection exists (or create if not)
	if !exists {
		err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: q.Config.CollectionName,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     vectorSize,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// Always ensure index for file_id exists
	fieldType := qdrant.FieldType(0)
	_, err = q.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: q.Config.CollectionName,
		FieldName:      "file_id",
		FieldType:      &fieldType,
	})
	if err != nil {
		// Log warning but don't fail, maybe it already exists or concurrent creation
		// In production, we might want to check if error is "already exists" specifically
		fmt.Printf("Warning: ensure file_id index attempt: %v\n", err)
	}

	return nil
}

// Graceful shutdown için
func (q *QdrantVectorDB) Close() error {
	return q.client.Close()
}

func (q *QdrantVectorDB) InsertEmbeddings(ctx context.Context, fileId uuid.UUID, docs []*file.FileMetadata) error {
	if len(docs) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(docs))

	for i, doc := range docs {
		points[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: doc.Id.String(),
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: doc.Vector,
					},
				},
			},
			Payload: map[string]*qdrant.Value{
				"text": {
					Kind: &qdrant.Value_StringValue{
						StringValue: doc.Text,
					},
				},
				"file_id": {
					Kind: &qdrant.Value_StringValue{
						StringValue: fileId.String(),
					},
				},
				"page_numbers": {
					Kind: &qdrant.Value_ListValue{
						ListValue: &qdrant.ListValue{
							Values: convertPageNumbers(doc.PageNumbers),
						},
					},
				},
			},
		}
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.Config.CollectionName,
		Points:         points,
		// Wait:           true,
	})

	return err
}

// Helper function - page numbers nasıl store edilir bunu kontrol et
func convertPageNumbers(pageNumbers []int) []*qdrant.Value {
	values := make([]*qdrant.Value, len(pageNumbers))
	for i, v := range pageNumbers {
		values[i] = &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{
				IntegerValue: int64(v),
			},
		}
	}
	return values
}

func (q *QdrantVectorDB) SearchByEmbedding(
	ctx context.Context,
	fileId uuid.UUID,
	embedding []float32,
) ([]string, error) {
	limit := uint64(q.Config.ChunkLimit)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: q.Config.CollectionName,
		Query:          qdrant.NewQuery(embedding...),
		Limit:          &limit,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		},
		ScoreThreshold: &q.Config.MinScore,
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: "file_id",
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Keyword{
									Keyword: fileId.String(),
								},
							},
						},
					},
				},
			},
		},
	}

	// Qdrant sorgusunu çalıştır
	points, err := q.client.Query(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	if len(points) == 0 {
		return nil, domain.ErrInfoNotFoundInPDF
	}

	var texts []string
	for _, point := range points {
		if point.Payload == nil {
			continue
		}

		val, ok := point.Payload["text"]
		if !ok || val == nil {
			continue
		}

		if textVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			text := textVal.StringValue
			if text != "" {
				texts = append(texts, text)
			}
		}
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("no valid text payloads found in results")
	}

	return texts, nil
}

func (q *QdrantVectorDB) DeleteEmbeddingsByFileId(ctx context.Context, fileId uuid.UUID) error {
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: q.Config.CollectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: &qdrant.Filter{
					Must: []*qdrant.Condition{
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "file_id",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Keyword{
											Keyword: fileId.String(),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
	)
	return err
}

func (q *QdrantVectorDB) DeleteEmbeddingsByFileIDs(ctx context.Context, fileIds []string) error {
	if len(fileIds) == 0 {
		return nil
	}

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "file_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keywords{
								Keywords: &qdrant.RepeatedStrings{
									Strings: fileIds,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: q.CollectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: filter,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete embeddings for file IDs: %w", err)
	}

	return nil
}

func (q *QdrantVectorDB) GetChunks(ctx context.Context, fileId uuid.UUID) ([]string, error) {
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "file_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: fileId.String(),
							},
						},
					},
				},
			},
		},
	}

	limit := int(q.Config.ChunkLimit)
	allChunks := make([]string, 0, limit*2) // Capacity hint
	var nextPageOffset *qdrant.PointId

	// Pagination loop
	for {
		points, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: q.CollectionName,
			Filter:         filter,
			Limit:          pointerUint32(100), // Fetch 100 at a time
			Offset:         nextPageOffset,
			WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scroll points: %w", err)
		}

		if len(points) == 0 {
			break
		}

		for _, point := range points {
			if text, ok := point.Payload["text"]; ok {
				if textVal, ok := text.GetKind().(*qdrant.Value_StringValue); ok {
					allChunks = append(allChunks, textVal.StringValue)
				}
			}
		}

		// Use last point's ID as offset for next page
		nextPageOffset = points[len(points)-1].Id
	}

	// Limit'ten az veya eşitse hepsini döndür
	if len(allChunks) <= limit {
		return allChunks, nil
	}

	// Random seçim yap
	rand.Shuffle(len(allChunks), func(i, j int) {
		allChunks[i], allChunks[j] = allChunks[j], allChunks[i]
	})

	return allChunks[:limit], nil
}

func (q *QdrantVectorDB) GetChunksByPages(ctx context.Context, fileId uuid.UUID, pageRange []int) ([]string, error) {
	if len(pageRange) == 0 {
		return []string{}, nil
	}

	fmt.Println("Pages length:", len(pageRange))
	fmt.Println("Page numbers:", pageRange)

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "file_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: fileId.String(),
							},
						},
					},
				},
			},
		},
	}

	// pageRange'i set'e çevir (O(1) lookup)
	pageSet := make(map[int]bool, len(pageRange))
	for _, page := range pageRange {
		pageSet[page] = true
	}

	matchingChunks := make([]string, 0)
	var nextPageOffset *qdrant.PointId

	// Pagination loop
	for {
		points, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: q.CollectionName,
			Filter:         filter,
			Limit:          pointerUint32(100),
			Offset:         nextPageOffset,
			WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scroll points by pages: %w", err)
		}

		if len(points) == 0 {
			break
		}

		for _, point := range points {
			if pageNumbers, ok := point.Payload["page_numbers"]; ok {
				if listVal, ok := pageNumbers.GetKind().(*qdrant.Value_ListValue); ok {
					hasMatchingPage := false

					for _, pageVal := range listVal.ListValue.Values {
						var pageNum int
						switch v := pageVal.GetKind().(type) {
						case *qdrant.Value_IntegerValue:
							pageNum = int(v.IntegerValue)
						case *qdrant.Value_StringValue:
							if num, err := strconv.Atoi(v.StringValue); err == nil {
								pageNum = num
							}
						}

						if pageSet[pageNum] {
							hasMatchingPage = true
							break
						}
					}

					if hasMatchingPage {
						if text, ok := point.Payload["text"]; ok {
							if textVal, ok := text.GetKind().(*qdrant.Value_StringValue); ok {
								matchingChunks = append(matchingChunks, textVal.StringValue)
							}
						}
					}
				}
			}
		}

		// Use last point's ID as offset for next page
		nextPageOffset = points[len(points)-1].Id
	}

	fmt.Println("Total chunks found:", len(matchingChunks))

	limit := int(q.Config.ChunkLimit)

	if len(matchingChunks) <= limit {
		return matchingChunks, nil
	}

	// Random seçim
	rand.Shuffle(len(matchingChunks), func(i, j int) {
		matchingChunks[i], matchingChunks[j] = matchingChunks[j], matchingChunks[i]
	})

	return matchingChunks[:limit], nil
}

func pointerUint32(v uint32) *uint32 {
	return &v
}
