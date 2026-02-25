package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// EmbeddingVectorStore holds documents with Gemini embeddings
type EmbeddingVectorStore struct {
	Documents []EmbeddingDocument
	APIKey    string
}

type EmbeddingDocument struct {
	Doc       Document
	Embedding []float64
}

type EmbeddingSearchResult struct {
	Doc   Document
	Score float64
}

// Embedding cache to avoid redundant API calls
type EmbeddingCache struct {
	Embeddings map[string]CachedEmbedding
	FilePath   string
}

type CachedEmbedding struct {
	Text      string    `json:"text"`
	Embedding []float64 `json:"embedding"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

var embeddingCache *EmbeddingCache

// InitEmbeddingCache initializes the embedding cache
func InitEmbeddingCache() {
	// Create data directory if it doesn't exist
	os.MkdirAll("data", 0755)

	embeddingCache = &EmbeddingCache{
		Embeddings: make(map[string]CachedEmbedding),
		FilePath:   "data/embedding_cache.json",
	}

	// Load existing cache
	data, err := os.ReadFile(embeddingCache.FilePath)
	if err == nil {
		var cached map[string]CachedEmbedding
		if json.Unmarshal(data, &cached) == nil {
			embeddingCache.Embeddings = cached
			fmt.Printf("📦 Loaded %d cached embeddings\n", len(cached))
		}
	}
}

// SaveEmbeddingCache saves the cache to disk
func SaveEmbeddingCache() error {
	data, err := json.MarshalIndent(embeddingCache.Embeddings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(embeddingCache.FilePath, data, 0644)
}

// GetCachedEmbedding retrieves embedding from cache or creates new one
func GetCachedEmbedding(text string, apiKey string) ([]float64, error) {
	// Create hash of text
	hash := md5.Sum([]byte(text))
	hashStr := hex.EncodeToString(hash[:])

	// Check cache
	if cached, ok := embeddingCache.Embeddings[hashStr]; ok {
		return cached.Embedding, nil
	}

	// Create new embedding via Gemini API
	fmt.Printf("🔄 Creating Gemini embedding for: %s...\n", text[:min(50, len(text))])
	embedding, err := CreateGeminiEmbedding(text, apiKey)
	if err != nil {
		return nil, err
	}

	// Cache it
	embeddingCache.Embeddings[hashStr] = CachedEmbedding{
		Text:      text[:min(100, len(text))], // Store truncated text for reference
		Embedding: embedding,
		Hash:      hashStr,
		CreatedAt: time.Now(),
	}

	// Save cache periodically (every 10 new embeddings)
	if len(embeddingCache.Embeddings)%10 == 0 {
		SaveEmbeddingCache()
	}

	return embedding, nil
}

// BuildEmbeddingVectorStore creates embeddings for all documents
func BuildEmbeddingVectorStore(docs []Document, apiKey string) (*EmbeddingVectorStore, error) {
	store := &EmbeddingVectorStore{
		Documents: make([]EmbeddingDocument, 0),
		APIKey:    apiKey,
	}

	fmt.Printf("🚀 Building embedding vector store for %d documents...\n", len(docs))

	for i, doc := range docs {
		// Create embedding for document (title + content)
		text := doc.Title + "\n" + doc.Content
		if len(text) > 8000 { // OpenAI limit
			text = text[:8000]
		}

		embedding, err := GetCachedEmbedding(text, apiKey)
		if err != nil {
			fmt.Printf("⚠️  Error creating embedding for doc %d: %v\n", i, err)
			continue
		}

		store.Documents = append(store.Documents, EmbeddingDocument{
			Doc:       doc,
			Embedding: embedding,
		})

		if (i+1)%5 == 0 {
			fmt.Printf("  ✅ Processed %d/%d documents\n", i+1, len(docs))
		}
	}

	fmt.Printf("✅ Vector store ready with %d documents\n", len(store.Documents))
	return store, nil
}

// SemanticSearchWithEmbeddings finds most relevant documents using OpenAI embeddings
func (vs *EmbeddingVectorStore) SemanticSearchWithEmbeddings(query string, topK int) ([]Document, error) {
	// Create embedding for query
	queryEmbedding, err := GetCachedEmbedding(query, vs.APIKey)
	if err != nil {
		return nil, err
	}

	// Calculate cosine similarity for each document
	results := make([]EmbeddingSearchResult, 0)
	for _, docEmb := range vs.Documents {
		similarity := CosineSimilarityGemini(queryEmbedding, docEmb.Embedding)

		// Boost score if query contains class number and doc matches
		if containsClassNumber(query, 4) {
			contentLower := toLower(docEmb.Doc.Content)
			if contains(contentLower, "4. sinif") || contains(contentLower, "4.sinif") {
				similarity *= 1.5 // Boost for correct class
			}
		}

		results = append(results, EmbeddingSearchResult{
			Doc:   docEmb.Doc,
			Score: similarity,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K
	docs := make([]Document, 0)
	for i := 0; i < topK && i < len(results); i++ {
		if results[i].Score > 0.6 { // Higher threshold - only truly relevant docs
			docs = append(docs, results[i].Doc)
		}
	}

	return docs, nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func containsClassNumber(text string, classNum int) bool {
	return contains(toLower(text), fmt.Sprintf("%d.", classNum)) ||
		contains(toLower(text), fmt.Sprintf("%d. sınıf", classNum))
}

func toLower(s string) string {
	// Simple Turkish-aware lowercase (basic version)
	return s // For now, use standard lowercase
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
