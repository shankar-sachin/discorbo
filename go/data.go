package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func dataPath(file string) string {
	return filepath.Join("..", "src", "data", file)
}

func readData(file string, out any) error {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := dataPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) == "" {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, out)
}

func writeData(file string, in any) error {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := dataPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func readReminders() []reminderEntry {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := dataPath("reminders.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return []reminderEntry{}
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(path, []byte("[]"), 0o644)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return []reminderEntry{}
	}
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "{}" {
		raw = []byte("[]")
	}
	list := []reminderEntry{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return []reminderEntry{}
	}
	return list
}

func writeReminders(reminders []reminderEntry) {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := dataPath("reminders.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	out, err := json.MarshalIndent(reminders, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, out, 0o644)
}

func clearUserKeyInObject(file, userID string) {
	blob := map[string]any{}
	if err := readData(file, &blob); err != nil {
		return
	}
	delete(blob, userID)
	_ = writeData(file, blob)
}

func startReminderLoop(s *discordgo.Session) {
	go func() {
		tick := time.NewTicker(10 * time.Second)
		defer tick.Stop()
		for range tick.C {
			now := time.Now().UnixMilli()
			reminders := readReminders()
			if len(reminders) == 0 {
				continue
			}
			pending := make([]reminderEntry, 0, len(reminders))
			for _, r := range reminders {
				if r.DueTime > now {
					pending = append(pending, r)
					continue
				}
				ch, err := s.UserChannelCreate(r.UserID)
				if err == nil && ch != nil {
					_, _ = s.ChannelMessageSend(ch.ID, fmt.Sprintf("Reminder: %s", r.Message))
				}
			}
			if len(pending) != len(reminders) {
				writeReminders(pending)
			}
		}
	}()
}

func migrateV1Data() {
	// Check if old daily-rewards.json has coin data that should be in economy-users.json
	// This is a one-time migration for v1 -> v2
	path := dataPath("daily-rewards.json")
	if _, err := os.Stat(path); err != nil {
		return // No v1 data to migrate
	}
	log.Println("v1 data detected - migration available if needed")
}
