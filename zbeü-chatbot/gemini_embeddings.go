package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
)

// Gemini Embedding API integration
// Model: text-embedding-004 (768 dimensions, multilingual, supports Turkish)

const geminiEmbeddingURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent"

type GeminiEmbedRequest struct {
	Model   string             `json:"model"`
	Content GeminiEmbedContent `json:"content"`
}

type GeminiEmbedContent struct {
	Parts []GeminiEmbedPart `json:"parts"`
}

type GeminiEmbedPart struct {
	Text string `json:"text"`
}

type GeminiEmbedResponse struct {
	Embedding GeminiEmbedding   `json:"embedding"`
	Error     *GeminiEmbedError `json:"error,omitempty"`
}

type GeminiEmbedding struct {
	Values []float64 `json:"values"`
}

type GeminiEmbedError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// CreateGeminiEmbedding creates a vector embedding using Gemini API
// Uses text-embedding-004 model (768 dimensions, multilingual)
func CreateGeminiEmbedding(text string, apiKey string) ([]float64, error) {
	// Gemini embedding API: model is specified in URL, not in body
	reqBody := map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []map[string]string{
				{"text": text},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Use the correct endpoint with model in URL
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent?key=%s", apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Gemini Embedding API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var embResp GeminiEmbedResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, err
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s", embResp.Error.Message)
	}

	if len(embResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding returned from Gemini")
	}

	return embResp.Embedding.Values, nil
}

// CosineSimilarityGemini calculates cosine similarity between two float64 vectors
func CosineSimilarityGemini(v1, v2 []float64) float64 {
	if len(v1) != len(v2) {
		return 0
	}

	dotProduct := 0.0
	mag1 := 0.0
	mag2 := 0.0

	for i := 0; i < len(v1); i++ {
		dotProduct += v1[i] * v2[i]
		mag1 += v1[i] * v1[i]
		mag2 += v2[i] * v2[i]
	}

	mag1 = math.Sqrt(mag1)
	mag2 = math.Sqrt(mag2)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	return dotProduct / (mag1 * mag2)
}
