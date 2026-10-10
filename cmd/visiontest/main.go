package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: visiontest <image_path>")
		os.Exit(1)
	}
	imgPath := os.Args[1]

	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	apiURL := strings.TrimRight(baseURL, "/") + "/chat/completions"

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		fmt.Println("LLM_API_KEY is required")
		os.Exit(1)
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gemini-2.5-pro" 
	}
	visionModel := os.Getenv("VISION_MODEL")
	if visionModel != "" {
		model = visionModel
	}

	imgData, err := os.ReadFile(imgPath)
	if err != nil {
		fmt.Printf("Error reading image: %v\n", err)
		os.Exit(1)
	}

	ext := strings.ToLower(filepath.Ext(imgPath))
	mimeType := "image/jpeg"
	if ext == ".png" {
		mimeType = "image/png"
	} else if ext == ".webp" {
		mimeType = "image/webp"
	}
	b64Img := base64.StdEncoding.EncodeToString(imgData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64Img)

	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role: "user",
				Content: []map[string]interface{}{
					{
						"type": "text",
						"text": "What is in this image?",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": dataURL,
						},
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		os.Exit(1)
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(respBody))
}
