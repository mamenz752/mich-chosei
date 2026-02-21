package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	// 1. Load environment variables from .env file
	godotenv.Load()
	token := os.Getenv("DISCORD_BOT_TOKEN")
	channelID := os.Getenv("CHANNEL_ID")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Create a new Discord session using the provided bot token
	dg, err := discordgo.New("Bot " + strings.TrimSpace(token))
	if err != nil {
		log.Fatal("Error creating session:", err)
	}

	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}

	log.Println("Bot is now running.")

	// TODO: Write your bot's logic here (e.g., event handlers, commands, etc.)
	// 3. Set up cron job settings
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatal("Error loading location:", err)
	}
	c := cron.New(cron.WithLocation(loc))

	baseDate := time.Date(2026, 2, 23, 0, 0, 0, 0, time.Local)

	_, err = c.AddFunc("0 9 * * 1", func() {
		now := time.Now()
		days := int(now.Sub(baseDate).Hours() / 24)
		weeks := days / 7

		if weeks%2 == 0 {
			dg.ChannelMessageSend(channelID, "🔔 【リマインド】来週は集まる週です！日程調整を始めましょう。")
			log.Println("Sent scheduled message.")
		} else {
			log.Println("Skipped: This is an off-week.")
		}
	})
	if err != nil {
		log.Fatal("Error adding cron job:", err)
	}
	c.Start()

	// 4. Health check server for Koyeb
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Bot is alive.")
			log.Printf("Health check server starting on port %s", port)
			if err := http.ListenAndServe(":"+port, nil); err != nil {
				log.Fatal(err)
			}
		})
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
