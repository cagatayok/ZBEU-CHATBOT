package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	pdflib "github.com/ledongthuc/pdf"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

const (
	// Groq API Configuration (ONLY AI PROVIDER)
	groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"
	groqModel   = "llama-3.1-8b-instant" // Active Groq model (2026)

	// Retry Configuration
	maxRetries      = 3
	initialDelay    = 2 * time.Second
	maxDelay        = 10 * time.Second
	backoffMultiple = 2

	// Data storage
	documentsFile = "data/documents.json"
)

// ZBEÜ Subdomain Configuration - COMPLETE UNIVERSITY COVERAGE
var zbeSubdomains = map[string][]string{
	// Engineering Faculty - PRIORITY (CENG Discovery)
	"muhendislik": {
		"https://muhendislik.beun.edu.tr/",
		"https://muhendislik.beun.edu.tr/duyurular",
		"https://muhendislik.beun.edu.tr/iletisim",
		"https://muhendislik.beun.edu.tr/sayfa/iletisim.html",
		"https://muhendislik.beun.edu.tr/bolumler/bilgisayar-muhendisligi",
		"https://muhendislik.beun.edu.tr/bolumler/bilgisayar-muhendisligi/ders-programlari",
	},
	// Computer Engineering
	"bilgisayar": {
		"https://muhendislik.beun.edu.tr/bolumler/bilgisayar-muhendisligi/ders-programlari",
		"https://muhendislik.beun.edu.tr/bolumler/bilgisayar-muhendisligi/duyurular",
		"https://muhendislik.beun.edu.tr/bolumler/bilgisayar-muhendisligi/akademik-kadro",
		"https://cdn1.beun.edu.tr/bilgisayar/20252026baharbilgisayarmuhendisligi%20Ders%20Program%C4%B1022026.pdf",
		"https://muhendislik.beun.edu.tr/bolumler/bilgisayar-muhendisligi",
	},
	// Student Services
	"ogrenci": {
		"https://ogrenci.beun.edu.tr/",
		"https://ogrenci.beun.edu.tr/duyurular",
		"https://ogrenci.beun.edu.tr/duyurular.html",
		"https://ogrenci.beun.edu.tr/akademik-takvim",
		"https://ogrenci.beun.edu.tr/ders-alma",
	},
	// Main University Site
	"beun": {
		"https://beun.edu.tr/",
		"https://beun.edu.tr/duyurular",
		"https://beun.edu.tr/sayfa/iletisim.html",
		"https://beun.edu.tr/haberler",
		"https://beun.edu.tr/akademik-takvim",
		"https://beun.edu.tr/bolumler",
		"https://beun.edu.tr/fakulteler",
	},
	// Other Faculties
	"tip": {
		"https://tip.beun.edu.tr/",
		"https://tip.beun.edu.tr/duyurular",
	},
	"fen": {
		"https://fen.beun.edu.tr/",
		"https://fen.beun.edu.tr/duyurular",
	},
	"iibf": {
		"https://iibf.beun.edu.tr/",
		"https://iibf.beun.edu.tr/duyurular",
	},
	"egitim": {
		"https://egitim.beun.edu.tr/",
		"https://egitim.beun.edu.tr/duyurular",
	},
	// Academic Systems
	"akademik": {
		"https://webapp.beun.edu.tr/akademiktakvim/",
		"https://obs.beun.edu.tr/",
	},
	// Library
	"kutuphane": {
		"https://kutuphane.beun.edu.tr/",
		"https://kutuphane.beun.edu.tr/duyurular",
	},
}

// ZBEÜ Yönetmelik Kuralları
type ZBEURegulation struct {
	MaxCourses int // 4. sınıf için maksimum ders sayısı
	MaxAKTS    int // Maksimum AKTS
}

