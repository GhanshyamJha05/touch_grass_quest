package main

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"touch-grass-quest/app"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed tasks.json
var tasksJSON []byte

var mockState int

type CycleFakeVerifier struct{}

func (f *CycleFakeVerifier) Verify(ctx context.Context, image []byte, mimeType string, taskText string) (app.Verdict, error) {
	mockState++
	if mockState%3 == 1 {
		return app.Verdict{Match: false, Outdoors: true, Reason: "Not quite. This doesn't look like the requested item."}, nil
	} else if mockState%3 == 2 {
		return app.Verdict{Match: true, Outdoors: false, Reason: "I see it, but I can also see laptop bezels."}, nil
	}
	return app.Verdict{Match: true, Outdoors: true, Reason: "Perfect match! I clearly see it outdoors."}, nil
}

func main() {
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("LLM_API_KEY is required")
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		log.Fatal("LLM_MODEL is required (e.g. gemini-2.5-pro)")
	}
	visionModel := os.Getenv("VISION_MODEL")
	if visionModel != "" {
		model = visionModel
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Max concurrent calls
	maxConcurrent := 5
	if mc := os.Getenv("MAX_CONCURRENT"); mc != "" {
		if val, err := strconv.Atoi(mc); err == nil {
			maxConcurrent = val
		}
	}

	// Parse tasks
	tm, err := app.NewTaskManager(bytes.NewReader(tasksJSON))
	if err != nil {
		log.Fatalf("Failed to load tasks: %v", err)
	}

	var verifier app.Verifier
	if os.Getenv("USE_MOCK") == "true" {
		verifier = &CycleFakeVerifier{}
		log.Println("Starting in CYCLING MOCK mode (bypassing API) for screenshots")
	} else {
		verifier = app.NewRealVerifier(baseURL, apiKey, model, maxConcurrent)
	}

	limiter := app.NewRateLimiter(10, time.Minute) // 10 requests per minute per IP

	// Setup static files sub fs
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub fs: %v", err)
	}

	server := app.NewServer(tm, verifier, limiter, staticFS)
	mux := server.Routes()

	addr := ":" + port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
