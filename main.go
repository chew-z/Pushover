package main

import (
	"log"
	"os"
	"time"

	"github.com/gregdel/pushover"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appKey       = getEnvWithDefault("APP_KEY", "")
	recipientKey = getEnvWithDefault("RECIPIENT_KEY", "")
	device       = getEnvWithDefault("DEVICE_NAME", "default-device")
)

func getEnvWithDefault(key string, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultValue
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: ./pushover <message> <title>")
	}

	if appKey == "" || recipientKey == "" {
		log.Fatal("Both APP_KEY and RECIPIENT_KEY environment variables must be set")
	}

	app := pushover.New(appKey)
	recipient := pushover.NewRecipient(recipientKey)

	message := pushover.NewMessageWithTitle(os.Args[1], os.Args[2])
	message.Priority = pushover.PriorityLowest
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
