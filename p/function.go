package p

import (
	"context"
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
	"cloud.google.com/go/firestore"
)

type DayResult struct {
	Day				string
	DateStr   		string
	Emoji      		string
	TotalCount 		int
	SpecificUsers	[]string
}

var (
	db *firestore.Client
	projectID = os.Getenv("GCP_PROJECT_ID")
	botToken = os.Getenv("DISCORD_BOT_TOKEN")
	channelID = os.Getenv("CHANNEL_ID")
	dayNames = []string{"月", "火", "水", "木", "金", "土", "日"}
)

func init() {
	ctx := context.Background()
	var err error
	db, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("Firestore init error: %v", err)
	}
}

func DiscordInteraction(w http.ResponseWriter, r *http.Request) {
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return
	}

	var interaction discordgo.InteractionCreate
	if err := json.NewDecoder(r.Body).Decode(&interaction); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if interaction.Type == discordgo.InteractionPing {
		json.NewEncoder(w).Encode(discordgo.InteractionResponse{
			Type: discordgo.InteractionResponsePong,
		})
		return
	}

	ctx := r.Context()
	switch interaction.ApplicationCommandData().Name {
	case "adjust":
		handleAdjustCommand(ctx, dg, &interaction)
	case "check":
		handleCheckCommand(ctx, dg, &interaction)
	case "vip":
		handleVIPCommand(ctx, dg, &interaction)
	}
}

func handleAdjustCommand(ctx context.Background, dg *discordgo.Session, i *discordgo.InteractionCreate) {
	dg.InteractionRespond(i Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	
	dates := getNextWeekRange(time.Now())
	emojis = []string{"🌙", "🔥", "💧", "🌲", "👑", "🏖️", "☀️"}

	content := "@everyone\n🔔 【リマインド】来週のMICHに向けて、日程調整を始めましょう。\n"
	for i, emoji := range emojis {
		content += fmt.Sprintf("%s → %s\n", emoji, dates[i])
	}

	msg, err := dg.ChannelMessageSend(channelID, content)
	if err != nil {
		http.Error(w, "Failed to send message", 500)
		return
	}
	
	_, _ = db.Collection("sessions").Doc("latest").Set(ctx, map[string]interface{}{
		"msg_id": msg.ID,
		"dates": dates,
	})

	for _, emoji := range emojis {
		_ := dg.MessageReactionAdd(channelID, msg.ID, emoji)
	}
	dg.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: pointer("日程調整メッセージを送信しました。"),
	})
}

func handleCheckCommand(ctx context.Background, dg *discordgo.Session, i *discordgo.InteractionCreate) {
	dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	specificIDs := []string{}
	if doc, err := db.Collection("configs").Doc("users").Get(ctx); err == nil {
		if ids, ok := doc.Data()["specific_ids"].([]any); ok {
			for _, v := range ids {
				specificIDs = append(specificIDs, v.(string))
			}
		}
	}

	doc, err := db.Collection("sessions").Doc("latest").Get(ctx)
	if err != nil {
		http.Error(w, "Failed to get session data", 404)
		return
	}
	msgID := doc.Data()["msg_id"].(string)
	dates := doc.Data()["dates"].([]any)
	
	var results []DayResult
	for i , e := range emojis {
		users, _ := dg.MessageReactions(channelID, msgID, emoji, 100, "", "")
		res := DayResult{Day: dayNames[i], DateStr: dates[i], Emoji: emoji}
		for _, u := range users {
			if u.Bot { continue }
			res.TotalCount++
			if slices.Contains(specificIDs, u.ID) {
				res.SpecificUsers = append(res.SpecificUsers, u.ID)
			}
		}
		results = append(results, res)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if len(results[i].SpecificUsers) != len(results[j].SpecificUsers) {
			return len(results[i].SpecificUsers) > len(results[j].SpecificUsers)
		}
		return results[i].TotalCount > results[j].TotalCount
	})

	best := results[0]
	resMsg := fmt.Sprintf("@everyone\n📊 【日程調整集計結果】\n次回は **%s** に決定しました。", best.DateStr)

	_, _ = dg.ChannelMessageSend(channelID,resMsg)
	dg.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: pointer("日程調整の集計完了メッセージを送信しました。"),
	})
}

func handleVIPCommand(ctx context.Context, dg *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().options
	targetUser := options[0].UserValue(dg)
	
	dogRef := db.Collection("configs").Doc("users")
	_ = db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(dogRef)
		if doc.Exists() {
			if cur, ok := doc.Data()["specific_ids"].([]any); ok {
				for _, v := range cur {
					ids = append(ids, v.(string))
				}
		}
		if !slices.Contains(ids, targetUser.ID) {
			ids = append(ids, targetUser.ID)
		}
		return tx.Set(dogRef, map[string]interface{}{
			"specific_ids": ids,
		})
	})

	dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("ユーザー %s を優先コアメンバーに登録しました。", targetUser.Username),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func getNextWeekRange(now time.Time) []string {
	// 1. 次の月曜日までの日数を計算する
	offset := int(time.Monday - now.Weekday())
	if offset <= 0 {
		offset += 7
	}
	nextMonday := now.AddDate(0, 0, offset)
	
	var schedule []string
	for i := 0; i < 7; i++ {
		target := nextMonday.AddDate(0, 0, i)
		idx := int(target.Weekday() + 6) % 7

		str := fmt.Sprintf("%04d/%02d/%02d（%s）20:30~",
			target.Year(),
			target.Month(),
			target.Day(),
			dayNames[idx],
		)
		schedule = append(schedule, str)
	}

	return schedule
}

func pointer[T any](v T) *T { return &v }