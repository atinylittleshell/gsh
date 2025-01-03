package history

import (
	"database/sql"
	"time"

	"github.com/atinylittleshell/gsh/pkg/reverse"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
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

	Command   string
	Directory string
	ExitCode  sql.NullInt32
}

func NewHistoryManager(dbFilePath string, logger *zap.Logger) (*HistoryManager, error) {
	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		logger.Error("error opening database", zap.Error(err))
		return nil, err
	}

	db.AutoMigrate(&HistoryEntry{})

	return &HistoryManager{
		db:     db,
		logger: logger,
	}, nil
}

func (historyManager *HistoryManager) StartCommand(command string, directory string) (*HistoryEntry, error) {
	entry := HistoryEntry{
		Command:   command,
		Directory: directory,
	}

	result := historyManager.db.Create(&entry)
	if result.Error != nil {
		historyManager.logger.Error("error creating history entry", zap.Error(result.Error))
		return nil, result.Error
	}

	historyManager.logger.Debug("history entry started", zap.String("command", entry.Command))

	return &entry, nil
}

func (historyManager *HistoryManager) FinishCommand(entry *HistoryEntry, exitCode int) (*HistoryEntry, error) {
	entry.ExitCode = sql.NullInt32{Int32: int32(exitCode), Valid: true}

	result := historyManager.db.Save(entry)
	if result.Error != nil {
		historyManager.logger.Error("error saving history entry", zap.Error(result.Error))
		return nil, result.Error
	}

	historyManager.logger.Debug("history entry finished", zap.String("command", entry.Command), zap.Int("exit_code", exitCode))

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
		historyManager.logger.Error("error fetching recent history entries", zap.Error(result.Error))
		return nil, result.Error
	}

	reverse.Reverse(entries)
	return entries, nil
}

func (historyManager *HistoryManager) ResetHistory() error {
	result := historyManager.db.Exec("DELETE FROM history_entries")
	if result.Error != nil {
		historyManager.logger.Error("error resetting history", zap.Error(result.Error))
		return result.Error
	}

	return nil
}
