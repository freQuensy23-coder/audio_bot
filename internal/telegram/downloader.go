package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mymmrac/telego"
)

// Downloader handles file downloads from Telegram.
type Downloader struct {
	bot *telego.Bot
}

// NewDownloader creates a new downloader.
func NewDownloader(bot *telego.Bot) *Downloader {
	return &Downloader{bot: bot}
}

// DownloadFile downloads a file from Telegram and saves it to a temporary directory.
func (d *Downloader) DownloadFile(ctx context.Context, fileID string) (string, error) {
	file, err := d.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileURL := d.bot.FileDownloadURL(file.FilePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "telegram-*.video")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return tmpFile.Name(), nil
}
