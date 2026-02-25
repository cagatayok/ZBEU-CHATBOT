package main

import (
	"math"
	"sort"
	"strings"
)

// VectorStore holds documents with TF-IDF vectors for semantic search
type VectorStore struct {
	Documents []DocumentVector
	IDF       map[string]float64
}

type DocumentVector struct {
	Doc    Document
	Vector map[string]float64 // TF-IDF vector
}

type SearchResult struct {
	Doc   Document
	Score float64
}

// BuildVectorStore creates TF-IDF vectors for all documents
func BuildVectorStore(docs []Document) *VectorStore {
	store := &VectorStore{
		Documents: make([]DocumentVector, 0),
		IDF:       make(map[string]float64),
	}

	// Calculate document frequency for each term
	termDocCount := make(map[string]int)
	totalDocs := len(docs)

	for _, doc := range docs {
		terms := tokenize(doc.Title + " " + doc.Content)
		seen := make(map[string]bool)
		for _, term := range terms {
			if !seen[term] {
				termDocCount[term]++
				seen[term] = true
			}
		}
	}

	// Calculate IDF
	for term, count := range termDocCount {
		store.IDF[term] = math.Log(float64(totalDocs) / float64(count))
	}

	// Create TF-IDF vectors for each document
	for _, doc := range docs {
		vector := createTFIDFVector(doc.Title+" "+doc.Content, store.IDF)
		store.Documents = append(store.Documents, DocumentVector{
			Doc:    doc,
			Vector: vector,
		})
	}

	return store
}

// SemanticSearch finds most relevant documents using cosine similarity
func (vs *VectorStore) SemanticSearch(query string, topK int) []Document {
	queryVector := createTFIDFVector(query, vs.IDF)

	// Calculate cosine similarity for each document
	results := make([]SearchResult, 0)
	for _, docVec := range vs.Documents {
		similarity := cosineSimilarity(queryVector, docVec.Vector)

		// Boost score if query contains class number and doc matches
		if strings.Contains(query, "4.") || strings.Contains(query, "4. sınıf") {
			contentLower := strings.ToLower(docVec.Doc.Content)
			if strings.Contains(contentLower, "4. sinif") || strings.Contains(contentLower, "4.sinif") {
				similarity *= 2.0 // Double the score
			}
			// Penalize wrong classes
			if strings.Contains(contentLower, "1. sinif") ||
				strings.Contains(contentLower, "2. sinif") ||
				strings.Contains(contentLower, "3. sinif") {
				similarity *= 0.3 // Reduce score significantly
			}
		}

		results = append(results, SearchResult{
			Doc:   docVec.Doc,
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
		if results[i].Score > 0 { // Only return if there's some similarity
			docs = append(docs, results[i].Doc)
		}
	}

	return docs
}

// tokenize splits text into terms
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// Remove punctuation and split
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		"(", " ", ")", " ", "[", " ", "]", " ",
		":", " ", ";", " ", "-", " ",
	)
	text = replacer.Replace(text)

	words := strings.Fields(text)

	// Filter out very short words
	filtered := make([]string, 0)
	for _, word := range words {
		if len(word) >= 2 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// createTFIDFVector creates TF-IDF vector for text
func createTFIDFVector(text string, idf map[string]float64) map[string]float64 {
	terms := tokenize(text)

	// Calculate term frequency
	tf := make(map[string]float64)
	for _, term := range terms {
		tf[term]++
	}

	// Normalize by document length
	docLength := float64(len(terms))
	for term := range tf {
		tf[term] /= docLength
	}

	// Calculate TF-IDF
	tfidf := make(map[string]float64)
	for term, tfVal := range tf {
		if idfVal, ok := idf[term]; ok {
			tfidf[term] = tfVal * idfVal
		}
	}

	return tfidf
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(v1, v2 map[string]float64) float64 {
	dotProduct := 0.0
	mag1 := 0.0
	mag2 := 0.0

	// Calculate dot product and magnitudes
	for term, val1 := range v1 {
		dotProduct += val1 * v2[term]
		mag1 += val1 * val1
	}

	for _, val2 := range v2 {
		mag2 += val2 * val2
	}

	mag1 = math.Sqrt(mag1)
	mag2 = math.Sqrt(mag2)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	return dotProduct / (mag1 * mag2)
}