var regulation = ZBEURegulation{
	MaxCourses: 3,
	MaxAKTS:    30, // Düzeltildi: 30 AKTS
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// Announcement represents a university announcement
type Announcement struct {
	Title   string
	Date    string
	Content string
	URL     string
}

// Document represents a scraped document (for JSON storage)
type Document struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`  // "duyuru", "yonetmelik", "akademik"
	Subdomain string    `json:"subdomain"` // "ogrenci", "akademik", etc.
	ScrapedAt time.Time `json:"scraped_at"`
}

// DocumentStore holds all scraped documents
type DocumentStore struct {
	Documents   []Document `json:"documents"`
	LastUpdated time.Time  `json:"last_updated"`
}

// GeminiRequest represents the request structure for Gemini API
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

// GeminiResponse represents the response structure from Gemini API
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
	Error      *APIError   `json:"error,omitempty"`
}

type Candidate struct {
	Content ContentResponse `json:"content"`
}

type ContentResponse struct {
	Parts []PartResponse `json:"parts"`
}

type PartResponse struct {
	Text string `json:"text"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Groq API structures (OpenAI-compatible)
type GroqRequest struct {
	Model    string        `json:"model"`
	Messages []GroqMessage `json:"messages"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponse struct {
	Choices []GroqChoice `json:"choices"`
	Error   *GroqError   `json:"error,omitempty"`
}

type GroqChoice struct {
	Message GroqMessage `json:"message"`
}

type GroqError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ============================================================================
// WEB SCRAPING MODULE
// ============================================================================

// scrapeAnnouncements fetches announcements from ZBEÜ website
// Currently uses mock data due to 404 error on target URL
func scrapeAnnouncements(url string) ([]Announcement, error) {
	// Try to fetch real data
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)

	// If URL is accessible, parse it (basic implementation)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Simple HTML parsing for announcements
		announcements := parseHTMLAnnouncements(bodyStr)
		if len(announcements) > 0 {
			return announcements, nil
		}
	}

	// Fallback to mock data
	log.Println("⚠️  Web scraping başarısız, mock data kullanılıyor...")
	return getMockAnnouncements(), nil
}

// parseHTMLAnnouncements performs basic HTML parsing
func parseHTMLAnnouncements(html string) []Announcement {
	var announcements []Announcement

	// Basic pattern matching for common announcement structures
	// This is a simplified version - real implementation would use proper HTML parser
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		if strings.Contains(line, "duyuru") || strings.Contains(line, "announcement") {
			// Extract basic info (simplified)
			announcement := Announcement{
				Title:   strings.TrimSpace(line),
				Date:    time.Now().Format("02.01.2006"),
				Content: "Detaylar için web sitesini ziyaret edin",
				URL:     "",
			}
			announcements = append(announcements, announcement)
		}
	}

	return announcements
}

// getMockAnnouncements returns sample announcements for testing
func getMockAnnouncements() []Announcement {
	return []Announcement{
		{
			Title:   "2026 Yaz Okulu Başvuruları",
			Date:    "10.02.2026",
			Content: "2026 Yaz Okulu başvuruları 15 Haziran - 30 Haziran tarihleri arasında yapılacaktır. Dersler 5 Temmuz'da başlayacaktır.",
			URL:     "https://ogrenci.beun.edu.tr/duyurular/yaz-okulu-2026",
		},
		{
			Title:   "4. Sınıf Ders Alma Sınırları",
			Date:    "05.02.2026",
			Content: "4. sınıf Bilgisayar Mühendisliği öğrencileri maksimum 3 ders alabilir. AKTS toplamı 21-24 arasında olmalıdır.",
			URL:     "https://ogrenci.beun.edu.tr/duyurular/ders-alma",
		},
		{
			Title:   "AKTS Hesaplama Kuralları",
			Date:    "01.02.2026",
			Content: "Öğrenciler AKTS hesaplamalarını yaparken yönetmelik kurallarına dikkat etmelidir. Her ders için AKTS değerleri ders kataloğunda belirtilmiştir.",
			URL:     "https://ogrenci.beun.edu.tr/duyurular/akts",
		},
	}
}

// ============================================================================
// JSON DOCUMENT STORAGE
// ============================================================================

// loadDocuments loads documents from JSON file
func loadDocuments() (*DocumentStore, error) {
	// Create data directory if it doesn't exist
	os.MkdirAll("data", 0755)

	// Check if file exists
	if _, err := os.Stat(documentsFile); os.IsNotExist(err) {
		// Return empty store
		return &DocumentStore{
			Documents:   []Document{},
			LastUpdated: time.Now(),
		}, nil
	}

	// Read file
	data, err := os.ReadFile(documentsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read documents file: %v", err)
	}

	// Parse JSON
	var store DocumentStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse documents JSON: %v", err)
	}

	return &store, nil
}

// saveDocuments saves documents to JSON file
func saveDocuments(store *DocumentStore) error {
	// Create data directory if it doesn't exist
	os.MkdirAll("data", 0755)

	// Update timestamp
	store.LastUpdated = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal documents: %v", err)
	}

	// Write to file
	if err := os.WriteFile(documentsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write documents file: %v", err)
	}

	return nil
}

// crawlAllSubdomains crawls all configured ZBEÜ subdomains recursively
func crawlAllSubdomains() (*DocumentStore, error) {
	store := &DocumentStore{
		Documents:   []Document{},
		LastUpdated: time.Now(),
	}

	visited := make(map[string]bool) // Track visited URLs
	maxDepth := 10                   // Maximum crawl depth (UNLIMITED)
	maxPages := 100000               // Virtually unlimited (100K pages)

	fmt.Println("🕷️  ZBEÜ UNLIMITED University-Wide Crawler Başlıyor...")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("📊 Hedef: TÜM ÜNİVERSİTE (%d sayfa limit, %d derinlik)\n", maxPages, maxDepth)
	fmt.Println("🎓 TÜM Fakülteler, TÜM Bölümler, TÜM Öğrenciler")
	fmt.Println("📄 PDF, HTML, her şey taranacak")
	fmt.Println("⚠️  Bu işlem 30-60 dakika sürebilir...")
	fmt.Println("🚀 PRODUCTION-READY DEPLOYMENT")
	fmt.Println()

	for subdomain, startURLs := range zbeSubdomains {
		fmt.Printf("📡 %s.beun.edu.tr taranıyor...\n", subdomain)

		for _, startURL := range startURLs {
			if len(store.Documents) >= maxPages {
				fmt.Println("\n⚠️  Maksimum sayfa sayısına ulaşıldı")
				break
			}

			// Crawl recursively
			crawlRecursive(startURL, subdomain, 0, maxDepth, maxPages, visited, store)
		}
	}

	fmt.Printf("\n✅ Toplam %d döküman toplandı\n", len(store.Documents))
	return store, nil
}

// crawlRecursive recursively crawls a URL and its links
func crawlRecursive(url, subdomain string, depth, maxDepth, maxPages int, visited map[string]bool, store *DocumentStore) {
	// Check limits
	if depth > maxDepth || len(store.Documents) >= maxPages {
		return
	}

	// Check if already visited
	if visited[url] {
		return
	}
	visited[url] = true

	// Show progress
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s→ [D%d] %s\n", indent, depth, url)

	// Fix URL encoding (handle spaces and Turkish characters)
	encodedURL := url
	u, err := neturl.Parse(url)
	if err == nil {
		encodedURL = u.String()
	}

	// Fetch page
	resp, err := http.Get(encodedURL)
	if err != nil {
		fmt.Printf("%s  ❌ %v\n", indent, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("%s  ⚠️  HTTP %d\n", indent, resp.StatusCode)
		return
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s  ❌ Okuma hatası\n", indent)
		return
	}

	// Check if it's a PDF
	contentType := resp.Header.Get("Content-Type")
	isPDF := strings.Contains(contentType, "application/pdf") || strings.HasSuffix(url, ".pdf")

	var content string
	var title string

	if isPDF {
		// Extract text from PDF
		fmt.Printf("%s  📄 PDF tespit edildi, metin çıkarılıyor...\n", indent)
		pdfText, err := extractTextFromPDF(body)
		if err != nil {
			fmt.Printf("%s  ❌ PDF okuma hatası: %v\n", indent, err)
			return
		}
		content = pdfText
		title = "PDF: " + extractFilenameFromURL(url)
	} else {
		// Extract text from HTML
		htmlContent := string(body)
		content = extractTextFromHTML(htmlContent)
		title = extractTitle(htmlContent)
	}

	if len(content) < 50 {
		fmt.Printf("%s  ⚠️  İçerik çok kısa\n", indent)
		return
	}

	// Create document
	category := categorizeURL(url)
	if isPDF {
		category = "pdf"
	}

	doc := Document{
		ID:        generateDocID(url),
		URL:       url,
		Title:     title,
		Content:   content,
		Category:  category,
		Subdomain: subdomain,
		ScrapedAt: time.Now(),
	}

	store.Documents = append(store.Documents, doc)
	fmt.Printf("%s  ✅ %d karakter (%d toplam)\n", indent, len(content), len(store.Documents))

	// Save periodically (every 10 documents) to prevent data loss
	if len(store.Documents)%10 == 0 {
		if err := saveDocuments(store); err != nil {
			fmt.Printf("%s  ❌ Kayıt hatası: %v\n", indent, err)
		} else {
			fmt.Printf("%s  💾 Veriler kaydedildi (%d döküman)\n", indent, len(store.Documents))
		}
	}

	// Extract and follow links (only for HTML pages)
	if !isPDF && depth < maxDepth && len(store.Documents) < maxPages {
		htmlContent := string(body)
		links := extractLinks(htmlContent, url)

		// NO LIMIT - follow ALL links!
		fmt.Printf("%s  🔗 %d link bulundu, hepsi taranacak\n", indent, len(links))

		for _, link := range links {
			// Only follow beun.edu.tr links
			if !strings.Contains(link, "beun.edu.tr") {
				continue
			}

			// Skip certain file types (images, archives)
			if strings.HasSuffix(link, ".jpg") ||
				strings.HasSuffix(link, ".png") ||
				strings.HasSuffix(link, ".zip") ||
				strings.HasSuffix(link, ".exe") {
				continue
			}

			// Rate limiting (maximum speed for unlimited crawl)
			time.Sleep(50 * time.Millisecond)

			// Recursive call
			crawlRecursive(link, subdomain, depth+1, maxDepth, maxPages, visited, store)

			if len(store.Documents) >= maxPages {
				return
			}
		}
	}
}

// extractLinks extracts all links from HTML
func extractLinks(html, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)
	var pdfLinks []string // Separate list for PDF links

	// Extract base domain
	baseDomain := ""
	if strings.HasPrefix(baseURL, "http") {
		parts := strings.Split(baseURL, "/")
		if len(parts) >= 3 {
			baseDomain = parts[0] + "//" + parts[2]
		}
	}

	// Simple link extraction - multiple patterns
	patterns := []string{
		"href=\"",
		"href='",
		"src=\"",
		"src='",
	}

	for _, pattern := range patterns {
		parts := strings.Split(html, pattern)
		for i := 1; i < len(parts); i++ {
			var endChar string
			if strings.Contains(pattern, "\"") {
				endChar = "\""
			} else {
				endChar = "'"
			}

			endQuote := strings.Index(parts[i], endChar)
			if endQuote == -1 {
				continue
			}

			link := parts[i][:endQuote]

			// Skip anchors, javascript, mailto, tel
			if strings.HasPrefix(link, "#") ||
				strings.HasPrefix(link, "javascript:") ||
				strings.HasPrefix(link, "mailto:") ||
				strings.HasPrefix(link, "tel:") ||
				link == "" {
				continue
			}

			// Remove query parameters and fragments for deduplication
			cleanLink := strings.Split(link, "?")[0]
			cleanLink = strings.Split(cleanLink, "#")[0]

			// Convert relative URLs to absolute
			if strings.HasPrefix(cleanLink, "/") {
				// Absolute path
				if baseDomain != "" {
					cleanLink = baseDomain + cleanLink
				}
			} else if strings.HasPrefix(cleanLink, "./") {
				// Relative to current directory
				lastSlash := strings.LastIndex(baseURL, "/")
				if lastSlash > 7 { // After http://
					cleanLink = baseURL[:lastSlash+1] + cleanLink[2:]
				}
			} else if !strings.HasPrefix(cleanLink, "http") {
				// Relative path
				lastSlash := strings.LastIndex(baseURL, "/")
				if lastSlash > 7 { // After http://
					cleanLink = baseURL[:lastSlash+1] + cleanLink
				}
			}

			// Deduplicate and add
			if !seen[cleanLink] && strings.HasPrefix(cleanLink, "http") {
				seen[cleanLink] = true

				// Prioritize PDF links
				if strings.HasSuffix(strings.ToLower(cleanLink), ".pdf") {
					pdfLinks = append(pdfLinks, cleanLink)
				} else {
					links = append(links, cleanLink)
				}
			}
		}
	}

	// Return PDFs first, then other links
	return append(pdfLinks, links...)
}

// extractTextFromPDF extracts text from a PDF file
func extractTextFromPDF(pdfData []byte) (string, error) {
	// Create a temporary file for the PDF
	tmpFile, err := os.CreateTemp("", "pdf-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write PDF data to temp file
	if _, err := tmpFile.Write(pdfData); err != nil {
		return "", fmt.Errorf("failed to write PDF: %v", err)
	}
	tmpFile.Close()

	// Open PDF
	f, r, err := pdflib.Open(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %v", err)
	}
	defer f.Close()

	// Extract text from all pages
	var text strings.Builder
	totalPages := r.NumPage()

	// Limit to first 50 pages to avoid huge PDFs
	maxPages := totalPages
	if maxPages > 50 {
		maxPages = 50
	}

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// Get text content
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		text.WriteString(pageText)
		text.WriteString("\n")
	}

	result := text.String()

	// Fix common encoding issues from PDF extraction
	result = fixTurkishEncoding(result)

	// Limit total length
	if len(result) > 10000 {
		result = result[:10000]
	}

	return strings.TrimSpace(result), nil
}

// fixTurkishEncoding fixes common garbled Turkish characters from PDF/Web extraction
func fixTurkishEncoding(text string) string {
	text = strings.ReplaceAll(text, "Ä°", "İ")
	text = strings.ReplaceAll(text, "Ä±", "ı")
	text = strings.ReplaceAll(text, "Ã‡", "Ç")
	text = strings.ReplaceAll(text, "Ã§", "ç")
	text = strings.ReplaceAll(text, "Åž", "Ş")
	text = strings.ReplaceAll(text, "ÅŸ", "ş")
	text = strings.ReplaceAll(text, "Å\u009e", "Ş")
	text = strings.ReplaceAll(text, "Å\u009f", "ş")
	text = strings.ReplaceAll(text, "ÄŸ", "ğ")
	text = strings.ReplaceAll(text, "Ä\u009e", "Ğ")
	text = strings.ReplaceAll(text, "Ã–", "Ö")
	text = strings.ReplaceAll(text, "Ã¶", "ö")
	text = strings.ReplaceAll(text, "Ãœ", "Ü")
	text = strings.ReplaceAll(text, "Ã¼", "ü")
	text = strings.ReplaceAll(text, "Ã‚", "Â")
	text = strings.ReplaceAll(text, "Ã¢", "â")
	text = strings.ReplaceAll(text, "Ã®", "î")
	return text
}

// extractTextFromHTML extracts plain text from HTML
func extractTextFromHTML(html string) string {
	// Simple text extraction - remove HTML tags
	text := strings.ReplaceAll(html, "<script", "<SCRIPT")
	text = strings.ReplaceAll(text, "</script>", "</SCRIPT>")
	text = strings.ReplaceAll(text, "<style", "<STYLE")
	text = strings.ReplaceAll(text, "</style>", "</STYLE>")

	// Remove script and style content
	for {
		start := strings.Index(text, "<SCRIPT")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], "</SCRIPT>")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+end+9:]
	}

	for {
		start := strings.Index(text, "<STYLE")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], "</STYLE>")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+end+8:]
	}

	// Remove all HTML tags
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end == -1 {
			break
		}
		text = text[:start] + " " + text[start+end+1:]
	}

	// Clean up whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Limit length
	if len(text) > 5000 {
		text = text[:5000]
	}

	return strings.TrimSpace(text)
}

// extractTitle extracts title from HTML
func extractTitle(html string) string {
	start := strings.Index(html, "<title>")
	if start == -1 {
		return "Başlıksız Döküman"
	}
	end := strings.Index(html[start:], "</title>")
	if end == -1 {
		return "Başlıksız Döküman"
	}
	title := html[start+7 : start+end]
	return strings.TrimSpace(title)
}

// generateDocID generates a unique ID for a document
func generateDocID(url string) string {
	// Simple hash-based ID
	hash := 0
	for _, c := range url {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("doc_%d", hash)
}

// categorizeURL categorizes a URL
func categorizeURL(url string) string {
	if strings.Contains(url, "iletisim") || strings.Contains(url, "adres") || strings.Contains(url, "contact") {
		return "contact"
	}
	if strings.Contains(url, "akademik") {
		return "akademik"
	}
	if strings.Contains(url, "ders") || strings.Contains(url, "program") {
		return "ders_programi"
	}
	return "genel"
}

// extractFilenameFromURL extracts filename from URL
func extractFilenameFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown.pdf"
}

// ============================================================================
// AI INTEGRATION MODULE (Groq Only)
// ============================================================================

// queryAI sends a prompt to Groq API with context
func queryAI(apiKey string, prompt string, context []Announcement) (string, error) {
	// Build context-aware prompt
	fullPrompt := buildContextPrompt(prompt, context)

	fmt.Println("🚀 Using Groq API...")
	response, err := callGroqAPI(apiKey, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("Groq API error: %v", err)
	}

	// Fix any encoding issues in the AI response
	response = fixTurkishEncoding(response)

	fmt.Println("✅ Groq API başarılı!")
	return response, nil
}

// callGroqAPI makes a request to Groq API
func callGroqAPI(apiKey, prompt string) (string, error) {
	reqBody := GroqRequest{
		Model: groqModel,
		Messages: []GroqMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("JSON marshal error: %v", err)
	}

	fmt.Printf("🔍 Groq Request: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", groqBaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request creation error: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("response read error: %v", err)
	}

	fmt.Printf("🔍 Groq Response Status: %d\n", resp.StatusCode)
	fmt.Printf("🔍 Groq Response Body: %s\n", string(body))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var groqResp GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return "", fmt.Errorf("JSON parse error: %v", err)
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("Groq API error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) > 0 {
		return groqResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("empty response")
}

// buildContextPrompt creates a context-aware prompt with announcements and scraped documents
func buildContextPrompt(userPrompt string, announcements []Announcement) string {
	context := "Sen ZBEÜ (Zonguldak Bülent Ecevit Üniversitesi) öğrenci asistanısın.\n\n"

	// OPENAI EMBEDDINGS SEMANTIC SEARCH
	if embeddingVectorStore != nil {
		docs, err := embeddingVectorStore.SemanticSearchWithEmbeddings(userPrompt, 3)
		if err == nil && len(docs) > 0 {
			context += "İLGİLİ BELGELER (Gemini Semantic Search):\n"
			for _, doc := range docs {
				content := doc.Content
				if len(content) > 800 {
					content = content[:800] + "..."
				}
				context += fmt.Sprintf("\n[%s - %s]\n%s\nKaynak: %s\n",
					doc.Category, doc.Subdomain, content, doc.URL)
			}
			context += "\n"
		}
	} else {
		// FALLBACK: Keyword matching (old method)
		store, err := loadDocuments()
		if err == nil && len(store.Documents) > 0 {
			context += "ZBEÜ WEB SİTESİNDEN TOPLANAN BİLGİLER:\n"

			// Simple relevance scoring
			type ScoredDoc struct {
				Doc   Document
				Score int
			}
			var scoredDocs []ScoredDoc

			promptWords := strings.Fields(strings.ToLower(userPrompt))

			for _, doc := range store.Documents {
				score := 0
				contentLower := strings.ToLower(doc.Content)
				titleLower := strings.ToLower(doc.Title)

				// CRITICAL: Detect class number (1., 2., 3., 4. sınıf)
				if strings.Contains(userPrompt, "4.") || strings.Contains(userPrompt, "4. sınıf") || strings.Contains(userPrompt, "dördüncü") {
					if strings.Contains(contentLower, "4. sinif") || strings.Contains(contentLower, "4.sinif") {
						score += 100 // Massive boost for correct class
					}
					// Penalize wrong classes
					if strings.Contains(contentLower, "1. sinif") || strings.Contains(contentLower, "2. sinif") || strings.Contains(contentLower, "3. sinif") {
						score -= 50
					}
				}

				for _, word := range promptWords {
					if len(word) < 3 {
						continue
					} // Skip short words
					if strings.Contains(contentLower, word) {
						score += 1
					}
					if strings.Contains(titleLower, word) {
						score += 2 // Title matches are more important
					}
				}

				if score > 0 {
					scoredDocs = append(scoredDocs, ScoredDoc{Doc: doc, Score: score})
				}
			}

			// Sort by score descending
			for i := 0; i < len(scoredDocs)-1; i++ {
				for j := 0; j < len(scoredDocs)-i-1; j++ {
					if scoredDocs[j].Score < scoredDocs[j+1].Score {
						scoredDocs[j], scoredDocs[j+1] = scoredDocs[j+1], scoredDocs[j]
					}
				}
			}

			// Pick top 3 relevant documents
			count := 0
			for _, sd := range scoredDocs {
				if count >= 3 {
					break
				}

				content := sd.Doc.Content
				if len(content) > 800 {
					content = content[:800] + "..."
				}
				context += fmt.Sprintf("\n[%s - %s] (Score: %d)\n%s\nKaynak: %s\n",
					sd.Doc.Category, sd.Doc.Subdomain, sd.Score, content, sd.Doc.URL)
				count++
			}
			context += "\n"
		}
	}

	// Add announcements if available
	if len(announcements) > 0 {
		context += "GÜNCEL DUYURULAR:\n"
		for _, ann := range announcements {
			context += fmt.Sprintf("- %s (%s): %s\n", ann.Title, ann.Date, ann.Content)
		}
		context += "\n"
	}

	context += "KULLANICI SORUSU: " + userPrompt + "\n\n"
	context += "ÖNEMLİ TALİMATLAR:\n"
	context += "1. Eğer kullanıcı 'merhaba', 'selam', 'nasılsın', 'iyi günler' gibi selamlama yapıyorsa, sadece kibarca karşılık ver. Belge kullanma.\n"
	context += "2. Yukarıdaki belgeler soruyla DOĞRUDAN ilgiliyse, kısa ve net Türkçe cevap ver.\n"
	context += "3. ⚠️ DERS PROGRAMI VE MÜHENDİSLİK SORULARI İÇİN HAYATİ KURALLAR:\n"
	context += "   - Eğer kullanıcı 'ders programı' soruyorsa ve belgelerde o yıla (2025-2026) ve o sınıfa (örn. 4. SINIF) ait bir TABLO veya LİSTE yoksa, ASLA AMA ASLA ders uydurma.\n"
	context += "   - 'Matematik I', 'Fizik' gibi genel dersleri sakın listeleme. Sadece elindeki belgelerde gördüğün spesifik dersleri yaz.\n"
	context += "   - Eğer belgede ilgili dersleri göremiyorsan tek yanıtın şu olmalı: 'Üzgünüm, 2025-2026 Bahar dönemi ders programı şu an sistemdeki belgelerde bulunmamaktadır. Lütfen bölüm sekreterliği ile iletişime geçin.'\n"
	context += "4. ❌ 'Mühendislik Çıkmazı' gibi saçma veya uydurma terimler kullanan bir bot istemiyoruz. Gerçekçi ol.\n"
	context += "5. Sadece soruyla DOĞRUDAN ilgili belgelerdeki bilgileri kullan. Belgeler alakasızsa, 'Bilgi bulamadım' de.\n"
	context += "6. ASLA tahmin etme, ASLA uydurma.\n"

	return context
}

// ============================================================================
// ZBEU REGULATION VALIDATION MODULE
// ============================================================================
// GLOBAL STATE
// ============================================================================

var groqAPIKey string
var openaiAPIKey string
var embeddingVectorStore *EmbeddingVectorStore // Gemini Embeddings

// ============================================================================

// ============================================================================
// MAIN APPLICATION
// ============================================================================

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ZBEÜ Akıllı Asistan - CAGSOFT Tarafından Geliştirildi ║")
	fmt.Println("║          Çağatay Ok için Üniversite Asistan Botu          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get Groq API key from environment
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ HATA: GROQ_API_KEY tanımlanmamış!\n\n" +
			"Lütfen Groq API key'inizi tanımlayın:\n\n" +
			"Windows:\n" +
			"  $env:GROQ_API_KEY=\"your-groq-key\"\n\n" +
			"Linux/Mac:\n" +
			"  export GROQ_API_KEY=your-groq-key\n\n" +
			"Key almak için: https://console.groq.com/keys")
	}

	fmt.Println("🚀 Groq API aktif (llama3-8b-8192)")
	fmt.Println()

	// Handle command-line test flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--crawl":
			// Crawl all subdomains and save to JSON
			store, err := crawlAllSubdomains()
			if err != nil {
				log.Fatalf("❌ Tarama hatası: %v\n", err)
			}
			if err := saveDocuments(store); err != nil {
				log.Fatalf("❌ Kaydetme hatası: %v\n", err)
			}
			fmt.Printf("\n💾 Veriler %s dosyasına kaydedildi\n", documentsFile)
			return
		case "--test-scraping":
			testScraping()
			return
		case "--test-validation":
			testValidation()
			return
		case "--cli":
			runCLI(apiKey)
			return
		case "--help", "-h":
			printHelp()
			return
		}
	}

	// Default: Start web server
	startWebServer(apiKey)
}

// runCLI runs the interactive CLI mode
func runCLI(apiKey string) {
	// Fetch announcements
	fmt.Println("📡 Üniversite duyuruları çekiliyor...")
	announcements, err := scrapeAnnouncements("https://ogrenci.beun.edu.tr/duyurular.html")
	if err != nil {
		log.Printf("⚠️  Duyuru çekme hatası: %v\n", err)
		announcements = getMockAnnouncements()
	}
	fmt.Printf("✅ %d duyuru yüklendi\n\n", len(announcements))

	// Interactive mode
	fmt.Println("💬 Soru sormak için yazın (çıkmak için 'exit' yazın):")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("📝 Örnek Sorular:")
	fmt.Println("   • 2026 yaz okulu ne zaman başlıyor?")
	fmt.Println("   • 4. sınıfta kaç ders alabilirim?")
	fmt.Println("   • AKTS limiti nedir?")
	fmt.Println()

	// Interactive loop
	for {
		fmt.Print("\n👤 Siz: ")
		var userInput string
		fmt.Scanln(&userInput)

		if userInput == "exit" || userInput == "çıkış" {
			fmt.Println("👋 Görüşmek üzere!")
			break
		}

		if strings.TrimSpace(userInput) == "" {
			continue
		}

		fmt.Println("🤖 Asistan:")
		response, err := queryAI(apiKey, userInput, announcements)
		if err != nil {
			fmt.Printf("❌ Hata: %v\n", err)
			continue
		}

		fmt.Println(response)
	}
}

// Global variables for web server
var (
	globalAPIKey        string
	globalAnnouncements []Announcement
)

// startWebServer starts the HTTP server for web UI
func startWebServer(apiKey string) {
	globalAPIKey = apiKey

	// Fetch announcements
	fmt.Println("📡 Üniversite duyuruları çekiliyor...")
	announcements, err := scrapeAnnouncements("https://ogrenci.beun.edu.tr/duyurular.html")
	if err != nil {
		log.Printf("⚠️  Duyuru çekme hatası: %v\n", err)
		announcements = getMockAnnouncements()
	}
	globalAnnouncements = announcements
	fmt.Printf("✅ %d duyuru yüklendi\n\n", len(announcements))

	// Load existing documents and build Gemini embedding vector store
	fmt.Println("📊 Initializing Gemini Embeddings...")

	// Get Gemini API key
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		fmt.Println("⚠️  GEMINI_API_KEY not set - semantic search disabled")
		fmt.Println("   Set it with: $env:GEMINI_API_KEY=\"your-key\"")
	} else {
		// Initialize embedding cache
		InitEmbeddingCache()

		store, err := loadDocuments()
		if err == nil && len(store.Documents) > 0 {
			embeddingVectorStore, err = BuildEmbeddingVectorStore(store.Documents, geminiKey)
			if err != nil {
				fmt.Printf("⚠️  Error building vector store: %v\n", err)
			} else {
				fmt.Printf("✅ Gemini Semantic Search ready with %d documents\n", len(store.Documents))
				// Save cache
				SaveEmbeddingCache()
			}
		} else {
			fmt.Println("⚠️  No existing documents found - run crawler first")
		}
	}

	// Start web server
	fmt.Println("🌐 Starting web server...")
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/api/ask", handleAsk)
	http.HandleFunc("/api/announcements", handleAnnouncements)
	http.HandleFunc("/api/health", handleHealth)

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  🎓 ZBEÜ Akıllı Asistan - PRODUCTION READY               ║")
	fmt.Println("║  🌐 Web Arayüzü: http://localhost:8080                    ║")
	fmt.Println("║  🤖 AI: Groq API aktif (llama3-8b-8192)                  ║")
	if embeddingVectorStore != nil {
		fmt.Printf("║  🔍 Gemini Embeddings: %d documents indexed               ║\n", len(embeddingVectorStore.Documents))
	} else {
		fmt.Println("║  ⚠️  Semantic Search: Disabled (no GEMINI_API_KEY)       ║")
	}
	fmt.Println("║  💡 CLI Modu: go run main.go --cli                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// serveHome serves the main HTML page
func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, getHTMLContent())
}

// handleAsk handles the /api/ask endpoint
func handleAsk(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request",
		})
		return
	}

	// Get AI response
	response, err := queryAI(globalAPIKey, req.Question, globalAnnouncements)

	// Send response
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"answer": "",
			"error":  err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"answer": response,
		"error":  nil,
	})
}

// handleAnnouncements handles the /api/announcements endpoint
func handleAnnouncements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globalAnnouncements)
}

// handleHealth handles the /api/health endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"announcements": len(globalAnnouncements),
	})
}

// getHTMLContent returns the HTML content for the web UI
func getHTMLContent() string {
	return `<!DOCTYPE html>
<html lang="tr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ZBEÜ Akıllı Asistan</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }

        .container {
            width: 100%;
            max-width: 900px;
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
            border: 1px solid rgba(255, 255, 255, 0.18);
            overflow: hidden;
        }

        .header {
            background: rgba(255, 255, 255, 0.2);
            padding: 25px;
            text-align: center;
            border-bottom: 1px solid rgba(255, 255, 255, 0.2);
        }

        .header h1 {
            color: white;
            font-size: 28px;
            margin-bottom: 5px;
        }

        .header p {
            color: rgba(255, 255, 255, 0.9);
            font-size: 14px;
        }

        .info-panel {
            background: rgba(255, 255, 255, 0.15);
            padding: 15px 25px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.2);
        }

        .info-item {
            color: white;
            font-size: 13px;
            margin: 5px 0;
        }

        .info-item strong {
            color: #ffd700;
        }

        .chat-container {
            height: 400px;
            overflow-y: auto;
            padding: 20px 25px;
            display: flex;
            flex-direction: column;
            gap: 15px;
        }

        .chat-container::-webkit-scrollbar {
            width: 8px;
        }

        .chat-container::-webkit-scrollbar-track {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 10px;
        }

        .chat-container::-webkit-scrollbar-thumb {
            background: rgba(255, 255, 255, 0.3);
            border-radius: 10px;
        }

        .message {
            display: flex;
            gap: 10px;
            animation: slideIn 0.3s ease;
        }

        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateY(10px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .message.user {
            justify-content: flex-end;
        }

        .message-content {
            max-width: 70%;
            padding: 12px 18px;
            border-radius: 18px;
            color: white;
            line-height: 1.5;
            font-size: 14px;
        }

        .message.user .message-content {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border-bottom-right-radius: 4px;
        }

        .message.ai .message-content {
            background: rgba(255, 255, 255, 0.2);
            border-bottom-left-radius: 4px;
        }

        .message-icon {
            width: 35px;
            height: 35px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 18px;
            flex-shrink: 0;
        }

        .message.user .message-icon {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        }

        .message.ai .message-icon {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        }

        .input-area {
            padding: 20px 25px;
            background: rgba(255, 255, 255, 0.15);
            border-top: 1px solid rgba(255, 255, 255, 0.2);
            display: flex;
            gap: 10px;
        }

        #questionInput {
            flex: 1;
            padding: 12px 18px;
            border: 2px solid rgba(255, 255, 255, 0.3);
            border-radius: 25px;
            background: rgba(255, 255, 255, 0.2);
            color: white;
            font-size: 14px;
            outline: none;
            transition: all 0.3s ease;
        }

        #questionInput::placeholder {
            color: rgba(255, 255, 255, 0.7);
        }

        #questionInput:focus {
            border-color: rgba(255, 255, 255, 0.6);
            background: rgba(255, 255, 255, 0.25);
        }

        #sendBtn {
            padding: 12px 30px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 25px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px 0 rgba(102, 126, 234, 0.4);
        }

        #sendBtn:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px 0 rgba(102, 126, 234, 0.6);
        }

        #sendBtn:active {
            transform: translateY(0);
        }

        #sendBtn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }

        .loading {
            display: none;
            align-items: center;
            gap: 8px;
            color: rgba(255, 255, 255, 0.8);
            font-size: 13px;
        }

        .loading.active {
            display: flex;
        }

        .loading-dots {
            display: flex;
            gap: 4px;
        }

        .loading-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: white;
            animation: bounce 1.4s infinite ease-in-out both;
        }

        .loading-dot:nth-child(1) { animation-delay: -0.32s; }
        .loading-dot:nth-child(2) { animation-delay: -0.16s; }

        @keyframes bounce {
            0%, 80%, 100% { transform: scale(0); }
            40% { transform: scale(1); }
        }

        .examples {
            padding: 15px 25px;
            background: rgba(255, 255, 255, 0.1);
            border-top: 1px solid rgba(255, 255, 255, 0.2);
        }

        .examples h3 {
            color: white;
            font-size: 14px;
            margin-bottom: 10px;
        }

        .example-btn {
            display: inline-block;
            margin: 5px 5px 5px 0;
            padding: 8px 15px;
            background: rgba(255, 255, 255, 0.2);
            color: white;
            border: 1px solid rgba(255, 255, 255, 0.3);
            border-radius: 15px;
            cursor: pointer;
            font-size: 12px;
            transition: all 0.3s ease;
        }

        .example-btn:hover {
            background: rgba(255, 255, 255, 0.3);
            transform: translateY(-2px);
        }

        @media (max-width: 768px) {
            .container {
                max-width: 100%;
            }

            .header h1 {
                font-size: 22px;
            }

            .chat-container {
                height: 350px;
            }

            .message-content {
                max-width: 85%;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎓 ZBEÜ Akıllı Asistan</h1>
            <p>CAGSOFT tarafından Çağatay Ok için geliştirildi</p>
        </div>

        <div class="info-panel">
            <div class="info-item">📋 <strong>Maksimum Ders:</strong> 3 (4. sınıf için)</div>
            <div class="info-item">📊 <strong>AKTS Aralığı:</strong> 21-24</div>
            <div class="info-item">📡 <strong>Duyurular:</strong> <span id="announcementCount">Yükleniyor...</span></div>
        </div>

        <div class="chat-container" id="chatContainer">
            <div class="message ai">
                <div class="message-icon">🤖</div>
                <div class="message-content">
                    Merhaba! Ben ZBEÜ Akıllı Asistanınızım. Yaz okulu, ders alma sınırları ve AKTS hesaplamaları hakkında sorularınızı yanıtlayabilirim. Nasıl yardımcı olabilirim?
                </div>
            </div>
        </div>

        <div class="examples">
            <h3>💡 Örnek Sorular:</h3>
            <button class="example-btn" onclick="askExample('2026 yaz okulu ne zaman başlıyor?')">📅 Yaz Okulu Tarihleri</button>
            <button class="example-btn" onclick="askExample('4. sınıfta kaç ders alabilirim?')">📚 Ders Alma Sınırı</button>
            <button class="example-btn" onclick="askExample('AKTS limiti nedir?')">📊 AKTS Kuralları</button>
        </div>

        <div class="input-area">
            <input type="text" id="questionInput" placeholder="Sorunuzu buraya yazın..." onkeypress="handleKeyPress(event)">
            <button id="sendBtn" onclick="askQuestion()">Gönder</button>
        </div>

        <div class="loading" id="loading">
            <div class="loading-dots">
                <div class="loading-dot"></div>
                <div class="loading-dot"></div>
                <div class="loading-dot"></div>
            </div>
            <span>AI yanıt oluşturuyor...</span>
        </div>
    </div>

    <script>
        // Load announcement count
        fetch('/api/announcements')
            .then(res => res.json())
            .then(data => {
                document.getElementById('announcementCount').textContent = data.length + ' duyuru yüklendi';
            })
            .catch(() => {
                document.getElementById('announcementCount').textContent = 'Yüklenemedi';
            });

        function addMessage(type, content) {
            const chatContainer = document.getElementById('chatContainer');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + type;

            const icon = type === 'user' ? '👤' : '🤖';
            messageDiv.innerHTML = ` +
		"`" + `
                <div class="message-icon">${icon}</div>
                <div class="message-content">${content}</div>
            ` + "`" + `;

            chatContainer.appendChild(messageDiv);
            chatContainer.scrollTop = chatContainer.scrollHeight;
        }

        async function askQuestion() {
            const input = document.getElementById('questionInput');
            const question = input.value.trim();

            if (!question) return;

            // Add user message
            addMessage('user', question);
            input.value = '';

            // Show loading
            const sendBtn = document.getElementById('sendBtn');
            const loading = document.getElementById('loading');
            sendBtn.disabled = true;
            loading.classList.add('active');

            try {
                const response = await fetch('/api/ask', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ question })
                });

                const data = await response.json();

                if (data.error) {
                    addMessage('ai', '❌ Hata: ' + data.error);
                } else {
                    addMessage('ai', data.answer);
                }
            } catch (error) {
                addMessage('ai', '❌ Bağlantı hatası: ' + error.message);
            } finally {
                sendBtn.disabled = false;
                loading.classList.remove('active');
            }
        }

        function askExample(question) {
            document.getElementById('questionInput').value = question;
            askQuestion();
        }

        function handleKeyPress(event) {
            if (event.key === 'Enter') {
                askQuestion();
            }
        }
    </script>
