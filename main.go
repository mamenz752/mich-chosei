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

func getNextWeekRange(now time.Time) []string {
	// 1. 次の月曜日までの日数を計算する
	// time.Weekdayは 日=0, 月=1, ..., 土=6
	daysUntilNextMonday := int(time.Monday - now.Weekday())
	if daysUntilNextMonday <= 0 {
		daysUntilNextMonday += 7
	}

	// 2. 次の月曜日の日付を取得
	nextMonday := now.AddDate(0, 0, daysUntilNextMonday+7)
	japaneseWeekdays := []string{"日", "月", "火", "水", "木", "金", "土"}

	var schedule []string

	// 3. 月曜日から日曜日までの7日分をループ
	for i := 0; i < 7; i++ {
		targetDate := nextMonday.AddDate(0, 0, i)

		// 書式化: "2026/03/02（月）20:30~"
		// ※時間は固定で20:30としています
		str := fmt.Sprintf("%04d/%02d/%02d（%s）20:30~",
			targetDate.Year(),
			targetDate.Month(),
			targetDate.Day(),
			japaneseWeekdays[targetDate.Weekday()],
		)
		schedule = append(schedule, str)
	}

	return schedule
}

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

		emojiWeek := []string{"🌙", "🔥", "💧", "🌲", "👑", "🏖️", "☀️"}
		dateOfWeek := getNextWeekRange(now)

		content := fmt.Sprintf(`
				🔔 【リマインド】来週のMICHに向けて、日程調整を始めましょう。
				🌙 → %s
				🔥 → %s
				💧 → %s
				🌲 → %s
				👑 → %s
				🏖️ → %s
				☀️ → %s
			`,
			dateOfWeek[0],
			dateOfWeek[1],
			dateOfWeek[2],
			dateOfWeek[3],
			dateOfWeek[4],
			dateOfWeek[5],
			dateOfWeek[6],
		)

		if weeks%2 == 0 {
			msg, err := dg.ChannelMessageSend(channelID, content)
			if err != nil {
				log.Println("Error sending message:", err)
				return
			}

			for _, emoji := range emojiWeek {
				err := dg.MessageReactionAdd(msg.ChannelID, msg.ID, emoji)
				if err != nil {
					log.Println("Error adding reaction:", err)
				}
			}
			log.Println("Sent scheduled message.")
		} else {
			log.Println("Skipped: This is an off-week.")
		}
	})
	if err != nil {
		log.Fatal("Error adding cron job:", err)
	}
	c.Start()

	// Final. Health check server for Koyeb
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
