package app

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestServer_CheckUpload(t *testing.T) {
	tm, _ := NewTaskManager(strings.NewReader(`[{"id": "t1", "text": "Task 1"}]`))
	verifier := &FakeVerifier{}
	limiter := NewRateLimiter(10, time.Minute)
	server := NewServer(tm, verifier, limiter, fstest.MapFS{})
	mux := server.Routes()

	// Helper to make multipart request
	makeReq := func(fileName, contentType string, content []byte) *http.Response {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("taskId", "t1")
		part, _ := w.CreateFormFile("photo", fileName)
		part.Write(content)
		w.Close()

		req := httptest.NewRequest("POST", "/api/check", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		return rr.Result()
	}

	t.Run("Valid JPEG outdoors", func(t *testing.T) {
		verifier.Next = Verdict{Match: true, Outdoors: true, Reason: "Good"}
		// valid fake jpeg bytes (ffd8ffe000104a46494600010101004800480000)
		jpegBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00}
		resp := makeReq("test.jpg", "image/jpeg", jpegBytes)
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var cr CheckResponse
		json.NewDecoder(resp.Body).Decode(&cr)
		if !cr.Success {
			t.Errorf("expected success, got false")
		}
	})

	t.Run("Indoors rejection", func(t *testing.T) {
		verifier.Next = Verdict{Match: true, Outdoors: false, Reason: "Indoor"}
		jpegBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00}
		resp := makeReq("test.jpg", "image/jpeg", jpegBytes)
		
		var cr CheckResponse
		json.NewDecoder(resp.Body).Decode(&cr)
		if cr.Success {
			t.Errorf("expected failure for indoors")
		}
		if !strings.Contains(cr.Message, "take it outside") {
			t.Errorf("expected indoors message, got %s", cr.Message)
		}
	})

	t.Run("Invalid file type", func(t *testing.T) {
		resp := makeReq("test.txt", "text/plain", []byte("hello world this is text data that is not an image at all it's long enough to be sniffed"))
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for bad type, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Invalid image format") {
			t.Errorf("expected format error, got %s", string(body))
		}
	})
	t.Run("Apostrophes and Prefix formatting", func(t *testing.T) {
		verifier.Next = Verdict{Match: false, Outdoors: true, Reason: "Not quite. It's a coffee mug."}
		jpegBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00}
		resp := makeReq("test.jpg", "image/jpeg", jpegBytes)
		
		var cr CheckResponse
		json.NewDecoder(resp.Body).Decode(&cr)
		
		expected := "Not quite. It's a coffee mug."
		if cr.Message != expected {
			t.Errorf("expected %q, got %q", expected, cr.Message)
		}
	})
}
