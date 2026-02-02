package domain

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type QueryType string

const (
	PdfSearchQuery     QueryType = "pdf_search_query"
	ClarificationQuery QueryType = "clarification_query"
	GeneralQuery       QueryType = "general_query"
	ImageQuery         QueryType = "image_query"
)

const (
	generalQuerySystemMessage = "Sen bir YKS rehberlik asistanısın. Öğrencinin sınav süreci, çalışma programı, motivasyon ve genel sorularını en iyi şekilde yanıtla. Cevaplarını Türkçe ver."

	clarificationQuerySystemMessage = "Sen bir öğrenci koçu ve eğitmensin. Önceki verdiğin cevabı daha basitleştirerek veya detaylandırarak açıkla. Yeni konu açma, mevcut konuyu pekiştirerek anlat."

	imageQuerySystemMessage = "Sen bir soru çözüm asistanısın. Gönderilen görseldeki soruyu analiz et ve çözümünü ver. Eğer görsel dersle/sınavla ilgili değilse, nazikçe cevap veremeyeceğini belirt."

	pdfSearchQuerySystemMessage = `Sen Uzman bir YKS Eğitmenisin. Soruları SADECE aşağıda verilen Bağlam (Context) içeriğini kullanarak cevapla.

KRİTİK KURALLAR:
1. **Cevap Kontrolü:**
   - ÖNCE verilen Bağlamı dikkatlice ara.
   - Eğer Bağlam içinde soruyla ilgili HERHANGİ bir bilgi varsa, mutlaka kullan.
   - SADECE ve SADECE bağlamda HİÇBİR bilgi yoksa "Yüklediğiniz dokümanda bu bilgi yer almamaktadır" de.
2. **Cevap Oluşturma:**
   - Önemli terimler için **Kalın** font kullan.
   - Listeler için madde işaretleri kullan.
   - Bağlam/Doküman içindeki bilgileri kullanarak kapsamlı ve doyurucu bir cevap ver.
   - Mümkünse dokümandan alıntı yap veya oradaki bilgiyi yorumla.
3. **Dil:**
   - Türkçe cevapla.
4. **Önemli:**
   - Kısmi bilgi varsa bile kullan.
   - Konuyla dolaylı ilgili bilgi varsa ekle.
   - Çok katı olma, öğrenciye yardımcı olmaya çalış (dokümandan sapmadan).
`

	QueryClassificationSystemPrompt = `Kullanıcının sorgusunu tam olarak aşağıdaki kategorilerden birine sınıflandır.
Sadece tek bir kelime döndür: PDFSearch, Clarification veya General.

Kuralları sırasıyla uygula:

1. PDFSearch
   Eğer aşağıdakilerden HERHANGİ biri geçerliyse PDFSearch olarak sınıflandır:
     - Kullanıcı doküman/kitap içinde olabilecek bir bilgi soruyorsa.
       (Örnekler: "X nedir", "Y'yi açıkla", "Z hakkında bilgi ver", "A bölümünü özetle")
     - Sorgu bir konu, tanım, açıklama veya bilgi içeriyorsa.
     - Sorgu dersle ilgiliyse (Matematik, Fizik, Türkçe, Tarih vb.).
     - Emin değilsen (Varsayılan olarak PDFSearch seç).
   Örnekler:
     "Türev alma kuralları nelerdir?"
     "Paragraf taktikleri ver"
     "Osmanlı duraklama dönemi nedenleri"

2. Clarification
   SADECE şu durumlarda Clarification seç:
     - Sorgu açıkça önceki cevabına veya kullanıcının önceki sorusuna atıfta bulunuyorsa.
     - Kullanıcı daha fazla detay, basitleştirme veya açıklama istiyorsa.
   Örnekler:
     "Daha detaylı anlatabilir misin?"
     "Ne demek istedim?"
     "Son kısmı açar mısın?"
     "Devam et"

3. General
   SADECE şu durumlarda General seç:
     - Kullanıcı selamlaşma, sohbet veya ders dışı kişisel sorular soruyorsa.
   Örnekler:
     "merhaba"
     "kimsin sen?"
     "nasılsın?"
     "bir fıkra anlat"

	 Talimatlar:
	Tek bir kelime ile cevap ver: "PDFSearch", "Clarification" veya "General". Emin olmadığında dersle ilgili her şey için "PDFSearch" seç.`

	QueryExpansionSystemPrompt = `Sen bir arama sorgusu genişletme uzmanısın. Görevin, kullanıcının asıl niyetini koruyarak farklı açılardan 5 adet Türkçe arama sorgusu üretmektir.

	Kurallar:
	1. Asıl niyeti koru - kullanıcının ne sorduğunu değiştirme.
	2. Farklı ifadeler, eş anlamlılar ve bakış açıları kullan.
	3. YKS (TYT/AYT) müfredatına uygun terimler kullanmaya çalış.
	4. Asıl sorguda olmayan tamamen yeni bilgi ekleme.

	SADECE şu formatta geçerli bir JSON döndür:
	{"queries": ["...", "...", "...", "...", "..."]}
`
)

