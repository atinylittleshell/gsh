package coach

import (
	"database/sql"
	"time"
)

// CoachProfile stores user gamification state
type CoachProfile struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	// Identity
	Username string `gorm:"uniqueIndex"`
	Title    string `gorm:"default:'Shell Novice'"`

	// Progression
	Level     int `gorm:"default:1"`
	CurrentXP int `gorm:"default:0"`
	TotalXP   int `gorm:"default:0"`
	Prestige  int `gorm:"default:0"`

	// Streaks
	CurrentStreak   int `gorm:"default:0"`
	LongestStreak   int `gorm:"default:0"`
	StreakFreezes   int `gorm:"default:0"`
	LastActiveDate  sql.NullTime
	StreakStartDate sql.NullTime

	// Customization
	EquippedBadges string `gorm:"type:text"` // JSON array of badge IDs (max 3)
	Settings       string `gorm:"type:text"` // JSON config

	// Tip generation tracking
	CommandsSinceLastTipGen int          `gorm:"default:0"`  // Commands since last LLM tip generation
	LastTipGenTime          sql.NullTime                     // When tips were last generated
	TipsSeeded              bool         `gorm:"default:false"` // Whether static tips have been seeded
}

// CoachAchievement tracks achievement progress for a user
type CoachAchievement struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	ProfileID     uint    `gorm:"index"`
	AchievementID string  `gorm:"index"`
	Progress      float64 `gorm:"default:0"` // 0.0 to 1.0
	CurrentValue  int     `gorm:"default:0"`
	UnlockedAt    sql.NullTime
	Notified      bool `gorm:"default:false"`
}

