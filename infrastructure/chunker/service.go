package chunker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/ledongthuc/pdf"
)

type ChunkingService interface {
	ChunkPDF(ctx context.Context, fileBytes []byte, fileID string, options file.ChunkOptions) ([]file.Chunk, error)
}

type PdfChunkingService struct {
	abbreviations map[string]bool
	Config        *file.ChunkOptions
}

func NewChunkingService(config *file.ChunkOptions) *PdfChunkingService {
	abbreviations := map[string]bool{
		"dr":   true,
		"mr":   true,
		"mrs":  true,
		"ms":   true,
		"prof": true,
		"vs":   true,
		"etc":  true,
		"inc":  true,
		"ltd":  true,
		"co":   true,
		"jr":   true,
		"sr":   true,
	}

	// defautlt values
	if config.MaxChunkSize == 0 {
		config.MaxChunkSize = 1000
	}

	if config.OverlapSize == 0 {
		config.OverlapSize = 200
	}

	return &PdfChunkingService{
		abbreviations: abbreviations,
		Config:        config,
	}
}

func (s *PdfChunkingService) ChunkPDF(ctx context.Context, fileBytes []byte, fileID string) ([]file.Chunk, error) {

	reader := bytes.NewReader(fileBytes)
	pdfReader, err := pdf.NewReader(reader, int64(len(fileBytes)))
	if err != nil {
		return nil, fmt.Errorf("PDF okuma hatası: %w", err)
	}

	// Tüm sayfaları oku ve metni çıkar
	var fullText strings.Builder
	pageRanges := make([]pageRange, 0, pdfReader.NumPage()) // Optimize edilmiş sayfa tracking
	currentPos := 0

	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Hata varsa devam et
			continue
		}

		// Metni temizle
		cleanedText := cleanText(text)
		if len(cleanedText) == 0 {
			continue
		}

		// Sayfa range'ini kaydet (O(1))
		startPos := currentPos
		fullText.WriteString(cleanedText)
		fullText.WriteString("\n\n") // Sayfalar arası boşluk
		currentPos = fullText.Len()

		pageRanges = append(pageRanges, pageRange{
			pageNum: pageNum,
			start:   startPos,
			end:     currentPos,
		})
	}

	if fullText.Len() == 0 {
		return nil, errors.New("PDF'den metin çıkarılamadı")
	}

	// Metni chunk'lara ayır
	chunks := s.splitIntoChunks(fullText.String(), pageRanges, fileID)

	return chunks, nil
}

// pageRange O(n) loop yerine range-based lookup için
type pageRange struct {
	pageNum int
	start   int
	end     int
}

// Verilen pozisyon için hangi sayfalarda olduğunu bul (optimize edilmiş)
func getPagesForRange(start, end int, pageRanges []pageRange) []int {
	pages := make(map[int]bool)
	for _, pr := range pageRanges {
		// Range overlap kontrolü
		if !(end <= pr.start || start >= pr.end) {
			pages[pr.pageNum] = true
		}
	}

	result := make([]int, 0, len(pages))
	for page := range pages {
		result = append(result, page)
	}
	return result
}

