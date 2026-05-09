package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

// Consts
const (
	openRouterBaseURL = "https://openrouter.ai/api/v1"

	queryExpansionSystemPrompt = `Sen bir arama sorgusu genişletme uzmanısın. Görevin, kullanıcının asıl niyetini koruyarak farklı açılardan 5 adet Türkçe arama sorgusu üretmektir.

	Kurallar:
	1. Asıl niyeti koru - kullanıcının ne sorduğunu değiştirme.
	2. Farklı ifadeler, eş anlamlılar ve bakış açıları kullan.
	3. Tıp fakültesi (TUS/DUS) müfredatına uygun tıbbi terimler kullanmaya çalış.
	4. Asıl sorguda olmayan tamamen yeni bilgi ekleme.

	SADECE şu formatta geçerli bir JSON döndür:
	{"queries": ["...", "...", "...", "...", "..."]}
`
)

// ----------------------------------------------------------------------
// STRUCTS (Native implementation to replace go-openai)
// ----------------------------------------------------------------------

type ChatMessagePartType string

const (
	ChatMessagePartTypeText     ChatMessagePartType = "text"
	ChatMessagePartTypeImageURL ChatMessagePartType = "image_url"
)

type ChatMessageImageURL struct {
	URL string `json:"url,omitempty"`
}

