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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Addr           string
	CollectionName string
	MinScore       float32
	VectorSize     uint64
	ChunkLimit     uint64
}

type QdrantVectorDB struct {
	Config
	client            qdrant.PointsClient
	collectionsClient qdrant.CollectionsClient
	conn              *grpc.ClientConn
}

func NewQdrantVectorDB(cfg Config) *QdrantVectorDB {
	conn, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("failed to connect to Qdrant: %w", err))
	}

	q := &QdrantVectorDB{
		client:            qdrant.NewPointsClient(conn),
		collectionsClient: qdrant.NewCollectionsClient(conn),
		conn:              conn,
		Config:            cfg,
	}

	if err := q.ensureCollection(context.Background(), cfg.VectorSize); err != nil {
		conn.Close()
		panic(err)
	}

	return q
}

func (q *QdrantVectorDB) ensureCollection(ctx context.Context, vectorSize uint64) error {
	_, err := q.collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: q.Config.CollectionName,
	})

	if err == nil {
		return nil
	}

	_, err = q.collectionsClient.Create(ctx, &qdrant.CreateCollection{
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

	return nil
}

// Graceful shutdown için
func (q *QdrantVectorDB) Close() error {
	return q.conn.Close()
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
	searchRequest := &qdrant.SearchPoints{
		CollectionName: q.Config.CollectionName,
		Vector:         embedding,
		Limit:          q.Config.ChunkLimit,
		ScoreThreshold: &q.Config.MinScore,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		},
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
	searchResult, err := q.client.Search(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	if len(searchResult.Result) == 0 {
		return nil, domain.ErrInfoNotFoundInPDF
	}

	var texts []string
	for _, point := range searchResult.Result {
		if point.Payload == nil {
			continue
		}

		val, ok := point.Payload["text"]
		if !ok || val == nil {
			continue
		}

		text := val.GetStringValue()
		if text == "" {
			continue
		}

		texts = append(texts, text)
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

	scrollResp, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: q.CollectionName,
		Filter:         filter,
		Limit:          pointerUint32(1000),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scroll points: %w", err)
	}

	allChunks := make([]string, 0)
	for _, point := range scrollResp.Result {
		if text, ok := point.Payload["text"]; ok {
			if textVal, ok := text.GetKind().(*qdrant.Value_StringValue); ok {
				allChunks = append(allChunks, textVal.StringValue)
			}
		}
	}

	if len(allChunks) <= 5 {
		return allChunks, nil
	}

	rand.Shuffle(len(allChunks), func(i, j int) {
		allChunks[i], allChunks[j] = allChunks[j], allChunks[i]
	})

	return allChunks[:5], nil
}

func (q *QdrantVectorDB) GetChunksByPages(ctx context.Context, fileId uuid.UUID, pageRange []int) ([]string, error) {
	if len(pageRange) == 0 {
		return []string{}, nil
	}

	fmt.Println("Pages length:", len(pageRange))
	fmt.Println("Page numbers:", pageRange)

	// file_id ile filtrele
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

	// Tüm file'a ait chunkları al
	scrollResp, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: q.CollectionName,
		Filter:         filter,
		Limit:          pointerUint32(1000),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scroll points by pages: %w", err)
	}

	// pageRange'i set'e çevir (hızlı lookup için)
	pageSet := make(map[int]bool)
	for _, page := range pageRange {
		pageSet[page] = true
	}

	// Sadece istenen sayfalardaki chunkları filtrele
	matchingChunks := make([]string, 0)
	for _, point := range scrollResp.Result {
		// page_numbers listesini kontrol et
		if pageNumbers, ok := point.Payload["page_numbers"]; ok {
			if listVal, ok := pageNumbers.GetKind().(*qdrant.Value_ListValue); ok {
				// Bu chunk'ın hangi sayfalarda olduğunu kontrol et
				hasMatchingPage := false
				for _, pageVal := range listVal.ListValue.Values {
					// Sayfa numarasını al (integer veya string olabilir)
					var pageNum int
					switch v := pageVal.GetKind().(type) {
					case *qdrant.Value_IntegerValue:
						pageNum = int(v.IntegerValue)
					case *qdrant.Value_StringValue:
						// String ise int'e çevir
						if num, err := strconv.Atoi(v.StringValue); err == nil {
							pageNum = num
						}
					}

					// Eğer bu sayfa istenen sayfalardan biriyse
					if pageSet[pageNum] {
						hasMatchingPage = true
						break
					}
				}

				// Eğer chunk istenen sayfalardan birindeyse text'i al
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

	fmt.Println("Total chunks found:", len(matchingChunks))

	// Eğer 5'ten az varsa hepsini döndür
	if len(matchingChunks) <= 5 {
		return matchingChunks, nil
	}

	// Random 5 chunk seç
	rand.Shuffle(len(matchingChunks), func(i, j int) {
		matchingChunks[i], matchingChunks[j] = matchingChunks[j], matchingChunks[i]
	})

	return matchingChunks[:5], nil
}

// Helper functions
func pointerUint32(v uint32) *uint32 {
	return &v
}

func pointerUint64(v uint64) *uint64 {
	return &v
}
