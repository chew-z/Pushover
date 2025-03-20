package main

import (
	"log"
	"os"
	"time"

	"github.com/gregdel/pushover"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appKey       = os.Getenv("APP_KEY")
	recipientKey = os.Getenv("RECIPIENT_KEY")
	device       = os.Getenv("DEVICE_NAME") // empty = all devices
)

func main() {
	notice := "Model thinking is over now."
	title := ""
	if len(os.Args) < 2 {
		log.Println("usage: push <message> [<title>]") // title is optional
		os.Exit(1)
	} else {
		notice = os.Args[1]
	}
	if len(os.Args) > 2 {
		title = os.Args[2]
	}
	if appKey == "" {
		appKey = "a84t4wvdijbn3pcjtpbhnb1vivdn7m" // Aider app
	}

	if recipientKey == "" {
		recipientKey = "gqq7a1wbs4b7nkahg6nemr7jyfprox" // Aider group
	}

	app := pushover.New(appKey)
	recipient := pushover.NewRecipient(recipientKey)

	message := pushover.NewMessageWithTitle(notice, title)
	message.Priority = pushover.PriorityLow
	message.Timestamp = time.Now().Unix()
	message.Expire = 3 * time.Minute
	message.DeviceName = device
	message.Sound = pushover.SoundVibrate
	if _, err := app.SendMessage(message, recipient); err != nil {
		log.Fatalf("failed to send message: %v", err)
	}
}

// This never worked really well - complicatons suck
//
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
