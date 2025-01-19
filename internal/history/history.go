package history

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/atinylittleshell/gsh/pkg/reverse"
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

func NewHistoryManager(dbFilePath string) (*HistoryManager, error) {
	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database")
		return nil, err
	}

	db.AutoMigrate(&HistoryEntry{})

	return &HistoryManager{
		db: db,
	}, nil
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

	reverse.Reverse(entries)
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
