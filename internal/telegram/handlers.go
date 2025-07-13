package telegram

import (
	"audioBot/internal/job"
	"context"
	"log"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func IsVideo() th.Predicate {
	return func(ctx context.Context, u telego.Update) bool {
		return u.Message != nil && (u.Message.Video != nil || u.Message.Document != nil)
	}
}

const maxBotAPIFileSize = 50 * 1024 * 1024 // 50 MB

// HandleVideo creates a handler function that enqueues video processing jobs.
func HandleVideo(q *job.Queue, forwardToID int64) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		var fileID, fileName string
		var fileSize int64

		bot := c.Bot()

		switch {
		case update.Message.Video != nil:
			fileID = update.Message.Video.FileID
			fileName = update.Message.Video.FileName
			fileSize = update.Message.Video.FileSize
		case update.Message.Document != nil:
			fileID = update.Message.Document.FileID
			fileName = update.Message.Document.FileName
			fileSize = update.Message.Document.FileSize
		default:
			log.Printf("Unhandled message type in chat %d", update.Message.Chat.ID)
			return nil
		}

		if fileID == "" {
			return nil
		}

		var req job.Request
		if fileSize > maxBotAPIFileSize {
			// Large file: forward and enqueue TDLib job
			fwdMsg, err := bot.ForwardMessage(c, &telego.ForwardMessageParams{
				ChatID:     telego.ChatID{ID: forwardToID},
				FromChatID: telego.ChatID{ID: update.Message.Chat.ID},
				MessageID:  update.Message.MessageID,
			})
			if err != nil {
				log.Printf("Failed to forward large file for chat %d: %v", update.Message.Chat.ID, err)
				return err
			}

			req = job.Request{
				ChatID:             update.Message.Chat.ID,
				FileName:           fileName,
				IsLargeFile:        true,
				ForwardedMessageID: fwdMsg.MessageID,
			}
			log.Printf("Forwarded large file %q, enqueuing TDLib job", fileName)
		} else {
			// Small file: enqueue regular job
			req = job.Request{
				ChatID:   update.Message.Chat.ID,
				FileID:   fileID,
				FileName: fileName,
			}
			log.Printf("Enqueued job for small file %q (ID: %s)", fileName, fileID)
		}

		if err := q.Enqueue(req); err != nil {
			log.Printf("Failed to enqueue job: %v", err)
			_, _ = bot.SendMessage(c, &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: req.ChatID},
				Text:   "Sorry, the job queue is currently full. Please try again later.",
			})
		}
		return nil
	}
}