type ChatRequest struct {
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Text string `json:"text"`
}

type QueryExpansionResponse struct {
	Queries []string `json:"queries"`
}

type UpdateChatRequest struct {
	Title string `json:"title"`
}

type UpdateChatResponse struct {
}

type ChatMessage struct {
	Role     Role     `json:"role"`
	Content  string   `json:"content"`
	ImageUrl []string `json:"image_url,omitempty"`
}

type ChatRequestManager struct {
	QueryType   QueryType
	UserContent string
	Req         *ChatRequest
}

func NewChatRequestManager(queryType QueryType, userContent string) *ChatRequestManager {
	return &ChatRequestManager{
		QueryType:   queryType,
		UserContent: userContent,
		Req:         NewChatRequest(queryType, userContent),
	}
}

func NewChatMessage(role Role, content string) ChatMessage {
	return ChatMessage{
		Role:    role,
		Content: content,
	}
}

func NewChatRequest(queryType QueryType, userContent string) *ChatRequest {
	systemMessage := getSystemMessageForQueryType(queryType)

	return &ChatRequest{
		Messages: []ChatMessage{
			NewChatMessage(RoleSystem, systemMessage),
			NewChatMessage(RoleUser, userContent),
		},
	}
}

func getSystemMessageForQueryType(queryType QueryType) string {
	switch queryType {
	case GeneralQuery:
		return generalQuerySystemMessage
	case ClarificationQuery:
		return clarificationQuerySystemMessage
	case PdfSearchQuery:
		return pdfSearchQuerySystemMessage
	case ImageQuery:
		return imageQuerySystemMessage
	default:
		return generalQuerySystemMessage
	}
}

func (r *ChatRequest) AddRelevantExcerpts(excerpts []string) {
	if len(excerpts) == 0 {
		return
	}

	for _, excerpt := range excerpts {
		r.Messages = append(r.Messages, NewChatMessage(RoleUser, excerpt))
	}
}

func (r *ChatRequest) AppendMessages(messages []ChatMessage) {
	r.Messages = append(r.Messages, messages...)
}

func (r *ChatRequest) RemoveUserMessageToEnd() {
	if len(r.Messages) < 2 {
		return
	}
	second := r.Messages[1]
	r.Messages = append(r.Messages[:1], r.Messages[2:]...)
	r.Messages = append(r.Messages, second)
}

func (cm *ChatMessage) UpdateRole(isUser bool) {
	if isUser {
		cm.Role = RoleUser
		return
	}
	cm.Role = RoleAssistant
}

func NewChatTitleRequest(firstUserMessage string) *ChatRequest {
	return &ChatRequest{
		Stream:   false,
		Messages: buildTitleGenerationMessage(firstUserMessage),
	}
}

func buildTitleGenerationMessage(firstUserMessage string) []ChatMessage {
	return []ChatMessage{
		NewChatMessage(
			RoleSystem,
			"Generate a 3-7 word title for this chat in turkish: "+firstUserMessage,
		),
	}
}

func (crm *ChatRequestManager) IsGeneralQuery() bool {
	return crm.QueryType == GeneralQuery
}

func (crm *ChatRequestManager) IsClarificationQuery() bool {
	return crm.QueryType == ClarificationQuery
}

func (crm *ChatRequestManager) IsPdfSearchQuery() bool {
	return crm.QueryType == PdfSearchQuery
}
