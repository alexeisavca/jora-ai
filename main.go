package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"google.golang.org/genai"
	
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	bot, err := tgbotapi.NewBotAPI(os.Getenv("telegramBotToken"))
	if err != nil {
		log.Panic(err)
	}

	cmdConfig := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "newsession", Description: "New chat"},
	)
	if _, err := bot.Request(cmdConfig); err != nil {
		log.Printf("Eroare la setarea comenzilor: %v", err)
	}

	// bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: os.Getenv("aiAPIKey"),
	})
	if err != nil {
		log.Fatalf("Eroare la crearea clientului GenAI: %v", err)
	}

	session, err := client.Chats.Create(ctx, "gemini-3.5-flash-lite", nil, nil)
	if err != nil {
		log.Fatalf("Eroare la crearea sesiunii GenAI: %v", err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			fmt.Println(update.Message.Command())
			switch update.Message.Command() {
			case "newsession":
				session, err = client.Chats.Create(ctx, "gemini-3.5-flash-lite", nil, nil)

				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ceva huinea: %v", err)))
					log.Printf("\nCeva huinea: %v", err)

					continue
				}

				bot.Send(tgbotapi.NewMessage(chatID, "New session started"))
				continue
			}

		}

		fmt.Printf("Received: %s", update.Message.Text)

		respStream := session.SendMessageStream(
			ctx,
			genai.Part{
				Text: update.Message.Text,
			},
		)

		initialMessage := tgbotapi.NewMessage(chatID, "Combombulating...")
		initialMessage.ParseMode = tgbotapi.ModeMarkdown
		im, err := bot.Send(initialMessage)
		if err != nil {
			log.Printf("\nCeva huinea: %v", err)
		}
		respText := ""

		for chunk, err := range respStream {
			if err != nil {
				log.Printf("\nCeva huinea: %v", err)
				continue
			}

			respText += chunk.Text()
			newMsg := tgbotapi.NewEditMessageText(chatID, im.MessageID, respText)
			newMsg.ParseMode = tgbotapi.ModeMarkdown
			_, err = bot.Send(newMsg)
			if err != nil {
				log.Printf("\nCeva huinea: %v", err)
				continue
			}
		}
	}
}
