package pipeline

import (
	"audioBot/internal/config"
	"audioBot/internal/elevenlabs"
	"audioBot/internal/job"
	"audioBot/internal/media"
	"audioBot/internal/tdlib"
	"audioBot/internal/telegram"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mymmrac/telego"
)

// namedReader implements an io.Reader with a Name method to satisfy telego's interface for file uploads.
type namedReader struct {
	io.Reader
	name string
}

// Name returns the file name.
func (r *namedReader) Name() string {
	return r.name
}

// Processor handles the full pipeline for a single transcription job.
type Processor struct {
	bot         *telego.Bot
	downloader  *telegram.Downloader
	transcriber *elevenlabs.Client
	tdlibClient *tdlib.Downloader
	cfg         *config.Config
}

// NewProcessor creates a new pipeline processor.
func NewProcessor(bot *telego.Bot, tdlibClient *tdlib.Downloader, cfg *config.Config) *Processor {
	return &Processor{
		bot:         bot,
		downloader:  telegram.NewDownloader(bot),
		transcriber: elevenlabs.NewClient(cfg.ElevenLabsAPIKey),
		tdlibClient: tdlibClient,
		cfg:         cfg,
	}
}

// Process orchestrates the download, audio extraction, and transcription.
func (p *Processor) Process(ctx context.Context, r job.Request) error {
	log.Printf("Processing job for chat %d, file: %s", r.ChatID, r.FileName)

	var videoPath string
	var err error

	if r.IsLargeFile {
		// 1a. Wait for TDLib download
		videoPath, err = p.tdlibClient.WaitFile(ctx, int64(r.ForwardedMessageID), r.ChatID)
		if err != nil {
			return fmt.Errorf("TDLib download wait failed: %w", err)
		}
		// For large files, we use the path from TDLib, so no defer os.Remove
		log.Printf("TDLib downloaded file to %s", videoPath)
	} else {
		// 1b. Download file via Bot API
		videoPath, err = p.downloader.DownloadFile(ctx, r.FileID)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		defer os.Remove(videoPath)
		log.Printf("Downloaded file to %s", videoPath)
	}

	// 2. Extract audio
	audioPath, err := media.ExtractAudio(ctx, p.cfg.FFMPEGPath, videoPath)
	if err != nil {
		return fmt.Errorf("audio extraction failed: %w", err)
	}
	defer os.Remove(audioPath)
	log.Printf("Extracted audio to %s", audioPath)

	// 3. Transcribe audio
	params := elevenlabs.TranscribeParams{Diarize: true}
	transcript, err := p.transcriber.Transcribe(ctx, audioPath, params)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}
	log.Printf("Transcription successful for chat %d", r.ChatID)

	// 4. Return transcript
	doc := telego.InputFile{
		File: &namedReader{
			Reader: bytes.NewReader([]byte(transcript.Text)),
			name:   "transcript.txt",
		},
	}
	_, err = p.bot.SendDocument(ctx, &telego.SendDocumentParams{
		ChatID:   telego.ChatID{ID: r.ChatID},
		Document: doc,
	})
	if err != nil {
		return fmt.Errorf("failed to send transcript: %w", err)
	}

	log.Printf("Successfully processed and sent transcript to chat %d", r.ChatID)
	return nil
}
