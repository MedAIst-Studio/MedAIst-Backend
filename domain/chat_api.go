package domain

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type QueryType string

const (
	PdfSearchQuery   QueryType = "pdf_search_query"
	ImageQuery       QueryType = "image_query"
	GeneralChatQuery QueryType = "general_chat_query"
)

const (
	generalChatSystemMessage = `Sen yardımsever ve samimi bir yapay zeka asistanısın. Kullanıcının her türlü sorusuna bilgilendirici ve doğal yanıtlar ver.
Cevaplarını Türkçe ver. Gerektiğinde Markdown formatını kullan.`

	imageQuerySystemMessage = "Sen bir soru çözüm asistanısın. Gönderilen görseldeki soruyu analiz et ve çözümünü ver. Eğer görsel dersle/sınavla ilgili değilse, nazikçe cevap veremeyeceğini belirt."

	pdfSearchQuerySystemMessage = `Sen Uzman bir YKS Eğitmenisin. Soruları SADECE aşağıda verilen Bağlam (Context) içeriğini kullanarak cevapla.

KRİTİK KURALLAR:
1. **Cevap Kontrolü:**
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
   - Konuyla dolaylı ilgili bilgi varsa ekle.
`

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
	case PdfSearchQuery:
		return pdfSearchQuerySystemMessage
	case ImageQuery:
		return imageQuerySystemMessage
	case GeneralChatQuery:
		return generalChatSystemMessage
	default:
		return generalChatSystemMessage
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

func (crm *ChatRequestManager) IsPdfSearchQuery() bool {
	return crm.QueryType == PdfSearchQuery
}

func (crm *ChatRequestManager) IsGeneralChatQuery() bool {
	return crm.QueryType == GeneralChatQuery
}
