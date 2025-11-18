package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func ScanInbox(inboxPath string) ([]Note, error) {
	entries, err := os.ReadDir(inboxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inbox: %w", err)
	}

	var notes []Note

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(inboxPath, entry.Name())
		fmt.Printf("[SCAN] Examining: %s\n", entry.Name())
		
		note, err := parseNote(fullPath)
		if err != nil {
			fmt.Printf("  ✗ Error parsing: %v\n", err)
			continue // Skip notes that fail to parse
		}

		if note != nil {
			fmt.Printf("  ✓ Tracked (expires: %s)\n", note.ExpiryDatetime.Format("2006-01-02 15:04:05"))
			notes = append(notes, *note)
		} else {
			fmt.Printf("  - No expiry_date field\n")
		}
	}

	return notes, nil
}

func parseNote(filePath string) (*Note, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Extract YAML frontmatter
	frontmatter, _, err := extractFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	// Parse YAML to find expiry_date
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return nil, err
	}

	expiryStr, ok := data["expiry_date"].(string)
	if !ok {
		return nil, nil // No expiry_date field, skip this note
	}

	// Parse duration (e.g., "7d", "30d")
	duration, err := parseDuration(expiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid expiry_date format: %w", err)
	}

	return &Note{
		Path:           filePath,
		ExpiryDatetime: time.Now().Add(duration),
		TrackedAt:      time.Now(),
	}, nil
}

func extractFrontmatter(content string) (string, string, error) {
	if !strings.HasPrefix(content, "---") {
		return "", content, nil
	}

	// Find the closing ---
	rest := content[3:] // Skip opening ---
	idx := strings.Index(rest, "---")
	if idx == -1 {
		return "", content, nil
	}

	frontmatter := rest[:idx]
	body := rest[idx+3:]

	return strings.TrimSpace(frontmatter), strings.TrimSpace(body), nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Match patterns like "7d", "30d", "2h", etc.
	re := regexp.MustCompile(`^(\d+)([dhms])$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	num, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	case "s":
		return time.Duration(num) * time.Second, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}
