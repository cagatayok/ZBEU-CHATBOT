package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAI Embedding API integration
type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// CreateEmbedding creates a vector embedding using OpenAI API
func CreateEmbedding(text string, apiKey string) ([]float64, error) {
	url := "https://api.openai.com/v1/embeddings"

	reqBody := OpenAIEmbeddingRequest{
		Input: text,
		Model: "text-embedding-3-small", // 1536 dimensions, $0.02/1M tokens
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var embResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, err
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embResp.Data[0].Embedding, nil
}

// CosineSimilarityFloat64 calculates cosine similarity between two float64 vectors
func CosineSimilarityFloat64(v1, v2 []float64) float64 {
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

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	return dotProduct / (mag1 * mag2)
}
