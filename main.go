package main

import (
	"fmt"
	"os"

	"github.com/gregdel/pushover"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appKey      = os.Getenv("APP_KEY")
	recipentKey = os.Getenv("RECIPENT_KEY")
)

func main() {
	// Create a new pushover app with a token
	app := pushover.New(appKey)

	// Create a new recipient
	recipient := pushover.NewRecipient(recipentKey)

	// Create the message to send
	// message := pushover.NewMessageWithTitle(os.Args[1], os.Args[2])

	// Send the message to the recipient
	// if _, err := app.SendMessage(message, recipient); err != nil {
	// 	log.Println(err.Error())
	// }

	// Print the response if you want
	// log.Println(response)
	title := os.Args[1]
	text := os.Args[2]
	count := 420
	pct := 69
	// Test Glances API
	fmt.Println(app.SendGlanceUpdate(&pushover.Glance{
		Title:      &title,
		Text:       &text,
		Count:      &count,
		Percent:    &pct,
		DeviceName: "iPhoneIX",
	}, recipient))

}
