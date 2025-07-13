package tdlib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	tdlib "github.com/zelenin/go-tdlib/client"
)

// --- singleton bootstrap ---

var (
	tdOnce   sync.Once
	tdClient *tdlib.Client
	initErr  error
)

type Opts struct {
	APIID, APIHash string
	Session        string // base-64 from TDLib/Telethon
	DownloadDir    string // absolute path
	MaxParallelDL  int    // e.g. 2
}

func Init(ctx context.Context, o Opts) (*Downloader, error) {
	tdOnce.Do(func() {
		cfg := &tdlib.SetTdlibParametersRequest{
			UseMessageDatabase: true,
			UseFileDatabase:    true,
			SystemLanguageCode: "en",
			DeviceModel:        "audioBot",
			ApplicationVersion: "0.1",
			ApiId:              mustAtoi(o.APIID),
			ApiHash:            o.APIHash,
			DatabaseDirectory:  filepath.Join(o.DownloadDir, "db"),
			FilesDirectory:     o.DownloadDir,
		}
		auth := tdlib.ClientAuthorizer(cfg)

		go func() {
			for {
				select {
				case <-auth.State:
					log.Println("Authorization state received: {}", auth.State)
					// session string is baked in => no interactive login
				case <-ctx.Done():
					return
				}
			}
		}()

		tdClient, initErr = tdlib.NewClient(auth)
	})

	if initErr != nil {
		return nil, initErr
	}

	return &Downloader{
		td:  tdClient,
		sem: make(chan struct{}, o.MaxParallelDL),
		dir: o.DownloadDir,
	}, nil
}

// --- Downloader with 2-parallel limit and 5 s API timeouts ---

type Downloader struct {
	td  *tdlib.Client
	sem chan struct{}
	dir string
}

// WaitFile blocks until TDLib fully saves file for given message, returns full path.
func (d *Downloader) WaitFile(ctx context.Context, msgID, chatID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // hard 5 s API deadline
	defer cancel()

	// 1) fetch message
	msg, err := d.td.GetMessage(&tdlib.GetMessageRequest{
		ChatId:    chatID,
		MessageId: msgID,
	})
	if err != nil {
		return "", err
	}

	// 2) locate fileID in content
	fileID, err := extractFileID(msg)
	if err != nil {
		return "", err
	}

	// 3) concurrency gate
	d.sem <- struct{}{}
	defer func() { <-d.sem }()

	// 4) kick download (non-blocking)
	_, _ = d.td.DownloadFile(&tdlib.DownloadFileRequest{
		FileId:      fileID,
		Priority:    32,
		Offset:      0,
		Limit:       0,
		Synchronous: false,
	})

	// 5) poll until complete or ctx timeout
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			f, _ := d.td.GetFile(&tdlib.GetFileRequest{FileId: fileID})
			if f != nil && f.Local != nil && f.Local.IsDownloadingCompleted {
				return f.Local.Path, nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// helper: extract video/doc fileID
func extractFileID(m *tdlib.Message) (int32, error) {
	switch c := m.Content.(type) {
	case *tdlib.MessageVideo:
		return c.Video.Video.Id, nil
	case *tdlib.MessageDocument:
		return c.Document.Document.Id, nil
	default:
		return 0, errors.New("unsupported content type")
	}
}

func mustAtoi(s string) int32 {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("bad API_ID: %v", err))
	}
	return int32(v)
}
