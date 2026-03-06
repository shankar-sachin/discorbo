package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ── In-memory data cache ──────────────────────────────────────────────

type cacheEntry struct {
	raw   json.RawMessage
	dirty bool
}

var (
	cacheMu sync.RWMutex
	cache   = map[string]*cacheEntry{}
)

func dataPath(file string) string {
	return filepath.Join("..", "src", "data", file)
}

// diskRead loads raw JSON from disk (no cache).
func diskRead(file string) (json.RawMessage, error) {
	path := dataPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return nil, err
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		raw = []byte("{}")
	}
	return json.RawMessage(raw), nil
}

// diskWrite flushes raw JSON to disk.
func diskWrite(file string, raw json.RawMessage) error {
	path := dataPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

// readData reads JSON into out, using the in-memory cache.
func readData(file string, out any) error {
	cacheMu.RLock()
	ce, ok := cache[file]
	if ok {
		raw := make([]byte, len(ce.raw))
		copy(raw, ce.raw)
		cacheMu.RUnlock()
		return json.Unmarshal(raw, out)
	}
	cacheMu.RUnlock()

	// Cache miss — load from disk
	cacheMu.Lock()
	defer cacheMu.Unlock()
	// Double-check after write lock
	if ce, ok := cache[file]; ok {
		raw := make([]byte, len(ce.raw))
		copy(raw, ce.raw)
		return json.Unmarshal(raw, out)
	}
	raw, err := diskRead(file)
	if err != nil {
		return err
	}
	cache[file] = &cacheEntry{raw: raw, dirty: false}
	return json.Unmarshal(raw, out)
}

// writeData writes JSON and updates the cache, marking it dirty.
func writeData(file string, in any) error {
	raw, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	cacheMu.Lock()
	cache[file] = &cacheEntry{raw: json.RawMessage(raw), dirty: true}
	cacheMu.Unlock()
	return nil
}

// startCacheFlush periodically writes dirty cache entries to disk.
func startCacheFlush() {
	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		for range tick.C {
			flushDirtyCache()
		}
	}()
}

func flushDirtyCache() {
	cacheMu.Lock()
	dirty := map[string]json.RawMessage{}
	for file, ce := range cache {
		if ce.dirty {
			dirty[file] = ce.raw
			ce.dirty = false
		}
	}
	cacheMu.Unlock()

	for file, raw := range dirty {
		if err := diskWrite(file, raw); err != nil {
			log.Printf("cache flush %s: %v", file, err)
		}
	}
	if len(dirty) > 0 {
		log.Printf("cache: flushed %d file(s) to disk", len(dirty))
	}
}

// flushAllCache writes all dirty cache entries (called on shutdown).
func flushAllCache() {
	flushDirtyCache()
	log.Println("cache: final flush complete")
}

func readReminders() []reminderEntry {
	cacheMu.RLock()
	ce, ok := cache["reminders.json"]
	if ok {
		raw := make([]byte, len(ce.raw))
		copy(raw, ce.raw)
		cacheMu.RUnlock()
		list := []reminderEntry{}
		if err := json.Unmarshal(raw, &list); err != nil {
			return []reminderEntry{}
		}
		return list
	}
	cacheMu.RUnlock()

	// Cache miss
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if ce, ok := cache["reminders.json"]; ok {
		raw := make([]byte, len(ce.raw))
		copy(raw, ce.raw)
		list := []reminderEntry{}
		_ = json.Unmarshal(raw, &list)
		return list
	}
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
	cache["reminders.json"] = &cacheEntry{raw: json.RawMessage(raw), dirty: false}
	list := []reminderEntry{}
	_ = json.Unmarshal(raw, &list)
	return list
}

func writeReminders(reminders []reminderEntry) {
	out, err := json.MarshalIndent(reminders, "", "  ")
	if err != nil {
		return
	}
	cacheMu.Lock()
	cache["reminders.json"] = &cacheEntry{raw: json.RawMessage(out), dirty: true}
	cacheMu.Unlock()
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
	// Migrate coin/streak data from daily-rewards.json → economy-users.json
	path := dataPath("daily-rewards.json")
	if _, err := os.Stat(path); err != nil {
		return // No v1 data
	}

	dailyUsers := map[string]dailyUser{}
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("migration: cannot read daily-rewards.json: %v", err)
		return
	}
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "[]" {
		return
	}
	if err := json.Unmarshal(raw, &dailyUsers); err != nil {
		log.Printf("migration: cannot parse daily-rewards.json: %v", err)
		return
	}
	if len(dailyUsers) == 0 {
		return
	}

	// Load economy-users.json
	economyUsers := map[string]economyUser{}
	_ = readData("economy-users.json", &economyUsers)

	migrated := 0
	for uid, du := range dailyUsers {
		eu := economyUsers[uid]
		// Take the higher coin balance (no coin loss)
		if du.Coins > eu.Coins {
			eu.Coins = du.Coins
		}
		if du.Username != "" && eu.Username == "" {
			eu.Username = du.Username
		}
		// Copy streak/claim metadata
		if du.LastClaim > eu.LastClaim {
			eu.LastClaim = du.LastClaim
		}
		if du.Streak > eu.Streak {
			eu.Streak = du.Streak
		}
		if du.TotalClaims > eu.TotalClaims {
			eu.TotalClaims = du.TotalClaims
		}
		if du.BossKills > eu.BossKills {
			eu.BossKills = du.BossKills
		}
		if eu.TotalEarned == 0 && eu.Coins > 0 {
			eu.TotalEarned = eu.Coins
		}
		economyUsers[uid] = eu
		migrated++
	}

	if err := writeData("economy-users.json", economyUsers); err != nil {
		log.Printf("migration: failed to write economy-users.json: %v", err)
		return
	}

	// Backup v1 file
	backupPath := path + ".v1bak"
	if err := os.Rename(path, backupPath); err != nil {
		log.Printf("migration: failed to backup daily-rewards.json: %v", err)
	}

	log.Printf("✅ v1→v2 migration complete: %d user(s) migrated, backup at daily-rewards.json.v1bak", migrated)
}
