package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Verdict struct {
	Match    bool   `json:"match"`
	Outdoors bool   `json:"outdoors"`
	Reason   string `json:"reason"`
}

type Verifier interface {
	Verify(ctx context.Context, image []byte, mimeType string, taskText string) (Verdict, error)
}

type RealVerifier struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	model      string
	concurrent chan struct{}
}

func NewRealVerifier(baseURL, apiKey, model string, maxConcurrent int) *RealVerifier {
	return &RealVerifier{
		client: &http.Client{Timeout: 30 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/") + "/chat/completions",
		apiKey:  apiKey,
		model:   model,
		concurrent: make(chan struct{}, maxConcurrent),
	}
}

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
}

var thoughtRe = regexp.MustCompile(`(?s)<thought>.*?</thought>`)

func (v *RealVerifier) Verify(ctx context.Context, image []byte, mimeType string, taskText string) (Verdict, error) {
	select {
	case v.concurrent <- struct{}{}:
		defer func() { <-v.concurrent }()
	case <-ctx.Done():
		return Verdict{}, ctx.Err()
	}

	b64Img := base64.StdEncoding.EncodeToString(image)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64Img)

	systemPrompt := `You are an AI assistant judging a photo scavenger hunt. 
The user is given a task and must upload a real photo taken outside matching the task.
You must return a JSON response with exactly three fields:
- "match": boolean (true if the photo clearly shows the requested task, false otherwise)
- "outdoors": boolean (true if the photo was clearly taken outdoors, false if it is indoors or a photo of a screen/printed picture)
- "reason": string (a short explanation, <=20 words, of why you gave this verdict)

Be fair but honest, accept reasonable interpretations, do not invent objects. Treat a photo of a screen or printed picture as not outdoors. 
Return strictly JSON. No markdown formatting, no other text.`

	userPrompt := fmt.Sprintf("Task: %s\nDoes this image match the task and is it outdoors?", taskText)

	reqBody := ChatRequest{
		Model:       v.model,
		Temperature: 0.0,
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: systemPrompt,
			},
			{
				Role: "user",
				Content: []map[string]interface{}{
					{
						"type": "text",
						"text": userPrompt,
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
		return Verdict{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", v.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return Verdict{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return Verdict{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return Verdict{}, fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return Verdict{}, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return Verdict{}, errors.New("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content

	// Strip thoughts
	if strings.Contains(content, "<thought>") {
		if !strings.Contains(content, "</thought>") {
			return Verdict{}, errors.New("unclosed thought tag")
		}
		content = thoughtRe.ReplaceAllString(content, "")
	}
	// Also clean up any markdown JSON blocks if the model ignored instructions
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	}
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}

	var verdict Verdict
	if err := json.Unmarshal([]byte(content), &verdict); err != nil {
		return Verdict{}, fmt.Errorf("invalid json verdict: %w, content: %s", err, content)
	}

	// Validate reason length (rough approximation by word split)
	words := strings.Fields(verdict.Reason)
	if len(words) > 30 { // allowing slight leeway from 20
		verdict.Reason = strings.Join(words[:30], " ") + "..."
	}

	return verdict, nil
}

type FakeVerifier struct {
	mu       sync.Mutex
	Next     Verdict
	NextErr  error
	Called   bool
	LastTask string
}

func (f *FakeVerifier) Verify(ctx context.Context, image []byte, mimeType string, taskText string) (Verdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Called = true
	f.LastTask = taskText
	return f.Next, f.NextErr
}
