package history

import (
	"database/sql"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type HistoryManager struct {
	db     *gorm.DB
	logger *zap.Logger
}

type HistoryEntry struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	Command  string
	Stdout   sql.NullString
	Stderr   sql.NullString
	ExitCode sql.NullInt32
}

func NewHistoryManager(dbFilePath string, logger *zap.Logger) (*HistoryManager, error) {
	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&HistoryEntry{})

	return &HistoryManager{
		db:     db,
		logger: logger,
	}, nil
}

func (historyManager *HistoryManager) StartCommand(command string) (*HistoryEntry, error) {
	entry := HistoryEntry{
		Command: command,
	}

	result := historyManager.db.Create(&entry)
	if result.Error != nil {
		return nil, result.Error
	}

	return &entry, nil
}

func (historyManager *HistoryManager) FinishCommand(entry *HistoryEntry, stdout, stderr string, exitCode int) (*HistoryEntry, error) {
	entry.Stdout = sql.NullString{String: stdout, Valid: stdout != ""}
	entry.Stderr = sql.NullString{String: stderr, Valid: stderr != ""}
	entry.ExitCode = sql.NullInt32{Int32: int32(exitCode), Valid: true}

	result := historyManager.db.Save(entry)
	if result.Error != nil {
		return nil, result.Error
	}

	return entry, nil
}

func (historyManager *HistoryManager) GetRecentEntries(limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	result := historyManager.db.Order("created_at desc").Limit(limit).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	return entries, nil
}
