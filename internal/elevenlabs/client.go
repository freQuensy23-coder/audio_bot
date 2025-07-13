package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const (
	defaultModelID = "scribe_v1"
	apiEndpoint    = "https://api.elevenlabs.io/v1/speech-to-text"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type TranscribeParams struct {
	ModelID     string
	Language    string
	Diarize     bool
	NumSpeakers int
}

type Transcript struct {
	Text string `json:"text"`
}

func (c *Client) Transcribe(ctx context.Context, filePath string, p TranscribeParams) (*Transcript, error) {
	if p.ModelID == "" {
		p.ModelID = defaultModelID
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file to buffer: %w", err)
	}

	_ = writer.WriteField("model_id", p.ModelID)
	if p.Language != "" {
		_ = writer.WriteField("language", p.Language)
	}
	if p.Diarize {
		_ = writer.WriteField("diarize", "true")
		if p.NumSpeakers > 0 {
			_ = writer.WriteField("num_speakers", fmt.Sprintf("%d", p.NumSpeakers))
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var transcript Transcript
	if err := json.NewDecoder(resp.Body).Decode(&transcript); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &transcript, nil
}