type ChatMessagePart struct {
	Type     ChatMessagePartType  `json:"type,omitempty"`
	Text     string               `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL `json:"image_url,omitempty"`
}

type ChatCompletionMessage struct {
	Role         string            `json:"role"`
	Content      interface{}       `json:"content"` // string or []ChatMessagePart
	MultiContent []ChatMessagePart `json:"-"`       // Helper to build content (not used in JSON marshaling directly, handled by logic)
}

type ResponseFormat struct {
	Type       string                `json:"type,omitempty"` // "json_object" or "json_schema"
	JSONSchema *ResponseFormatSchema `json:"json_schema,omitempty"`
}

type ResponseFormatSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

type ChatCompletionRequest struct {
	Model          string                  `json:"model"`
	Messages       []ChatCompletionMessage `json:"messages"`
	Temperature    float32                 `json:"temperature,omitempty"`
	Stream         bool                    `json:"stream,omitempty"`
	ResponseFormat *ResponseFormat         `json:"response_format,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

type EmbeddingRequest struct {
	Input interface{} `json:"input"` // string or []string
	Model string      `json:"model"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"` // OpenRouter usually returns float64
	Index     int       `json:"index"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  interface{}     `json:"usage"`
}

// ----------------------------------------------------------------------
// DYNAMIC PROMPT SYSTEM - Subject-specific quiz generation
// ----------------------------------------------------------------------

type SubjectPromptConfig struct {
	SubjectName       string
	SystemPromptExtra string // Subject-specific additions to the base system prompt
	DistractorRules   string // Subject-specific distractor crafting rules
	FewShotExample    string // One high-quality TUS/Komite example per subject
}

var subjectPrompts = map[string]SubjectPromptConfig{
	"anatomi": {
		SubjectName:       "Anatomi",
		SystemPromptExtra: `Bu bir ANATOMİ sorusudur. İnsan vücudunun yapıları, organlar, kemikler, kaslar, sinirler ve damarlar gibi anatomi alt dallarına hakimsin.`,
		DistractorRules: `- Çeldiricilerde komşu veya benzer işlevli anatomik yapıları kullan
- Sinir inervasyonları veya kanlanma varyasyonlarını çeldirici olarak ekle
- Yön ve yerleşim terimlerini (anterior/posterior, medial/lateral) karıştır
- Benzer isimli anatomik oluşumları kullan`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Aşağıdaki kaslardan hangisi N. musculocutaneus tarafından inerve edilir?",
  "difficulty": "orta",
  "topic": "Üst Ekstremite Kasları",
  "options": [
    {"no": 1, "label": "A", "text": "M. biceps brachii", "isCorrect": true},
    {"no": 2, "label": "B", "text": "M. triceps brachii", "isCorrect": false},
    {"no": 3, "label": "C", "text": "M. deltoideus", "isCorrect": false},
    {"no": 4, "label": "D", "text": "M. brachioradialis", "isCorrect": false},
    {"no": 5, "label": "E", "text": "M. pronator teres", "isCorrect": false}
  ],
  "explanation": "M. biceps brachii, m. coracobrachialis ve m. brachialis'in büyük bir kısmı N. musculocutaneus tarafından inerve edilir. Triceps n. radialis, deltoid n. axillaris, brachioradialis n. radialis, pronator teres ise n. medianus tarafından uyarılır."
}`,
	},
	"fizyoloji": {
		SubjectName:       "Fizyoloji",
		SystemPromptExtra: `Bu bir FİZYOLOJİ sorusudur. Hücre, sinir, kas, kan, dolaşım, solunum, boşaltım, sindirim ve endokrin sistem fizyolojisine hakimsin.`,
		DistractorRules: `- Çeldiricilerde zıt etki gösteren hormonları/mekanizmaları kullan
- Fizyolojik süreçlerin adımlarını karıştır
- Reseptör tiplerini ve etkilerini birbirine karıştır
- İyon kanallarının açılma/kapanma durumlarını ve yönlerini (içe/dışa) yanlış eşleştir`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Aksiyon potansiyelinin depolarizasyon evresinden temel olarak hangi iyonun hücre içine girmesi sorumludur?",
  "difficulty": "kolay",
  "topic": "Hücre Fizyolojisi",
  "options": [
    {"no": 1, "label": "A", "text": "Sodyum (Na+)", "isCorrect": true},
    {"no": 2, "label": "B", "text": "Potasyum (K+)", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Kalsiyum (Ca2+)", "isCorrect": false},
    {"no": 4, "label": "D", "text": "Klorür (Cl-)", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Magnezyum (Mg2+)", "isCorrect": false}
  ],
  "explanation": "Aksiyon potansiyelinin depolarizasyon evresinde voltaj kapılı sodyum kanalları açılır ve hücre içine yoğun miktarda Na+ girer. Repolarizasyondan ise K+ çıkışı sorumludur."
}`,
	},
	"biyokimya": {
		SubjectName:       "Biyokimya",
		SystemPromptExtra: `Bu bir BİYOKİMYA sorusudur. Karbonhidrat, lipit, protein, nükleik asit metabolizması, enzimler ve moleküler biyoloji konularına hakimsin.`,
		DistractorRules: `- Çeldiricilerde aynı yolaktaki diğer enzimleri/metabolitleri kullan
- Hız kısıtlayıcı adımları ve regülatörlerini karıştır
- Benzer isimli enzim/koenzimleri çeldirici yap
- Defekt durumunda biriken/eksilen maddeleri tersine çevir`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Glikoliz yolağında ATP üreten ilk basamağı katalize eden enzim aşağıdakilerden hangisidir?",
  "difficulty": "zor",
  "topic": "Karbonhidrat Metabolizması",
  "options": [
    {"no": 1, "label": "A", "text": "Fosfogliserat kinaz", "isCorrect": true},
    {"no": 2, "label": "B", "text": "Hekzokinaz", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Fosfofruktokinaz-1", "isCorrect": false},
    {"no": 4, "label": "D", "text": "Pirüvat kinaz", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Gliseraldehit 3-fosfat dehidrogenaz", "isCorrect": false}
  ],
  "explanation": "Glikolizde ilk ATP üretimi, 1,3-bisfosfogliseratın 3-fosfogliserata dönüştüğü basamakta fosfogliserat kinaz tarafından gerçekleştirilir. Hekzokinaz ve PFK-1 ATP tüketen enzimlerdir."
}`,
	},
	"patoloji": {
		SubjectName:       "Patoloji",
		SystemPromptExtra: `Bu bir PATOLOJİ sorusudur. Hücresel hasar, inflamasyon, neoplazi, hemodinamik bozukluklar ve sistemik patoloji konularına hakimsin.`,
		DistractorRules: `- Çeldiricilerde benzer mikroskobik/makroskobik bulguları olan farklı hastalıkları kullan
- Kanserlerde farklı doku tiplerinin öncü lezyonlarını karıştır
- Benzer isimli sendromları veya bulguları çeldirici olarak ver
- Spesifik boyamaları veya genetik mutasyonları karıştır`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Akut inflamasyonda dokuya ilk göç eden ve en erken dönemde baskın olan inflamatuar hücre aşağıdakilerden hangisidir?",
  "difficulty": "kolay",
  "topic": "İnflamasyon",
  "options": [
    {"no": 1, "label": "A", "text": "Nötrofil", "isCorrect": true},
    {"no": 2, "label": "B", "text": "Makrofaj", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Lenfosit", "isCorrect": false},
    {"no": 4, "label": "D", "text": "Eozinofil", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Plazma hücresi", "isCorrect": false}
  ],
  "explanation": "Akut inflamasyonun ilk 6-24 saatlik döneminde dokuya ilk ve en yoğun göç eden hücre nötrofildir. Monositler (makrofajlar) genellikle 24-48 saat sonra alana gelir."
}`,
	},
	"farmakoloji": {
		SubjectName:       "Farmakoloji",
		SystemPromptExtra: `Bu bir FARMAKOLOJİ sorusudur. İlaçların farmakokinetiği, farmakodinamiği, otonom, santral, kardiyovasküler sistem farmakolojisi ve kemoterapötiklere hakimsin.`,
		DistractorRules: `- Çeldiricilerde aynı gruptan diğer ilaçları kullan ama etkilerini değiştir
- İlaç yan etkilerini/toksisitelerini birbirine karıştır
- Antidot eşleştirmelerini yanlış yap
- Farklı reseptör tipine etki eden ilaçları seçeneklere ekle`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Aşağıdaki ilaçlardan hangisi opioid intoksikasyonu tedavisinde kullanılan spesifik bir antidottur?",
  "difficulty": "orta",
  "topic": "Santral Sinir Sistemi Farmakolojisi",
  "options": [
    {"no": 1, "label": "A", "text": "Naloksan", "isCorrect": true},
    {"no": 2, "label": "B", "text": "Flumazenil", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Fomepizol", "isCorrect": false},
    {"no": 4, "label": "D", "text": "N-asetilsistein", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Protamin sülfat", "isCorrect": false}
  ],
  "explanation": "Naloksan, opioid reseptörlerinin kompetitif antagonistidir ve zehirlenmelerinde kullanılır. Flumazenil benzodiazepin, N-asetilsistein parasetamol, protamin sülfat heparin zehirlenmesinde kullanılır."
}`,
	},
	"mikrobiyoloji": {
		SubjectName:       "Mikrobiyoloji",
		SystemPromptExtra: `Bu bir MİKROBİYOLOJİ sorusudur. Genel ve özel bakteriyoloji, viroloji, mikoloji, parazitoloji ve immünoloji konularına hakimsin.`,
		DistractorRules: `- Çeldiricilerde aynı familyadan diğer etkenleri kullan
- Bakterilerin gram boyanma/morfoloji özelliklerini karıştır
- Virüslerin nükleik asit tiplerini (DNA/RNA, zarflı/zarfsız) birbirine karıştır
- Spesifik vektörleri ve ara konakları yanlış eşleştir`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Gram negatif, oksidaz pozitif, virgüle benzer eğri çomak şeklinde olan ve pirinç suyu görünümünde ishale yol açan etken hangisidir?",
  "difficulty": "orta",
  "topic": "Özel Bakteriyoloji",
  "options": [
    {"no": 1, "label": "A", "text": "Vibrio cholerae", "isCorrect": true},
    {"no": 2, "label": "B", "text": "Escherichia coli", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Salmonella typhi", "isCorrect": false},
    {"no": 4, "label": "D", "text": "Shigella dysenteriae", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Campylobacter jejuni", "isCorrect": false}
  ],
  "explanation": "Vibrio cholerae gram negatif, virgül şeklinde (curved rod), oksidaz pozitif bir bakteridir ve tipik sulu ishal tablosu oluşturur. Diğerleri farklı morfoloji veya klinik tablolara neden olur."
}`,
	},
	"klinik": {
		SubjectName:       "Klinik Bilimler",
		SystemPromptExtra: `Bu bir KLİNİK sorusudur. Dahiliye, pediatri, genel cerrahi, kadın hastalıkları ve doğum, psikiyatri gibi klinik tıp bilimlerine hakimsin.`,
		DistractorRules: `- Çeldiricilerde benzer semptomları olan ayırıcı tanıdaki diğer hastalıkları kullan
- En olası tanı/ilk istenecek tetkik/altın standart tanı yöntemlerini karıştır
- Birinci basamak tedavilerle kurtarma (salvage) tedavilerini birbirinin yerine koy
- Hastalıkların komplikasyonlarını birbiriyle değiştir`,
		FewShotExample: `ÖRNEK SORU:
{
  "questionText": "Kırk beş yaşında obez kadın hasta, özellikle yağlı yemeklerden sonra sağ üst kadranda beliren ve sağ skapulaya yayılan ağrı şikayetiyle başvuruyor. Fizik muayenede Murphy belirtisi pozitiftir. Bu hastadaki en olası tanı nedir?",
  "difficulty": "orta",
  "topic": "Genel Cerrahi / Gastrointestinal Sistem",
  "options": [
    {"no": 1, "label": "A", "text": "Akut kolesistit", "isCorrect": true},
    {"no": 2, "label": "B", "text": "Akut pankreatit", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Peptik ülser perforasyonu", "isCorrect": false},
    {"no": 4, "label": "D", "text": "Akut apandisit", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Nefrolitiazis", "isCorrect": false}
  ],
  "explanation": "Orta yaş, kadın, obez, yağlı yemek sonrası sağ üst kadran ağrısı ve pozitif Murphy belirtisi klasik olarak akut kolesistiti işaret eder. Pankreatit daha çok sırta kuşak tarzı yayılır."
}`,
	},
	"genel": {
		SubjectName:       "Tıp Genel",
		SystemPromptExtra: `Konuyu tıbbi metnin içeriğinden belirle ve tıp öğrencilerine uygun seviyede bir soru üret.`,
		DistractorRules: `- Çeldiriciler sorunun tıbbi konusuyla aynı alandan olmalıdır
- Her çeldirici mantıklı ama yanlış olmalıdır
- Çeldiriciler birbirine yakın klinik/temel kavramlardan seçilmelidir`,
		FewShotExample: "",
	},
}

// subjectKeywords maps keywords to subject identifiers for zero-cost classification
var subjectKeywords = map[string][]string{
	"anatomi":       {"kemik", "kas", "sinir", "damar", "arter", "ven", "eklem", "ligaman", "fasya", "kafatası", "vertebra", "toraks", "abdomen", "pelvis", "ekstremite", "pleksus", "foramen", "fossa", "anatomi", "kadavra", "innervasyon", "origo", "insertio"},
	"fizyoloji":     {"hücre", "aksiyon potansiyeli", "sinaps", "reseptör", "hormon", "kalp debisi", "kan basıncı", "solunum", "filtrasyon", "sekresyon", "absorbsiyon", "metabolizma", "termoregülasyon", "kan", "koagülasyon", "fizyoloji", "işlev", "potansiyel"},
	"biyokimya":     {"enzim", "protein", "karbonhidrat", "lipit", "aminoasit", "vitamin", "mineral", "metabolizma", "glikoliz", "krebs", "oksidatif fosforilasyon", "dna", "rna", "replikasyon", "transkripsiyon", "translasyon", "biyokimya", "molekül", "kolesterol"},
	"patoloji":      {"inflamasyon", "nekroz", "apoptoz", "neoplazi", "tümör", "kanser", "karsinom", "sarkom", "granülom", "hiperplazi", "hipertrofi", "atrofi", "metaplazi", "displazi", "patoloji", "biyopsi", "makroskopi", "mikroskopi", "kist"},
	"farmakoloji":   {"ilaç", "reseptör", "agonist", "antagonist", "farmakokinetik", "farmakodinamik", "yan etki", "toksisite", "antibiyotik", "analjezik", "anestezik", "antihipertansif", "antidiyabetik", "farmakoloji", "doz", "endikasyon", "kontrendikasyon", "antidot"},
	"mikrobiyoloji": {"bakteri", "virüs", "mantar", "parazit", "enfeksiyon", "antibiyotik direnci", "kültür", "pcr", "antijen", "antikor", "bağışıklık", "aşı", "flora", "patojen", "mikrobiyoloji", "mikroskop", "boyama", "agar"},
	"klinik":        {"hipertansiyon", "diyabet", "astım", "yetmezlik", "siroz", "anemi", "lösemi", "tanı", "tedavi", "semptom", "bulgu", "sendrom", "muayene", "akut", "kronik", "komplikasyon", "anamnez", "klinik", "cerrahi", "hasta", "ağrı"},
}

func classifySubject(chunk string) string {
	lowerChunk := strings.ToLower(chunk)

	scores := make(map[string]int)
	for subject, keywords := range subjectKeywords {
		for _, keyword := range keywords {
			count := strings.Count(lowerChunk, keyword)
			if count > 0 {
				scores[subject] += count
			}
		}
	}

	bestSubject := "genel"
	bestScore := 0
	for subject, score := range scores {
		if score > bestScore {
			bestScore = score
			bestSubject = subject
		}
	}

	// Minimum eşik: en az 3 keyword match olmalı
	if bestScore < 3 {
		return "genel"
	}

	return bestSubject
}

func buildSubjectSystemPrompt(subject string) string {
	config, exists := subjectPrompts[subject]
	if !exists {
		config = subjectPrompts["genel"]
	}

	basePrompt := fmt.Sprintf(`Sen 20 yıllık deneyime sahip, kıdemli bir TUS (Tıpta Uzmanlık Eğitimi Giriş Sınavı) ve tıp fakültesi komite soru hazırlayıcısısın.
Uzmanlık alanın: %s

%s

TEMEL GÖREVİN: Verilen metne dayalı, TUS standardında veya tıp fakültesi zorluğunda yüksek kaliteli çoktan seçmeli sorular üretmek.

KRİTİK KURALLAR:
1. SORU KÖKÜ: Net, anlaşılır ve tek başına okunduğunda ne sorulduğu anlaşılacak şekilde yaz. Vakaya dayalı (klinik) veya temel bilgi sorgulayan türde olabilir. (20-60 kelime).
2. TUTARLILIK: Her şık (A-E), soru köküyle dilbilgisel ve anlamsal olarak uyumlu olmalıdır. Soru kökünü oku, ardından her şıkkı soru köküne ekleyerek oku — cümle doğal ve anlamlı olmalı.
3. KONU BİRLİĞİ: Tüm şıklar aynı konu alanından olmalıdır. Şıklar birbiriyle karşılaştırılabilir nitelikte olmalıdır.
4. ÇELDİRİCİ KALİTESİ:
   - Yanlış şıklar "neredeyse doğru" olmalı — tamamen alakasız veya saçma OLMAMALI.
   - İyi bir çeldirici, konuyu yüzeysel bilen bir tıp öğrencisini yanıltabilmeli.
   - Tüm şıklar benzer uzunlukta ve aynı dil yapısında olmalıdır.
   - Bir öğrenci şıklara baktığında hepsinin makul bir tıbbi tanı/tedavi/mekanizma olabileceğini düşünmeli.
%s

❌ KÖTÜ ÇELDİRİCİ ÖRNEKLERİ (BUNLARI YAPMA):
- Soruyla tamamen alakasız şıklar (ör: dahiliye sorusunda alakasız anatomik terim)
- Açıkça saçma veya tıp dışı seçenekler
- Aynı şıkkın farklı kelimelerle tekrarlanması
- Şıklar arasında belirgin uzunluk farkı (doğru cevap çok uzun, diğerleri kısa)

5. ZORLUK: Sorunun zorluk seviyesini belirle (kolay/orta/zor). Orta-zor TUS seviyesini hedefle.
6. KONU: Sorunun hangi tıbbi konuyla ilgili olduğunu kısaca belirt.
7. Sadece verilen JSON şemasına uygun çıktı üret.
8. Türkçe yaz.`, config.SubjectName, config.SystemPromptExtra, config.DistractorRules)

	if config.FewShotExample != "" {
		basePrompt += "\n\n" + config.FewShotExample
	}

	return basePrompt
}

var (
	quizQuestionSchema = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"questionText": map[string]interface{}{
				"type":        "string",
				"description": "Soru kökü metni (20-40 kelime)",
			},
			"difficulty": map[string]interface{}{
				"type":        "string",
				"description": "Sorunun zorluk seviyesi",
				"enum":        []string{"kolay", "orta", "zor"},
			},
			"topic": map[string]interface{}{
				"type":        "string",
				"description": "Sorunun ilgili olduğu konu (ör: Hücre Biyolojisi, Newton Hareket Yasaları)",
			},
			"explanation": map[string]interface{}{
				"type":        "string",
				"description": "Doğru cevabın açıklaması ve diğer şıkların neden yanlış olduğu",
			},
			"options": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"no": map[string]interface{}{
							"type":        "integer",
							"description": "Şık numarası (1-5)",
							"minimum":     1,
							"maximum":     5,
						},
						"label": map[string]interface{}{
							"type":        "string",
							"description": "Şık etiketi (A-E)",
							"enum":        []string{"A", "B", "C", "D", "E"},
						},
						"text": map[string]interface{}{
							"type":        "string",
							"description": "Şık metni",
						},
						"isCorrect": map[string]interface{}{
							"type":        "boolean",
							"description": "Bu şıkkın doğru olup olmadığı",
						},
					},
					"required":             []string{"no", "label", "text", "isCorrect"},
					"additionalProperties": false,
				},
				"description": "Tam olarak 5 seçenek listesi",
				"minItems":    5,
				"maxItems":    5,
			},
		},
		"required":             []string{"questionText", "difficulty", "topic", "explanation", "options"},
		"additionalProperties": false,
	}
)

type Config struct {
	ApiKey                 string
	ChatModel              string
	GenerateQuestionModel  string
	ExpandQuestionModel    string
	GenerateChatTitleModel string
	ReasoningModel         string
	VisionModel            string
	EmbeddingModel         string
}

type OpenRouterService struct {
	httpClient *http.Client
	Config
}

func NewOpenRouterService(config Config) *OpenRouterService {
	return &OpenRouterService{
		httpClient: &http.Client{},
		Config:     config,
	}
}

// ----------------------------------------------------------------------
// HELPER METHODS
// ----------------------------------------------------------------------

func (s *OpenRouterService) makeRequest(ctx context.Context, method, endpoint string, body interface{}, respDest interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", openRouterBaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ApiKey)
	// OpenRouter specific headers (optional but good practice)
	req.Header.Set("HTTP-Referer", "https://medaiist.com") // Replace with actual site if strictly needed
	req.Header.Set("X-Title", "MedAIist Backend")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if respDest != nil {
		if err := json.NewDecoder(resp.Body).Decode(respDest); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// ----------------------------------------------------------------------
// CHAT
// ----------------------------------------------------------------------

func (s *OpenRouterService) SendChatRequest(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	apiMessages := make([]ChatCompletionMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if len(msg.ImageUrl) > 0 {
			parts := []ChatMessagePart{
				{Type: ChatMessagePartTypeText, Text: msg.Content},
			}
			for _, url := range msg.ImageUrl {
				parts = append(parts, ChatMessagePart{
					Type:     ChatMessagePartTypeImageURL,
					ImageURL: &ChatMessageImageURL{URL: url},
				})
			}
			apiMessages = append(apiMessages, ChatCompletionMessage{
				Role:    string(msg.Role),
				Content: parts,
			})
		} else {
			apiMessages = append(apiMessages, ChatCompletionMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	apiReq := ChatCompletionRequest{
		Model:    s.ChatModel,
		Messages: apiMessages,
		Stream:   req.Stream,
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	content := ""
	if str, ok := apiResp.Choices[0].Message.Content.(string); ok {
		content = str
	} else if parts, ok := apiResp.Choices[0].Message.Content.([]interface{}); ok {
		for _, p := range parts {
			if m, ok := p.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					content += t
				}
			}
		}
	}

	return &domain.ChatResponse{
		Text: content,
	}, nil
}

func (s *OpenRouterService) GenerateChatTitle(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	apiMessages := make([]ChatCompletionMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		apiMessages = append(apiMessages, ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	apiReq := ChatCompletionRequest{
		Model:    s.GenerateChatTitleModel,
		Messages: apiMessages,
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to generate title: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	return &domain.ChatResponse{
		Text: content,
	}, nil
}

// ----------------------------------------------------------------------
// CLASSIFICATION & EXPANSION
// ----------------------------------------------------------------------

func (s *OpenRouterService) ExpandQuery(ctx context.Context, content string) ([]string, error) {
	apiReq := ChatCompletionRequest{
		Model: s.ExpandQuestionModel,
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: queryExpansionSystemPrompt},
			{Role: "user", Content: content},
		},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("expand query failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	respContent, _ := apiResp.Choices[0].Message.Content.(string)

	var parsed domain.QueryExpansionResponse
	if err := json.Unmarshal([]byte(respContent), &parsed); err != nil {
		return []string{content}, nil
	}

	return parsed.Queries, nil
}

// ----------------------------------------------------------------------
// QUIZ
// ----------------------------------------------------------------------

func (s *OpenRouterService) GenerateQuizQuestion(ctx context.Context, chunk string) (*domain.QuizQuestion, error) {
	// Auto-detect subject from chunk content (zero-cost, keyword-based)
	subject := classifySubject(chunk)
	systemPrompt := buildSubjectSystemPrompt(subject)

	prompt := fmt.Sprintf(`Aşağıdaki tıbbi/bilimsel metin parçasını dikkatlice oku ve bu metne dayalı bir çoktan seçmeli soru üret.

--- METİN BAŞLANGIÇ ---
%s
--- METİN BİTİŞ ---

SORU ÜRETME ADIMLARI (Sırasıyla uygula, her adımı düşün):

1. METİN ANALİZİ: Metindeki en önemli tıbbi/klinik kavramı, ilişkiyi veya bilgiyi belirle. Konuyu "topic" alanına yaz.
2. ZORLUK BELİRLE: Sorunun zorluk seviyesini belirle ve "difficulty" alanına yaz.
3. SORU KÖKÜ YAZIMI: Bu kavramı ölçen, net ve anlaşılır bir soru kökü veya vaka yaz (20-60 kelime).
   - Soru kökü tek başına okunduğunda ne sorulduğu anlaşılmalı.
   - TUS veya tıp fakültesi komite sınavları tarzında olmalı.
4. DOĞRU ŞIK: Metne dayalı olarak doğru cevabı yaz.
5. ÇELDİRİCİ ŞIKLAR: Kalan 4 şıkkı oluştururken:
   a) Her çeldirici, soru kökünü doğal şekilde tamamlamalı veya cevaplamalıdır.
   b) Çeldiriciler, sorunun konusuyla AYNI konu alanından olmalıdır.
   c) Çeldiriciler mantıklı ama yanlış olmalıdır — rastgele veya alakasız bilgi YAZMA.
   d) Tüm şıklar benzer uzunlukta ve dil yapısında olmalıdır.
   e) Bir tıp öğrencisi şıklara baktığında hepsinin makul bir tıbbi cevap olabileceğini düşünmeli.
6. KENDİ KENDİNİ KONTROL ET: Soru kökünü oku, sonra her şıkkı soru köküne ekleyerek oku. Cümle doğal mı? Şıklar aynı kategoriden mi? Alakasız şık var mı?
7. AÇIKLAMA: Doğru cevabın neden doğru olduğunu VE diğer şıkların neden yanlış olduğunu tıbbi mekanizmalarla kısaca açıkla.

FORMAT KURALLARI:
- Tam 5 şık (A-E), sadece 1 tanesi doğru.
- Türkçe yaz.
- SADECE JSON çıktısı ver, başka metin yazma.`, chunk)

	schemaJSON, err := json.Marshal(quizQuestionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	apiReq := ChatCompletionRequest{
		Model: s.GenerateQuestionModel,
		Messages: []ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &ResponseFormatSchema{
				Name:   "quiz_question",
				Schema: json.RawMessage(schemaJSON),
				Strict: true,
			},
		},
		Temperature: 0.4,
	}

	question, err := s.executeQuizRequest(ctx, apiReq)
	if err != nil {
		// Retry once on failure
		question, err = s.executeQuizRequest(ctx, apiReq)
		if err != nil {
			return nil, fmt.Errorf("quiz generation failed after retry: %w", err)
		}
	}

	return question, nil
}

// executeQuizRequest sends the API request and validates the response
func (s *OpenRouterService) executeQuizRequest(ctx context.Context, apiReq ChatCompletionRequest) (*domain.QuizQuestion, error) {
	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("quiz generation failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty quiz response")
	}

	respContent, _ := apiResp.Choices[0].Message.Content.(string)
	content := strings.TrimSpace(respContent)

	var question domain.QuizQuestion
	if err := json.Unmarshal([]byte(content), &question); err != nil {
		return nil, fmt.Errorf("invalid quiz JSON: %w (content: %s)", err, content)
	}

	// Structural validation
	if question.QuestionText == "" {
		return nil, fmt.Errorf("question text is empty")
	}
	if len(question.Options) != 5 {
		return nil, fmt.Errorf("options count incorrect: %d", len(question.Options))
	}

	// Validate exactly 1 correct option
	correctCount := 0
	for _, o := range question.Options {
		if o.IsCorrect {
			question.CorrectOptionLabel = o.Label
			correctCount++
		}
	}
	if correctCount != 1 {
		return nil, fmt.Errorf("expected exactly 1 correct option, got %d", correctCount)
	}

	// Quality validation: check option length consistency
	var totalLen int
	for _, o := range question.Options {
		totalLen += len(o.Text)
	}
	avgLen := totalLen / 5
	for _, o := range question.Options {
		optLen := len(o.Text)
		// If any option is 5x longer or shorter than average, flag it
		if avgLen > 0 && (optLen > avgLen*5 || optLen < avgLen/5) {
			return nil, fmt.Errorf("option length inconsistency detected (option %s: %d chars, avg: %d chars)", o.Label, optLen, avgLen)
		}
	}

	// Quality validation: check minimum question length
	if len(question.QuestionText) < 20 {
		return nil, fmt.Errorf("question text too short: %d chars", len(question.QuestionText))
	}

	return &question, nil
}

// ----------------------------------------------------------------------
// VISION & REASONING
// ----------------------------------------------------------------------

func (s *OpenRouterService) ExtractTextFromImage(ctx context.Context, imageBase64 string) (string, error) {
	if !strings.HasPrefix(imageBase64, "data:image") {
		imageBase64 = "data:image/jpeg;base64," + imageBase64
	}

	parts := []ChatMessagePart{
		{Type: ChatMessagePartTypeText, Text: "Sen bir OCR motorusun. Resimdeki sorudaki metinleri ve çizimleri çıkar. Sakın soruyu çözme. Sadece text çıktısı ver."},
		{Type: ChatMessagePartTypeImageURL, ImageURL: &ChatMessageImageURL{URL: imageBase64}},
	}

	apiReq := ChatCompletionRequest{
		Model: s.VisionModel,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: parts},
		},
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return "", fmt.Errorf("ocr/vision request failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty ocr response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	return content, nil
}

func (s *OpenRouterService) SolveQuestionWithReasoning(ctx context.Context, questionText string) (string, error) {
	apiReq := ChatCompletionRequest{
		Model: s.ReasoningModel,
		Messages: []ChatCompletionMessage{
			{
				Role: "system",
				Content: `Soru çözücüsün. Giriş cümlesi olmadan direkt çözüme başla.

FORMAT (Markdown kullan):
### Konu: [Konuyu kısa belirt]

###  Çözüm
Adımları numaralı liste halinde yaz. Her adımda:
- Önemli terimleri **kalın** yap
- Formülleri LaTeX formatında yaz: $formül$
- Gerekirse kısa açıklama ekle

### Cevap
Sonucu net ve belirgin şekilde yaz.

KURALLAR:
- Kendini tanıtma, selamlama, "Merhaba", "Hadi bakalım" gibi dolgu cümleler YAZMA
- Soruyu tekrar alıntılama
- Her adım öz ve anlaşılır olsun, ama gerekli açıklamayı atla demek değil
- Türkçe yanıtla`,
			},
			{
				Role:    "user",
				Content: questionText,
			},
		},
		Temperature: 0.4,
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return "", fmt.Errorf("reasoning/solving failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty reasoning response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	return content, nil
}

// ----------------------------------------------------------------------
// EMBEDDING
// ----------------------------------------------------------------------

func (s *OpenRouterService) GenerateEmbedding(ctx context.Context, req *domain.SingleEmbeddingRequest) (*domain.SingleEmbeddingResponse, error) {
	apiReq := EmbeddingRequest{
		Input: req.Text,
		Model: s.EmbeddingModel,
	}

	var apiResp EmbeddingResponse
	if err := s.makeRequest(ctx, "POST", "/embeddings", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	src := apiResp.Data[0].Embedding
	vector := make([]float32, len(src))
	for i, v := range src {
		vector[i] = float32(v)
	}

	return &domain.SingleEmbeddingResponse{
		Vector: vector,
	}, nil
}

func (s *OpenRouterService) GenerateEmbeddingBatch(ctx context.Context, req *domain.EmbeddingBatchRequest) (*domain.EmbeddingResponse, error) {
	// []Text -> []string
	inputs := make([]string, len(req.Texts))
	for i, t := range req.Texts {
		inputs[i] = t.Text
	}

	apiReq := EmbeddingRequest{
		Input: inputs,
		Model: s.EmbeddingModel,
	}

	var apiResp EmbeddingResponse
	if err := s.makeRequest(ctx, "POST", "/embeddings", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	results := make([]domain.Embedding, len(apiResp.Data))
	for i, data := range apiResp.Data {
		vector := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vector[j] = float32(v)
		}

		results[i] = domain.Embedding{
			Id:          req.Texts[i].Id,
			Text:        req.Texts[i].Text,
			Vector:      vector,
			PageNumbers: []int{},
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("empty embedding results")
	}

	return &domain.EmbeddingResponse{
		Results:   results,
		Dimension: len(results[0].Vector),
		Count:     len(results),
	}, nil
}
