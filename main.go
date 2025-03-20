package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gregdel/pushover"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appKey      = os.Getenv("APP_KEY")
	recipentKey = os.Getenv("RECIPENT_KEY")
	device      = os.Getenv("DEVICE_NAME")
)

func main() {
	// Create a new pushover app with a token
	app := pushover.New(appKey)

	// Create a new recipient
	recipient := pushover.NewRecipient(recipentKey)

	// Create the message to send
	// message := pushover.NewMessageWithTitle(os.Args[1], os.Args[2])
	message := &pushover.Message{
		Message:    os.Args[1],
		Title:      os.Args[2],
		Priority:   pushover.PriorityLowest,
		Timestamp:  time.Now().Unix(),
		Expire:     3 * time.Minute,
		DeviceName: device,
		Sound:      pushover.SoundVibrate,
	}
	// Send the message to the recipient
	response, err := app.SendMessage(message, recipient)
	if err != nil {
		fmt.Println(err.Error())
	}
	// Print the response if you want
	log.Println(response)

}

// Send message to Watch complications
// title := firstN(os.Args[1], 100)
// text := firstN(os.Args[2], 100)
// subtext := firstN(os.Args[3], 100)
// count := 480
// pct := 36
// // Test Glances API
// fmt.Println(app.SendGlanceUpdate(&pushover.Glance{
// 	Title:      &title,
// 	Text:       &text,
// 	Subtext:    &subtext,
// 	Count:      &count,
// 	Percent:    &pct,
// 	DeviceName: device,
// }, recipient))

// func firstN(s string, n int) string {
// 	if len(s) < 1 {
// 		return ""
// 	} else if len(s) > n {
// 		return s[:n]
// 	}
// 	return s
// }
