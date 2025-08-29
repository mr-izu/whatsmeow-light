package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mdp/qrterminal/v3"
	_ "github.com/lib/pq"

	// NOTE: depending on your checkout / module, you may need to change these imports.
	// If your repo/module uses "go.mau.fi/whatsmeow" replace accordingly.
	"github.com/tulir/whatsmeow"                 // if your module is different, change to go.mau.fi/whatsmeow
	"github.com/tulir/whatsmeow/store/sqlstore" // adjust path if needed
	"github.com/tulir/whatsmeow/types/events"   // adjust path if needed
	"github.com/tulir/whatsmeow/types/proto"    // adjust path if needed

	"context"
)

func main() {
	// postgres DSN from env or default (change user/password/dbname as needed)
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		// Example DSN: "postgres://user:pass@localhost:5432/whatsmeow?sslmode=disable"
		dsn = "postgres://postgres:postgres@localhost:5432/whatsmeow?sslmode=disable"
	}

	// Create SQL store (this will create tables if required)
	db, err := sqlstore.New("postgres", dsn, nil)
	if err != nil {
		log.Fatalf("failed to open sqlstore: %v", err)
	}

	// Get or create container named "default"
	container, err := db.GetContainer("default")
	if err != nil {
		log.Fatalf("failed to get container: %v", err)
	}

	// Create a new whatsmeow client using the container (session will be persisted in Postgres)
	client := whatsmeow.NewClient(container, nil)

	// Add an event handler to reply "hi" to every incoming message
	client.AddEventHandler(func(evt interface{}) {
		switch e := evt.(type) {
		case *events.Message:
			// Only handle messages that have text (conversation)
			msg := e.Message
			if msg == nil {
				return
			}
			// Determine chat/jid to reply to
			jid := e.Info.Sender
			if jid == "" {
				return
			}

			// Send "hi" message back
			_, err := client.SendMessage(context.Background(), jid, &proto.Message{
				Conversation: proto.String("hi"),
			})
			if err != nil {
				log.Printf("failed to send reply to %s: %v", jid, err)
			} else {
				log.Printf("replied to %s with \"hi\"", jid)
			}
		}
	})

	// If we are not logged in, start QR pairing
	if client.Store.ID == nil {
		// Get a QR channel and print QR to terminal
		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			log.Fatalf("failed to get qr channel: %v", err)
		}

		// Run QR printing in a goroutine so we can also Connect
		go func() {
			for evt := range qrChan {
				switch evt.Event {
				case "code":
					fmt.Println("Scan the following QR/pair code with your phone (or read the code):")
					// evt.Code contains the pairing code / qrcode string
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					fmt.Println("\nPair code:", evt.Code)
				case "success":
					fmt.Println("Paired successfully")
				case "error":
					fmt.Println("QR error:", evt.Error)
				}
			}
		}()
	}

	// Connect (this will use the stored session from postgres if available)
	if err := client.Connect(); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	fmt.Println("Connected. Waiting for messages... (Ctrl+C to exit)")

	// Wait for SIGINT / SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
	client.Disconnect()
}