// CoachChallenge tracks daily/weekly challenge progress
type CoachChallenge struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`

	ProfileID     uint   `gorm:"index"`
	ChallengeID   string `gorm:"index"`
	Type          string // "daily", "weekly", "special"
	StartTime     time.Time
	EndTime       time.Time
	Progress      float64 `gorm:"default:0"`
	CurrentValue  int     `gorm:"default:0"`
	Completed     bool    `gorm:"default:false"`
	RewardClaimed bool    `gorm:"default:false"`
}

// CoachDailyStats aggregates daily activity
type CoachDailyStats struct {
	ID        uint   `gorm:"primaryKey"`
	ProfileID uint   `gorm:"uniqueIndex:idx_profile_date"`
	Date      string `gorm:"uniqueIndex:idx_profile_date"` // YYYY-MM-DD format

	// Command Stats
	CommandsExecuted   int `gorm:"default:0"`
	CommandsSuccessful int `gorm:"default:0"`
	CommandsFailed     int `gorm:"default:0"`
	UniqueCommands     int `gorm:"default:0"`
	CommandCount       int `gorm:"default:0"` // For calculating average command time

	// Efficiency Stats
	AliasesUsed     int `gorm:"default:0"`
	PredictionsUsed int `gorm:"default:0"`
	KeystrokesTotal int `gorm:"default:0"`
	KeystrokesSaved int `gorm:"default:0"`
	PipelinesUsed   int `gorm:"default:0"`

	// Time Stats
	TotalSessionTime int `gorm:"default:0"` // Seconds
	AvgCommandTimeMs int `gorm:"default:0"` // Milliseconds
	FastestCommandMs int `gorm:"default:0"` // Milliseconds

	// XP
	XPEarned int `gorm:"default:0"`

	// Detailed breakdowns (JSON)
	HourlyActivity string `gorm:"type:text"` // JSON [24]int - commands per hour
	CommandTypes   string `gorm:"type:text"` // JSON map[string]int - command categories
	Directories    string `gorm:"type:text"` // JSON map[string]int - directories visited
}

// CoachDismissedInsight tracks dismissed suggestions
type CoachDismissedInsight struct {
	ID          uint   `gorm:"primaryKey"`
	ProfileID   uint   `gorm:"index"`
	InsightHash string `gorm:"index"`
	DismissedAt time.Time
	DismissType string // "skip", "never"
}

// CoachTipHistory tracks shown tips to avoid repetition
type CoachTipHistory struct {
	ID         uint   `gorm:"primaryKey"`
	ProfileID  uint   `gorm:"index"`
	TipID      string `gorm:"index"`
	ShownAt    time.Time
	ShownCount int `gorm:"default:1"`
}

// CoachGeneratedTip stores LLM-generated tips for persistence
type CoachGeneratedTip struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	ProfileID uint      `gorm:"index"`

	TipID      string `gorm:"uniqueIndex"`
	Type       string
	Category   string
	Title      string
	Content    string `gorm:"type:text"`
	Reasoning  string `gorm:"type:text"`
	Command    string
	Suggestion string `gorm:"type:text"`
	Impact     string
	Confidence float64
	Priority   int
	Actionable bool
	ActionType string
	BasedOn    string `gorm:"type:text"` // JSON array

	ExpiresAt   time.Time `gorm:"index"`
	ShownCount  int       `gorm:"default:0"`
	LastShownAt sql.NullTime
	Dismissed   bool `gorm:"default:false"`
	Applied     bool `gorm:"default:false"` // If actionable tip was applied
}

// CoachTipFeedback stores user feedback on tips
type CoachTipFeedback struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	ProfileID uint   `gorm:"index"`
	TipID     string `gorm:"index"`
	Feedback  string // "helpful", "not_helpful", "already_knew", "applied"
	Comment   string `gorm:"type:text"`
}

// CoachDatabaseTip stores all tips (both static and LLM-generated) in the database
type CoachDatabaseTip struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	TipID    string `gorm:"uniqueIndex"` // Unique identifier for the tip
	Source   string `gorm:"index"`       // "static" or "llm"
	Category string `gorm:"index"`       // Category like productivity, shortcut, command, etc.
	Icon     string
	Title    string
	Content  string `gorm:"type:text"`
	Priority int    `gorm:"default:5"` // 1-10, higher = more likely to show

	// LLM-specific fields
	Reasoning  string `gorm:"type:text"` // Why this tip is relevant (for LLM tips)
	Command    string                    // Related command
	Suggestion string `gorm:"type:text"` // Actionable suggestion
	Impact     string                    // Estimated impact
	BasedOn    string `gorm:"type:text"` // JSON array of commands this tip is based on

	// Display tracking
	ShownCount  int `gorm:"default:0"`
	LastShownAt sql.NullTime
	Active      bool `gorm:"default:true"` // Whether this tip should be shown
}

// CoachNotification stores pending notifications for the user
type CoachNotification struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	ProfileID uint      `gorm:"index"`

	Type    string // "achievement", "level_up", "challenge", "streak", "milestone"
	Title   string
	Content string `gorm:"type:text"`
	Icon    string
	XPGain  int
	Shown   bool `gorm:"default:false"`
}

// ====== In-memory types for runtime use ======

// TipType represents the category of a tip
type TipType string

const (
	TipTypeProductivity   TipType = "productivity"
	TipTypeEfficiency     TipType = "efficiency"
	TipTypeLearning       TipType = "learning"
	TipTypeErrorFix       TipType = "error_fix"
	TipTypeWorkflow       TipType = "workflow"
	TipTypeAlias          TipType = "alias"
	TipTypeToolDiscovery  TipType = "tool_discovery"
	TipTypeSecurityTip    TipType = "security"
	TipTypeGitWorkflow    TipType = "git"
	TipTypeTimeManagement TipType = "time_management"
	TipTypeFunFact        TipType = "fun_fact"
	TipTypeEncouragement  TipType = "encouragement"
)

// InsightCategory represents categories for generated insights
type InsightCategory string

const (
	InsightAlias           InsightCategory = "alias_suggestion"
	InsightTypo            InsightCategory = "typo_pattern"
	InsightError           InsightCategory = "error_pattern"
	InsightProductivity    InsightCategory = "productivity_tip"
	InsightLearning        InsightCategory = "learning_opportunity"
	InsightWorkflow        InsightCategory = "workflow_optimization"
	InsightTimeManagement  InsightCategory = "time_management"
	InsightToolDiscovery   InsightCategory = "tool_discovery"
	InsightSecurityTip     InsightCategory = "security_tip"
	InsightGitWorkflow     InsightCategory = "git_workflow"
	InsightDirectoryNav    InsightCategory = "directory_navigation"
	InsightCommandChaining InsightCategory = "command_chaining"
)

// GeneratedTip represents an LLM-generated personalized tip
type GeneratedTip struct {
	ID          string    `json:"id"`
	Type        TipType   `json:"type"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Reasoning   string    `json:"reasoning"`
	Command     string    `json:"command"`
	Suggestion  string    `json:"suggestion"`
	Impact      string    `json:"impact"`
	Confidence  float64   `json:"confidence"`
	Priority    int       `json:"priority"`
	Actionable  bool      `json:"actionable"`
	ActionType  string    `json:"action_type"`
	GeneratedAt time.Time `json:"generated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	BasedOn     []string  `json:"based_on"`
}

// CoachDisplayContent represents content to show in the Assistant Box
type CoachDisplayContent struct {
	Type     string // "tip", "achievement_progress", "achievement_unlock", "challenge", "streak", "insight", "milestone", "fun_fact", "startup"
	Icon     string // Emoji
	Title    string
	Content  string
	Progress float64 // For progress bars (0-1)
	Action   string  // Optional action hint
	Priority int     // Display priority
}

// Insight represents a productivity insight
type Insight struct {
	ID          string
	Category    InsightCategory
	Priority    int // 1-10, higher = more important
	Title       string
	Description string
	Command     string
	Suggestion  string
	Impact      string
	Difficulty  string // "easy", "medium", "hard"
	Actionable  bool
	LearnMore   string
}

// CommandFrequency tracks how often a command is used
type CommandFrequency struct {
	Command   string
	Count     int
	LastUsed  time.Time
	ErrorRate float64
}

// CommandSequence tracks frequently occurring command sequences
type CommandSequence struct {
	Commands []string
	Count    int
	Context  string // Directory or time-based
}

// ErrorPattern tracks recurring error patterns
type ErrorPattern struct {
	Pattern    string
	Count      int
	Command    string
	ErrorType  string // "not_found", "permission", "syntax", "typo"
	Suggestion string
}

// UserStatistics holds computed statistics for display
type UserStatistics struct {
	TotalCommands      int
	TotalSessions      int
	TotalErrors        int
	ErrorRate          float64
	KeystrokesSaved    int
	TimeSavedSeconds   int
	UniqueCommandsUsed int
	AliasUsageRate     float64
	PredictionAccuracy float64
	AvgCommandTimeMs   int
	MostProductiveHour int
	MostProductiveDay  string
}
