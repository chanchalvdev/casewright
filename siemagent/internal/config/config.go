package config

import "os"

type Config struct {
	BaseURL       string // Kimchi Inference OpenAI-compatible endpoint
	APIKey        string
	ModelName     string
	Workers       int
	Port          string
	QdrantAddr    string // gRPC address for Qdrant (host:port)
	OllamaURL     string // Ollama base URL for embeddings
	AllowedOrigin string // CORS allowed origin
}

func Load() Config {
	baseURL := os.Getenv("KIMCHI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.kimchi.ai/v1"
	}

	apiKey := os.Getenv("KIMCHI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	model := os.Getenv("SIEM_MODEL")
	if model == "" {
		model = "kimi-k2-5"
	}

	port := os.Getenv("CONDUCTOR_PORT")
	if port == "" {
		port = "8080"
	}

	qdrantAddr := os.Getenv("QDRANT_ADDR")
	if qdrantAddr == "" {
		qdrantAddr = "localhost:6334"
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}

	return Config{
		BaseURL:       baseURL,
		APIKey:        apiKey,
		ModelName:     model,
		Workers:       5,
		Port:          port,
		QdrantAddr:    qdrantAddr,
		OllamaURL:     ollamaURL,
		AllowedOrigin: allowedOrigin,
	}
}