func (s *PdfChunkingService) splitIntoChunks(text string, pageRanges []pageRange, fileID string) []file.Chunk {
	var chunks []file.Chunk

	// PreserveContext flag'ine göre strateji belirle
	var segments []string
	if s.Config.PreserveContext {
		// Paragraflara ayır (context korumak için)
		segments = splitIntoParagraphs(text)
	} else {
		// Sadece cümlelere ayır (daha agresif chunking)
		segments = s.splitIntoSentences(text)
	}

	var currentChunk strings.Builder
	startPos := 0
	currentStartPos := 0
	chunkIndex := 0

	for _, segment := range segments {
		segmentLen := len(segment)

		// Segment tek başına çok büyükse, zorla böl
		if segmentLen > s.Config.MaxChunkSize {
			// Mevcut chunk'ı kaydet
			if currentChunk.Len() > 0 {
				chunkText := strings.TrimSpace(currentChunk.String())
				chunks = append(chunks, file.Chunk{
					Text: chunkText,
					Metadata: file.ChunkMetadata{
						FileID:      fileID,
						ChunkIndex:  chunkIndex,
						PageNumbers: getPagesForRange(currentStartPos, startPos, pageRanges),
						CharCount:   len(chunkText),
					},
				})
				chunkIndex++
				currentChunk.Reset()
			}

			// Büyük segmenti cümlelere böl
			sentences := s.splitIntoSentences(segment)
			currentStartPos = startPos

			for _, sent := range sentences {
				sentLen := len(sent)

				if currentChunk.Len()+sentLen > s.Config.MaxChunkSize && currentChunk.Len() > 0 {
					// Chunk'ı kaydet
					chunkText := strings.TrimSpace(currentChunk.String())
					chunks = append(chunks, file.Chunk{
						Text: chunkText,
						Metadata: file.ChunkMetadata{
							FileID:      fileID,
							ChunkIndex:  chunkIndex,
							PageNumbers: getPagesForRange(currentStartPos, startPos, pageRanges),
							CharCount:   len(chunkText),
						},
					})
					chunkIndex++

					// Overlap için son kısmı sakla ve trim et
					if s.Config.OverlapSize > 0 {
						overlapText := getLastNChars(chunkText, s.Config.OverlapSize)
						overlapText = strings.TrimSpace(overlapText)
						// Cümle ortasından başlamayı önle
						overlapText = trimToSentenceStart(overlapText)
						currentChunk.Reset()
						currentChunk.WriteString(overlapText)
						currentChunk.WriteString(" ")
					} else {
						currentChunk.Reset()
					}

					currentStartPos = startPos
				}

				currentChunk.WriteString(sent)
				currentChunk.WriteString(" ")
				startPos += sentLen
			}
			continue
		}

		// Normal durum: segment chunk'a sığıyorsa ekle
		if currentChunk.Len()+segmentLen > s.Config.MaxChunkSize && currentChunk.Len() > 0 {
			// Mevcut chunk'ı kaydet
			chunkText := strings.TrimSpace(currentChunk.String())
			chunks = append(chunks, file.Chunk{
				Text: chunkText,
				Metadata: file.ChunkMetadata{
					FileID:      fileID,
					ChunkIndex:  chunkIndex,
					PageNumbers: getPagesForRange(currentStartPos, startPos, pageRanges),
					CharCount:   len(chunkText),
				},
			})
			chunkIndex++

			// Overlap için son kısmı sakla ve trim et
			if s.Config.OverlapSize > 0 {
				overlapText := getLastNChars(chunkText, s.Config.OverlapSize)
				overlapText = strings.TrimSpace(overlapText)
				overlapText = trimToSentenceStart(overlapText)
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentChunk.WriteString("\n\n")
			} else {
				currentChunk.Reset()
			}

			currentStartPos = startPos
		}

		currentChunk.WriteString(segment)
		if s.Config.PreserveContext {
			currentChunk.WriteString("\n\n")
		} else {
			currentChunk.WriteString(" ")
		}
		startPos += segmentLen
	}

	if currentChunk.Len() > 0 {
		chunkText := strings.TrimSpace(currentChunk.String())
		chunks = append(chunks, file.Chunk{
			Text: chunkText,
			Metadata: file.ChunkMetadata{
				FileID:      fileID,
				ChunkIndex:  chunkIndex,
				PageNumbers: getPagesForRange(currentStartPos, startPos, pageRanges),
				CharCount:   len(chunkText),
			},
		})
	}

	return chunks
}

func cleanText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func splitIntoParagraphs(text string) []string {
	paragraphs := strings.Split(text, "\n\n")
	var result []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			result = append(result, p)
		}
	}
	return result
}

var abbreviationPattern = regexp.MustCompile(`\b[A-Z][a-z]{0,3}\.$`)

func (s *PdfChunkingService) splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		// Cümle sonu işaretleri
		if (r == '.' || r == '!' || r == '?') && i < len(runes)-1 {
			// Kısaltma kontrolü
			currentText := current.String()
			if r == '.' {
				// Son kelimeyi al
				words := strings.Fields(currentText)
				if len(words) > 0 {
					lastWord := strings.ToLower(strings.TrimSuffix(words[len(words)-1], "."))
					// Bilinen kısaltma mı?
					if s.abbreviations[lastWord] {
						continue
					}
					// Regex ile kısaltma pattern'i kontrolü
					if abbreviationPattern.MatchString(words[len(words)-1]) {
						continue
					}
				}
			}

			// Sonraki karakter boşluk veya büyük harfse cümle sonu
			next := runes[i+1]
			if unicode.IsSpace(next) || unicode.IsUpper(next) {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	// Son cümleyi ekle
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

func getLastNChars(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[len(runes)-n:])
}

func trimToSentenceStart(text string) string {
	for i, r := range text {
		if r == '.' || r == '!' || r == '?' {
			// Sonraki karakter boşluksa, oradan başla
			if i+1 < len(text) && unicode.IsSpace(rune(text[i+1])) {
				return strings.TrimSpace(text[i+1:])
			}
		}
	}
	return text
}
