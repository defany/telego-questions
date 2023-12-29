# Telego-questions
## This package provides a question middleware that helps to handle users answers on questions

### Usage:
```go
package main

import (
	"context"
	"fmt"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"log"
	"os"
	qnmanager "github.com/DeFaNy/telego-questions/manager"
)

func main() {
	botToken := os.Getenv("TOKEN_HERE")

	ctx := context.Background()

	// Create Bot with debug on
	// Note: Please keep in mind that default logger may expose sensitive information, use in development only
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get updates channel
	updates, _ := bot.UpdatesViaLongPolling(nil, telego.WithLongPollingContext(ctx))

	qm := qnmanager.NewManager(ctx)

	// Create bot handler
	bh, _ := th.NewBotHandler(bot, updates)

	bh.Use(qm.Middleware)

	// Handle any message
	bh.HandleMessage(func(bot *telego.Bot, message telego.Message) {
		const questionResponseText = "Hey, %s, how old are you?"
		const result = "Your name is %s and you are a %s y.o."

		chatID := tu.ID(message.Chat.ID)

		text := "Hello, what is your name?"

		smp := tu.Message(chatID, text)

		_, err := bot.SendMessage(smp)
		if err != nil {
			log.Fatalf("failed to send message: %s", err.Error())

			return
		}

		err = qm.NewQuestion(bot, message, func(ctx context.Context, bot *telego.Bot, answer qnmanager.Answer) {
			// Get an answer from user and checking if question chan open
			// There is chan and if you want to ask new question another question 
			// just send a message and call this function
			userNameResponse, isOpen := answer()
			if !isOpen {
				log.Println("question chan was closed")

				return
			}

			chatID := userNameResponse.Chat.ID

			// Create an answer message from bot with user name
			smp := tu.Message(tu.ID(chatID), fmt.Sprintf(questionResponseText, userNameResponse.Text))

			_, err := bot.SendMessage(smp)
			if err != nil {
				log.Println("failed to send question to user")

				return
			}
			
			userAgeResponse, isOpen := answer()
			if !isOpen {
				log.Println("question chan was closed")
				
				return 
			}
			
			smp.WithText(fmt.Sprintf(result, userNameResponse.Text, userAgeResponse.Text))

			_, err = bot.SendMessage(smp)
			if err != nil {
				log.Println("failed to send question to user")

				return
			}

			return
		})
		if err != nil {
			log.Fatalf("failed to create question: %s", err.Error())

			return
		}

		return
	})

	// Stop handling updates on exit
	defer bh.Stop()
	defer bot.StopLongPolling()

	// Start handling updates
	bh.Start()
}
```
