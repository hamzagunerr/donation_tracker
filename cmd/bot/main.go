package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hayratyardim/donation_tracker/internal/bot"
	"github.com/hayratyardim/donation_tracker/internal/config"
	"github.com/hayratyardim/donation_tracker/internal/database"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config yüklenemedi: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Veritabanı bağlantısı kurulamadı: %v", err)
	}
	defer db.Close()

	log.Println("Veritabanı bağlantısı başarılı")

	telegramBot, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("Bot oluşturulamadı: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Kapatma sinyali alındı, bot durduruluyor...")
		cancel()
	}()

	log.Println("Bot çalışıyor... Durdurmak için Ctrl+C")

	if err := telegramBot.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot hatası: %v", err)
	}

	log.Println("Bot başarıyla kapatıldı")
}