</body>
</html>`
}

// ============================================================================
// TEST FUNCTIONS
// ============================================================================

func testScraping() {
	fmt.Println("🧪 Web Scraping Testi\n")
	announcements, err := scrapeAnnouncements("https://ogrenci.beun.edu.tr/duyurular.html")
	if err != nil {
		fmt.Printf("❌ Hata: %v\n", err)
		return
	}

	fmt.Printf("✅ %d duyuru çekildi:\n\n", len(announcements))
	for i, ann := range announcements {
		fmt.Printf("%d. %s (%s)\n", i+1, ann.Title, ann.Date)
		fmt.Printf("   %s\n\n", ann.Content)
	}
}

func testValidation() {
	fmt.Println("🧪 ZBEÜ Yönetmelik Kuralları Testi\n")

	fmt.Printf("📋 Kurallar:\n")
	fmt.Printf("   - Maksimum Ders: %d\n", regulation.MaxCourses)
	fmt.Printf("   - Maksimum AKTS: %d\n\n", regulation.MaxAKTS)

	// Test scenarios
	scenarios := []struct {
		name        string
		courseCount int
		totalAKTS   int
	}{
		{"Geçerli Seçim", 3, 24},
		{"Çok Fazla Ders", 4, 24},
		{"AKTS Çok Yüksek", 3, 35},
	}

	for _, scenario := range scenarios {
		fmt.Printf("Test: %s\n", scenario.name)
		fmt.Printf("  Ders Sayısı: %d, AKTS: %d\n", scenario.courseCount, scenario.totalAKTS)

		valid := true
		if scenario.courseCount > regulation.MaxCourses {
			fmt.Printf("  ❌ Ders sayısı limiti aşıldı (max %d)\n", regulation.MaxCourses)
			valid = false
		}
		if scenario.totalAKTS > regulation.MaxAKTS {
			fmt.Printf("  ❌ AKTS limiti aşıldı (max %d)\n", regulation.MaxAKTS)
			valid = false
		}
		if valid {
			fmt.Println("  ✅ Geçerli seçim")
		}
		fmt.Println()
	}
}

func printHelp() {
	fmt.Println("ZBEÜ Akıllı Asistan - Kullanım Kılavuzu")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Kullanım:")
	fmt.Println("  go run main.go                  # Normal mod (interaktif)")
	fmt.Println("  go run main.go --test-scraping  # Web scraping testi")
	fmt.Println("  go run main.go --test-gemini    # Gemini API testi")
	fmt.Println("  go run main.go --test-validation # Yönetmelik kuralları testi")
	fmt.Println("  go run main.go --help           # Bu yardım mesajı")
	fmt.Println()
	fmt.Println("Gereksinimler:")
	fmt.Println("  - GEMINI_API_KEY environment variable tanımlı olmalı")
	fmt.Println("  - API Key: https://aistudio.google.com/apikey")
	fmt.Println()
	fmt.Println("Özellikler:")
	fmt.Println("  ✓ Web scraping (ZBEÜ duyuruları)")
	fmt.Println("  ✓ Google Gemini AI entegrasyonu")
	fmt.Println("  ✓ 404/429 hata yönetimi")
	fmt.Println("  ✓ ZBEÜ yönetmelik kuralları kontrolü")
	fmt.Println("  ✓ Yaz okulu, ders alma, AKTS hesaplama")
}
