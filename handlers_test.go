package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestApp creates an App with a temp storage directory for testing.
func newTestApp(t *testing.T) *App {
	t.Helper()
	return &App{
		Config: Config{
			StoragePath: t.TempDir(),
			Username:    "admin",
			Password:    "admin",
		},
	}
}

// createTestFile creates a file with the given content in the app's storage directory.
func createTestFile(t *testing.T, app *App, name, content string) string {
	t.Helper()
	path := filepath.Join(app.Config.StoragePath, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- parseContentRangeStart ---

func TestParseContentRangeStart(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    int64
		wantErr bool
	}{
		{"normal range", "bytes 1000-1999/5000", 1000, false},
		{"start at zero", "bytes 0-999/5000", 0, false},
		{"unknown total", "bytes 500-999/*", 500, false},
		{"no dash", "invalid", 0, true},
		{"non-numeric start", "bytes abc-999/5000", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseContentRangeStart(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseContentRangeStart(%q) error = %v, wantErr %v", tt.header, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseContentRangeStart(%q) = %d, want %d", tt.header, got, tt.want)
			}
		})
	}
}

// --- safePath ---

func TestSafePath(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"normal filename", "test.txt", false},
		{"dot rejected", ".", true},
		{"dotdot rejected", "..", true},
		{"path traversal stripped to base", "../../etc/passwd", false}, // Base() extracts "passwd"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.safePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("safePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify path is within storage
				absGot, _ := filepath.Abs(got)
				absStorage, _ := filepath.Abs(app.Config.StoragePath)
				if !strings.HasPrefix(absGot, absStorage) {
					t.Errorf("safePath(%q) = %q, not within storage %q", tt.input, absGot, absStorage)
				}
			}
		})
	}
}

// --- indexHandler ---

func TestIndexHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("root path returns HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		app.indexHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Errorf("Content-Type = %q, want text/html", ct)
		}
	})

	t.Run("non-root path returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		w := httptest.NewRecorder()
		app.indexHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// --- uploadHandler (multipart POST) ---

func TestUploadHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("successful upload", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "hello.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("hello world"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		app.uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(app.Config.StoragePath, "hello.txt"))
		if err != nil {
			t.Fatalf("uploaded file not found: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("file content = %q, want %q", string(content), "hello world")
		}
	})

	t.Run("no file returns 400", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("name", "value") // non-file field
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		app.uploadHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("GET method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/upload", nil)
		w := httptest.NewRecorder()
		app.uploadHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// --- putUploadHandler ---

func TestPutUploadHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("new file upload returns 201", func(t *testing.T) {
		content := "test file content"
		req := httptest.NewRequest(http.MethodPut, "/upload/newfile.txt", strings.NewReader(content))
		w := httptest.NewRecorder()
		app.putUploadHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}

		data, err := os.ReadFile(filepath.Join(app.Config.StoragePath, "newfile.txt"))
		if err != nil {
			t.Fatalf("file not found: %v", err)
		}
		if string(data) != content {
			t.Errorf("file content = %q, want %q", string(data), content)
		}
	})

	t.Run("resume upload with Content-Range", func(t *testing.T) {
		// Create initial file with partial content
		initial := "AAAAA"
		createTestFile(t, app, "resume.txt", initial)

		resumed := "BBBBB"
		req := httptest.NewRequest(http.MethodPut, "/upload/resume.txt", strings.NewReader(resumed))
		req.Header.Set("Content-Range", "bytes 5-9/10")
		w := httptest.NewRecorder()
		app.putUploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		data, err := os.ReadFile(filepath.Join(app.Config.StoragePath, "resume.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "AAAAABBBBB" {
			t.Errorf("file content = %q, want %q", string(data), "AAAAABBBBB")
		}
	})

	t.Run("empty filename returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/upload/", strings.NewReader("data"))
		w := httptest.NewRecorder()
		app.putUploadHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("POST method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/upload/file.txt", nil)
		w := httptest.NewRecorder()
		app.putUploadHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// --- uploadRouter ---

func TestUploadRouter(t *testing.T) {
	app := newTestApp(t)

	t.Run("DELETE returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/upload", nil)
		w := httptest.NewRecorder()
		app.uploadRouter(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// --- downloadHandler ---

func TestDownloadHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("successful download", func(t *testing.T) {
		createTestFile(t, app, "download.txt", "file data")

		req := httptest.NewRequest(http.MethodGet, "/download?filename=download.txt", nil)
		w := httptest.NewRecorder()
		app.downloadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		cd := w.Header().Get("Content-Disposition")
		if !strings.Contains(cd, "download.txt") {
			t.Errorf("Content-Disposition = %q, want to contain filename", cd)
		}

		if !strings.Contains(w.Body.String(), "file data") {
			t.Errorf("body = %q, want to contain %q", w.Body.String(), "file data")
		}
	})

	t.Run("file not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/download?filename=nonexistent.txt", nil)
		w := httptest.NewRecorder()
		app.downloadHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("missing filename returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/download", nil)
		w := httptest.NewRecorder()
		app.downloadHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

// --- videoHandler ---

func TestVideoHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("mp4 content type", func(t *testing.T) {
		createTestFile(t, app, "test.mp4", "fake mp4 data")

		req := httptest.NewRequest(http.MethodGet, "/video?filename=test.mp4", nil)
		w := httptest.NewRecorder()
		app.videoHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		ct := w.Header().Get("Content-Type")
		if ct != "video/mp4" {
			t.Errorf("Content-Type = %q, want %q", ct, "video/mp4")
		}
	})

	t.Run("webm content type", func(t *testing.T) {
		createTestFile(t, app, "test.webm", "fake webm data")

		req := httptest.NewRequest(http.MethodGet, "/video?filename=test.webm", nil)
		w := httptest.NewRecorder()
		app.videoHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		ct := w.Header().Get("Content-Type")
		if ct != "video/webm" {
			t.Errorf("Content-Type = %q, want %q", ct, "video/webm")
		}
	})

	t.Run("missing filename returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/video", nil)
		w := httptest.NewRecorder()
		app.videoHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("nonexistent file returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/video?filename=nope.mp4", nil)
		w := httptest.NewRecorder()
		app.videoHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// --- apiFilesHandler ---

func TestApiFilesHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("returns files as JSON", func(t *testing.T) {
		createTestFile(t, app, "a.txt", "aaa")
		createTestFile(t, app, "b.txt", "bbb")

		req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
		w := httptest.NewRecorder()
		app.apiFilesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var files []FileInfo
		if err := json.NewDecoder(w.Body).Decode(&files); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("got %d files, want 2", len(files))
		}
	})

	t.Run("empty directory returns empty array", func(t *testing.T) {
		emptyApp := newTestApp(t) // fresh temp dir

		req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
		w := httptest.NewRecorder()
		emptyApp.apiFilesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		body := strings.TrimSpace(w.Body.String())
		if body != "[]" {
			t.Errorf("body = %q, want %q", body, "[]")
		}
	})
}

// --- apiDeleteHandler ---

func TestApiDeleteHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("successful delete", func(t *testing.T) {
		path := createTestFile(t, app, "todelete.txt", "data")

		form := url.Values{"filename": {"todelete.txt"}}
		req := httptest.NewRequest(http.MethodPost, "/api/delete", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		app.apiDeleteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("file should have been deleted")
		}
	})

	t.Run("nonexistent file returns 404", func(t *testing.T) {
		form := url.Values{"filename": {"nope.txt"}}
		req := httptest.NewRequest(http.MethodPost, "/api/delete", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		app.apiDeleteHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("missing filename returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/delete", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		app.apiDeleteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("GET method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/delete", nil)
		w := httptest.NewRecorder()
		app.apiDeleteHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// --- filesHandler ---

func TestFilesHandler(t *testing.T) {
	app := newTestApp(t)

	t.Run("directory listing", func(t *testing.T) {
		createTestFile(t, app, "listed.txt", "data")

		req := httptest.NewRequest(http.MethodGet, "/files/", nil)
		w := httptest.NewRecorder()
		app.filesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if !strings.Contains(w.Body.String(), "listed.txt") {
			t.Error("directory listing should contain the file name")
		}
	})

	t.Run("file direct download", func(t *testing.T) {
		createTestFile(t, app, "direct.txt", "direct content")

		req := httptest.NewRequest(http.MethodGet, "/files/direct.txt", nil)
		w := httptest.NewRecorder()
		app.filesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		body, _ := io.ReadAll(w.Result().Body)
		if !strings.Contains(string(body), "direct content") {
			t.Errorf("body should contain file content, got %q", string(body))
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files/../../etc/passwd", nil)
		w := httptest.NewRecorder()
		app.filesHandler(w, req)

		// Should be either 403 Forbidden or 404 Not Found
		if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 403 or 404", w.Code)
		}
	})
}

// --- Integration: full request through router with auth ---

func TestIntegrationRouterWithAuth(t *testing.T) {
	app := newTestApp(t)
	mux := app.NewRouter()

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("authenticated request succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
		req.SetBasicAuth("admin", "admin")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}
