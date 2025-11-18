package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Don't delete, just show what would be deleted")
	dbPath := flag.String("db", "knox.db", "Path to SQLite database")
	flag.Parse()

	// Initialize database
	db, err := NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Scan inbox for fleeting notes
	inboxPath := os.ExpandEnv("$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/ludus/inbox")
	notes, err := ScanInbox(inboxPath)
	if err != nil {
		log.Fatalf("Failed to scan inbox: %v", err)
	}

	// Track notes in database
	for _, note := range notes {
		if err := db.TrackNote(note); err != nil {
			log.Printf("Failed to track note %s: %v", note.Path, err)
		}
	}

	// Check for expired notes
	expired, err := db.GetExpired()
	if err != nil {
		log.Fatalf("Failed to get expired notes: %v", err)
	}

	// Delete or report expired notes
	if len(expired) > 0 {
		fmt.Printf("Found %d expired note(s)\n", len(expired))
		for _, note := range expired {
			fmt.Printf("  - %s (expired at %s)\n", note.Path, note.ExpiryDatetime.Format(time.RFC3339))
			if !*dryRun {
				if err := os.Remove(note.Path); err != nil {
					log.Printf("Failed to delete %s: %v", note.Path, err)
					continue
				}
				if err := db.DeleteNote(note.Path); err != nil {
					log.Printf("Failed to remove from DB %s: %v", note.Path, err)
				}
			}
		}
	} else {
		fmt.Println("No expired notes found")
	}

	// Write reminder note for expiring notes
	expiringWithin7d, err := db.GetExpiringWithin(7 * 24 * time.Hour)
	if err != nil {
		log.Printf("Failed to get expiring notes: %v", err)
	} else {
		reminderPath := filepath.Join(inboxPath, "_expiry-reminders.md")
		if err := writeReminderNote(reminderPath, expiringWithin7d); err != nil {
			log.Printf("Failed to write reminder note: %v", err)
		} else if len(expiringWithin7d) > 0 {
			fmt.Printf("Updated reminder note (%d note(s) expiring soon)\n", len(expiringWithin7d))
		}
	}
}

func writeReminderNote(path string, notes []Note) error {
	if len(notes) == 0 {
		// Delete reminder if no notes expiring
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
		return nil
	}

	content := "---\ntitle: Expiry Reminders\ntags:\n  - system\n---\n\n"
	content += "# Notes Expiring Soon\n\n"
	content += fmt.Sprintf("Last updated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	for _, note := range notes {
		filename := filepath.Base(note.Path)
		timeUntil := time.Until(note.ExpiryDatetime)
		days := int(timeUntil.Hours() / 24)
		hours := int(timeUntil.Hours()) % 24

		content += fmt.Sprintf("- [[%s]] - Expires in %dd %dh (%s)\n",
			filename,
			days,
			hours,
			note.ExpiryDatetime.Format("2006-01-02 15:04"),
		)
	}

	return os.WriteFile(path, []byte(content), 0644)
}
