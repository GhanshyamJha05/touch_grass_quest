package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	tasks    *TaskManager
	verifier Verifier
	limiter  *RateLimiter
	staticFS fs.FS
}

func NewServer(tasks *TaskManager, verifier Verifier, limiter *RateLimiter, staticFS fs.FS) *Server {
	return &Server{
		tasks:    tasks,
		verifier: verifier,
		limiter:  limiter,
		staticFS: staticFS,
	}
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.ServeMux{}

	// Static files
	mux.Handle("/", http.FileServer(http.FS(s.staticFS)))

	// API
	mux.HandleFunc("/api/task", s.handleGetTask)
	mux.HandleFunc("/api/check", s.handleCheck)

	return &mux
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	task := s.tasks.GetTaskForDate(GetTodayDateStr())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

type CheckResponse struct {
	Success bool   `json:"success"` // true if passed
	Message string `json:"message"` // The message to show to the user
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := GetIP(r)
	if !s.limiter.Allow(ip) {
		http.Error(w, "Too many requests, try again later.", http.StatusTooManyRequests)
		return
	}

	// 10 MB limit for safety, though frontend downscales
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large or bad request", http.StatusBadRequest)
		return
	}

	taskID := r.FormValue("taskId")
	task, ok := s.tasks.GetTaskByID(taskID)
	if !ok {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Missing photo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imgData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading photo", http.StatusInternalServerError)
		return
	}

	mimeType := http.DetectContentType(imgData)
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
		// DetectContentType might return "application/octet-stream" for webp sometimes in older go versions, 
		// but let's do a basic webp signature check if it fails
		if len(imgData) > 12 && string(imgData[0:4]) == "RIFF" && string(imgData[8:12]) == "WEBP" {
			mimeType = "image/webp"
		} else {
			http.Error(w, "Invalid image format. Only JPEG, PNG, or WebP are allowed.", http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var verdict Verdict
	var verifyErr error

	// Retry logic for bad JSON / temporary failures
	for i := 0; i < 2; i++ {
		verdict, verifyErr = s.verifier.Verify(ctx, imgData, mimeType, task.Text)
		if verifyErr == nil {
			break
		}
		log.Printf("Verification attempt %d failed: %v", i+1, verifyErr)
		time.Sleep(500 * time.Millisecond) // small backoff
	}

	if verifyErr != nil {
		sendCheckResponse(w, false, "Couldn't check that, try again.")
		return
	}

	if verdict.Match && verdict.Outdoors {
		sendCheckResponse(w, true, fmt.Sprintf("Pass! %s", verdict.Reason))
	} else if verdict.Match && !verdict.Outdoors {
		sendCheckResponse(w, false, "Looks like it's indoors or a screen, take it outside. " + verdict.Reason)
	} else {
		reason := verdict.Reason
		if strings.HasPrefix(strings.ToLower(reason), "not quite.") {
			reason = strings.TrimSpace(reason[10:])
		}
		sendCheckResponse(w, false, fmt.Sprintf("Not quite. %s", reason))
	}
}

func sendCheckResponse(w http.ResponseWriter, success bool, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CheckResponse{
		Success: success,
		Message: msg,
	})
}
