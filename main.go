package main

import (
	"audioBot/internal/config"
	"audioBot/internal/job"
	"audioBot/internal/pipeline"
	"audioBot/internal/tdlib"
	"audioBot/internal/telegram"
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	bot, err := telego.NewBot(cfg.TelegramToken, telego.WithDefaultLogger(false, true))
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	tdClient, err := tdlib.Init(ctx, tdlib.Opts{
		APIID:         strconv.Itoa(cfg.TDLibApiID),
		APIHash:       cfg.TDLibApiHash,
		Session:       cfg.TDLibSession,
		DownloadDir:   cfg.TDLibDownloadDir,
		MaxParallelDL: 2,
	})
	if err != nil {
		log.Fatalf("Failed to create TDLib client: %v", err)
	}

	proc := pipeline.NewProcessor(bot, tdClient, cfg)
	queue := job.NewQueue(ctx, cfg, proc)

	updates, _ := bot.UpdatesViaLongPolling(ctx, nil)
	bh, _ := th.NewBotHandler(bot, updates)

	bh.Handle(telegram.HandleVideo(queue, cfg.TDLibForwardTo), telegram.IsVideo())

	go func() {
		log.Println("Bot starting...")
		bh.Start()
	}()

	<-ctx.Done()

	log.Println("Shutting down gracefully...")
	queue.Shutdown()
	log.Println("Bot stopped.")
}
