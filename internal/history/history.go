package history

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type HistoryManager struct {
	db *gorm.DB
}

type HistoryEntry struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	Command   string
	Directory string
	ExitCode  sql.NullInt32
}

const (
	historySchemaVersion = 1
)

func NewHistoryManager(dbFilePath string) (*HistoryManager, error) {
	dbFileExists := true
	if _, err := os.Stat(dbFilePath); errors.Is(err, os.ErrNotExist) {
		dbFileExists = false
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "error checking history db: %v\n", err)
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database")
		return nil, err
	}

	if needsMigration(dbFileExists, db) {
		if err := db.AutoMigrate(&HistoryEntry{}); err != nil {
			fmt.Fprintf(os.Stderr, "error auto-migrating database schema: %v\n", err)
			return nil, err
		}
		if err := writeSchemaVersion(historySchemaVersion); err != nil {
			fmt.Fprintf(os.Stderr, "error writing history schema version: %v\n", err)
			return nil, err
		}
	}

	return &HistoryManager{
		db: db,
	}, nil
}

func needsMigration(dbFileExists bool, db *gorm.DB) bool {
	if !dbFileExists {
		return true
	}

	versionMatches, err := schemaVersionMatches()
	if err != nil || !versionMatches {
		return true
	}

	// If the version marker is present but the table is missing (corruption or manual deletion),
	// re-run migrations to restore the schema.
	return !db.Migrator().HasTable(&HistoryEntry{})
}

func writeSchemaVersion(version int) error {
	versionPath := schemaVersionPath()
	return os.WriteFile(versionPath, []byte(strconv.Itoa(version)), 0644)
}

func schemaVersionMatches() (bool, error) {
	versionPath := schemaVersionPath()
	data, err := os.ReadFile(versionPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err != nil {
		return false, err
	}
	trimmed := strings.TrimSpace(string(data))
	version, err := strconv.Atoi(trimmed)
	if err != nil {
		return false, err
	}
	if version != historySchemaVersion {
		return false, fmt.Errorf("history schema version mismatch: got %d, want %d", version, historySchemaVersion)
	}
	return true, nil
}

func schemaVersionPath() string {
	return filepath.Join(core.DataDir(), "history_schema_version")
}

func (historyManager *HistoryManager) StartCommand(command string, directory string) (*HistoryEntry, error) {
	entry := HistoryEntry{
		Command:   command,
		Directory: directory,
	}

	result := historyManager.db.Create(&entry)
	if result.Error != nil {
		return nil, result.Error
	}

	return &entry, nil
}

func (historyManager *HistoryManager) FinishCommand(entry *HistoryEntry, exitCode int) (*HistoryEntry, error) {
	entry.ExitCode = sql.NullInt32{Int32: int32(exitCode), Valid: true}

	result := historyManager.db.Save(entry)
	if result.Error != nil {
		return nil, result.Error
	}

	return entry, nil
}

func (historyManager *HistoryManager) GetRecentEntries(directory string, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	var db = historyManager.db
	if directory != "" {
		db = db.Where("directory = ?", directory)
	}
	result := db.Order("created_at desc").Limit(limit).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	slices.Reverse(entries)
	return entries, nil
}

func (historyManager *HistoryManager) DeleteEntry(id uint) error {
	result := historyManager.db.Delete(&HistoryEntry{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no history entry found with id %d", id)
	}

	return nil
}

func (historyManager *HistoryManager) ResetHistory() error {
	result := historyManager.db.Exec("DELETE FROM history_entries")
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (historyManager *HistoryManager) GetRecentEntriesByPrefix(prefix string, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	result := historyManager.db.Where("command LIKE ?", prefix+"%").
		Order("created_at desc").
		Limit(limit).
		Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	return entries, nil
}

// SearchHistory searches for history entries containing the given substring.
// Returns entries in reverse chronological order (most recent first).
func (historyManager *HistoryManager) SearchHistory(query string, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	result := historyManager.db.Where("command LIKE ?", "%"+query+"%").
		Order("created_at desc").
		Limit(limit).
		Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	return entries, nil
}
