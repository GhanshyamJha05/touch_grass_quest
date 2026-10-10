package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"touch-grass-quest/app"
)

type ManifestEntry struct {
	Image    string `json:"image"`
	TaskID   string `json:"taskId"`
	Expected bool   `json:"expected"`
}

func main() {
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("LLM_API_KEY is required to run the evaluation")
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gemini-2.5-pro"
	}
	visionModel := os.Getenv("VISION_MODEL")
	if visionModel != "" {
		model = visionModel
	}

	manifestPath := filepath.Join("testdata", "eval", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Fatalf("Failed to read manifest: %v", err)
	}

	var entries []ManifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Fatalf("Failed to parse manifest: %v", err)
	}

	tasksJSON, err := os.ReadFile("tasks.json")
	if err != nil {
		log.Fatalf("Failed to read tasks.json: %v", err)
	}

	var tasks []app.Task
	if err := json.Unmarshal(tasksJSON, &tasks); err != nil {
		log.Fatalf("Failed to parse tasks: %v", err)
	}

	taskMap := make(map[string]string)
	for _, t := range tasks {
		taskMap[t.ID] = t.Text
	}

	verifier := app.NewRealVerifier(baseURL, apiKey, model, 1)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Image\tTask\tExpected\tActual\tReason\tPass/Fail\tLatency")
	fmt.Fprintln(w, "-----\t----\t--------\t------\t------\t---------\t-------")

	var latencies []time.Duration
	var slowest time.Duration
	
	for _, entry := range entries {
		imgPath := filepath.Join("testdata", "eval", entry.Image)
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			log.Printf("Skip %s: %v", entry.Image, err)
			continue
		}

		mimeType := http.DetectContentType(imgData)
		if !strings.HasPrefix(mimeType, "image/") {
			mimeType = "image/jpeg"
		}

		taskText := taskMap[entry.TaskID]
		if taskText == "" {
			taskText = "Unknown task"
		}

		start := time.Now()
		ctx := context.Background()
		verdict, err := verifier.Verify(ctx, imgData, mimeType, taskText)
		latency := time.Since(start)
		latencies = append(latencies, latency)
		if latency > slowest {
			slowest = latency
		}

		actual := verdict.Match && verdict.Outdoors
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\t%v\tERROR\t%v\tFAIL\t%v\n", entry.Image, entry.TaskID, entry.Expected, err, latency)
			continue
		}

		passFail := "FAIL"
		if actual == entry.Expected {
			passFail = "PASS"
		}

		fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%s\t%s\t%v\n", 
			entry.Image, entry.TaskID, entry.Expected, actual, verdict.Reason, passFail, latency)
	}
	w.Flush()

	if len(latencies) > 0 {
		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		avg := total / time.Duration(len(latencies))
		fmt.Printf("\nMedian latency (approx avg): %v\n", avg)
		fmt.Printf("Slowest time: %v\n", slowest)
	}
}
