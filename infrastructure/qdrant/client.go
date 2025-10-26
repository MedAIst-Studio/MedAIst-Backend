package vector

import (
	"context"
	"fmt"

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

func (q *QdrantVectorDB) SearchByEmbedding(ctx context.Context, fileId uuid.UUID, embedding []float32) (string, error) {
	searchRequest := &qdrant.SearchPoints{
		CollectionName: q.Config.CollectionName,
		Vector:         embedding,
		Limit:          q.Config.ChunkLimit,
		ScoreThreshold: &q.Config.MinScore,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
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

	searchResult, err := q.client.Search(ctx, searchRequest)
	if err != nil {
		return "", fmt.Errorf("failed to search vectors: %w", err)
	}

	if len(searchResult.Result) == 0 {
		return "", domain.ErrNotFoundInPDF
	}

	payload := searchResult.Result[0].Payload
	if payload == nil {
		return "", fmt.Errorf("no payload found in search result")
	}

	textValue, ok := payload["text"]
	if !ok {
		return "", fmt.Errorf("text field not found in payload")
	}

	text := textValue.GetStringValue()
	if text == "" {
		return "", fmt.Errorf("text content is empty")
	}

	return text, nil
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
