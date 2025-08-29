package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mdp/qrterminal/v3"
	_ "github.com/lib/pq"

	// NOTE: if your local checkout / module path differs (e.g. go.mau.fi/whatsmeow),
	// replace these imports with the correct paths used in your repo.
	"github.com/tulir/whatsmeow"
	"github.com/tulir/whatsmeow/store/sqlstore"
	"github.com/tulir/whatsmeow/types/events"
	"github.com/tulir/whatsmeow/types/proto"
)

func ensureSSLMode(dsn string) string {
	// If the DSN already contains sslmode=, leave it alone.
	if strings.Contains(dsn, "sslmode=") {
		return dsn
	}
	// If DSN already has query params, append sslmode=require, otherwise add it.
	if strings.Contains(dsn, "?") {
		return dsn + "&sslmode=require"
	}
	return dsn + "?sslmode=require"
}

func main() {
	// Prefer DATABASE_URL (common on cloud providers); fallback to POSTGRES_DSN; then default local example
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		// Local default (only for local testing)
		dsn = "postgres://postgres:postgres@localhost:5432/whatsmeow?sslmode=disable"
		log.Println("POSTGRES_DSN / DATABASE_URL not set, falling back to local default (sslmode=disable)")
	} else {
		// For cloud-hosted Postgres, ensure sslmode is set (most cloud providers require SSL)
		dsn = ensureSSLMode(dsn)
	}

	log.Println("Using Postgres DSN from environment (password hidden)")

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
	// Note: depending on whatsmeow version the store ID check might differ; adjust if needed.
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
