package main

import (
	"log"
	"os"
	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	appID := os.Getenv("DISCORD_APP_ID") // BotのApplication ID
	guildID := os.Getenv("DISCORD_GUILD_ID") // テスト用サーバのID

	dg, _ := discordgo.New("Bot " + token)
	commands := []*discordgo.ApplicationCommand{
		{Name: "adjust", Description: "来週の日程調整を開始します"},
		{Name: "check", Description: "リアクションを集計して結果を発表します"},
		{Name: "vip", Description: "特定のユーザーを優先メンバーとして登録します",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "登録するユーザー",
					Required:    true,
				},
			},
		},
	}

	_, err := dg.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	if err != nil {
		log.Fatalf("コマンドの登録に失敗しました: %v", err)
	}
	log.Println("コマンドの登録が完了しました！")
}