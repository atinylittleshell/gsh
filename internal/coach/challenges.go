package coach

import (
	"fmt"
	"math/rand"
	"time"
)

// ChallengeType represents the type of challenge
type ChallengeType string

const (
	ChallengeTypeDaily   ChallengeType = "daily"
	ChallengeTypeWeekly  ChallengeType = "weekly"
	ChallengeTypeSpecial ChallengeType = "special"
)

// ChallengeDifficulty represents challenge difficulty
type ChallengeDifficulty string

const (
	DifficultyEasy   ChallengeDifficulty = "easy"
	DifficultyMedium ChallengeDifficulty = "medium"
	DifficultyHard   ChallengeDifficulty = "hard"
)

// ChallengeDefinition defines a single challenge
type ChallengeDefinition struct {
	ID          string
	Name        string
	Description string
	Type        ChallengeType
	Category    string // speed, accuracy, productivity, exploration
	Icon        string
	XPReward    int
	Requirement int
	Metric      string
	Difficulty  ChallengeDifficulty
}

// DailyChallenges contains all daily challenge definitions
var DailyChallenges = []ChallengeDefinition{
	// Speed Challenges
	{ID: "daily_speed_demon", Name: "Speed Demon", Description: "Complete 20 commands in under 5 minutes", Type: ChallengeTypeDaily, Category: "speed", Icon: "⚡", XPReward: 50, Requirement: 20, Metric: "commands_in_5min", Difficulty: DifficultyEasy},
	{ID: "daily_blazing_fast", Name: "Blazing Fast", Description: "Execute 10 commands under 30ms each", Type: ChallengeTypeDaily, Category: "speed", Icon: "🔥", XPReward: 75, Requirement: 10, Metric: "fast_commands_30ms", Difficulty: DifficultyMedium},
	{ID: "daily_rapid_fire", Name: "Rapid Fire", Description: "Execute 50 commands in 10 minutes", Type: ChallengeTypeDaily, Category: "speed", Icon: "💨", XPReward: 100, Requirement: 50, Metric: "commands_in_10min", Difficulty: DifficultyHard},

	// Accuracy Challenges
	{ID: "daily_error_free", Name: "Error-Free", Description: "Execute 25 commands without errors", Type: ChallengeTypeDaily, Category: "accuracy", Icon: "🎯", XPReward: 50, Requirement: 25, Metric: "consecutive_success", Difficulty: DifficultyEasy},
	{ID: "daily_perfect_aim", Name: "Perfect Aim", Description: "Achieve 98% accuracy today", Type: ChallengeTypeDaily, Category: "accuracy", Icon: "🏹", XPReward: 75, Requirement: 98, Metric: "daily_accuracy_pct", Difficulty: DifficultyMedium},
	{ID: "daily_no_mistakes", Name: "No Mistakes", Description: "Execute 100 commands with zero errors", Type: ChallengeTypeDaily, Category: "accuracy", Icon: "✨", XPReward: 150, Requirement: 100, Metric: "total_success_no_error", Difficulty: DifficultyHard},

	// Productivity Challenges
	{ID: "daily_alias_advocate", Name: "Alias Advocate", Description: "Use aliases for 10 commands", Type: ChallengeTypeDaily, Category: "productivity", Icon: "🔗", XPReward: 50, Requirement: 10, Metric: "alias_usage_count", Difficulty: DifficultyEasy},
	{ID: "daily_pipe_dream", Name: "Pipe Dream", Description: "Create 5 command pipelines", Type: ChallengeTypeDaily, Category: "productivity", Icon: "🔀", XPReward: 75, Requirement: 5, Metric: "pipelines_used", Difficulty: DifficultyMedium},
	{ID: "daily_efficiency_expert", Name: "Efficiency Expert", Description: "Save 500 keystrokes via predictions/aliases", Type: ChallengeTypeDaily, Category: "productivity", Icon: "💎", XPReward: 100, Requirement: 500, Metric: "keystrokes_saved", Difficulty: DifficultyHard},

	// Exploration Challenges
	{ID: "daily_new_horizons", Name: "New Horizons", Description: "Use 3 commands you haven't used this week", Type: ChallengeTypeDaily, Category: "exploration", Icon: "🌅", XPReward: 75, Requirement: 3, Metric: "new_commands_week", Difficulty: DifficultyMedium},
	{ID: "daily_directory_explorer", Name: "Directory Explorer", Description: "Work in 5 different directories", Type: ChallengeTypeDaily, Category: "exploration", Icon: "🗺️", XPReward: 50, Requirement: 5, Metric: "unique_directories", Difficulty: DifficultyEasy},
	{ID: "daily_flag_finder", Name: "Flag Finder", Description: "Use 10 different command flags", Type: ChallengeTypeDaily, Category: "exploration", Icon: "🚩", XPReward: 75, Requirement: 10, Metric: "unique_flags", Difficulty: DifficultyMedium},

	// Volume Challenges
	{ID: "daily_warm_up", Name: "Warm Up", Description: "Execute 25 commands today", Type: ChallengeTypeDaily, Category: "volume", Icon: "🏃", XPReward: 25, Requirement: 25, Metric: "daily_commands", Difficulty: DifficultyEasy},
	{ID: "daily_active", Name: "Stay Active", Description: "Execute 50 commands today", Type: ChallengeTypeDaily, Category: "volume", Icon: "💪", XPReward: 50, Requirement: 50, Metric: "daily_commands", Difficulty: DifficultyEasy},
	{ID: "daily_power_hour", Name: "Power Hour", Description: "Execute 100 commands today", Type: ChallengeTypeDaily, Category: "volume", Icon: "⚡", XPReward: 75, Requirement: 100, Metric: "daily_commands", Difficulty: DifficultyMedium},
}

// WeeklyChallenges contains all weekly challenge definitions
var WeeklyChallenges = []ChallengeDefinition{
	// Consistency Challenges
	{ID: "weekly_consistency_king", Name: "Consistency King", Description: "Use gsh every day this week", Type: ChallengeTypeWeekly, Category: "streak", Icon: "👑", XPReward: 200, Requirement: 7, Metric: "active_days", Difficulty: DifficultyMedium},
	{ID: "weekly_early_bird", Name: "Early Bird Week", Description: "Execute commands before 7 AM for 5 days", Type: ChallengeTypeWeekly, Category: "streak", Icon: "🐦", XPReward: 150, Requirement: 5, Metric: "early_morning_days", Difficulty: DifficultyHard},

	// Improvement Challenges
	{ID: "weekly_accuracy_ascent", Name: "Accuracy Ascent", Description: "Improve accuracy by 5% vs last week", Type: ChallengeTypeWeekly, Category: "accuracy", Icon: "📈", XPReward: 200, Requirement: 5, Metric: "accuracy_improvement_pct", Difficulty: DifficultyMedium},
	{ID: "weekly_speed_sprint", Name: "Speed Sprint", Description: "Improve avg command time by 10%", Type: ChallengeTypeWeekly, Category: "speed", Icon: "🏃", XPReward: 200, Requirement: 10, Metric: "speed_improvement_pct", Difficulty: DifficultyMedium},

	// Volume Challenges
	{ID: "weekly_500_commands", Name: "500 Commander", Description: "Execute 500 commands this week", Type: ChallengeTypeWeekly, Category: "milestone", Icon: "🎖️", XPReward: 200, Requirement: 500, Metric: "weekly_commands", Difficulty: DifficultyMedium},
	{ID: "weekly_1000_commands", Name: "Thousand Commander", Description: "Execute 1000 commands this week", Type: ChallengeTypeWeekly, Category: "milestone", Icon: "🏅", XPReward: 300, Requirement: 1000, Metric: "weekly_commands", Difficulty: DifficultyHard},
	{ID: "weekly_power_user", Name: "Power User", Description: "Complete all daily challenges this week", Type: ChallengeTypeWeekly, Category: "meta", Icon: "⭐", XPReward: 500, Requirement: 7, Metric: "daily_challenges_completed", Difficulty: DifficultyHard},

	// Learning Challenges
	{ID: "weekly_command_explorer", Name: "Command Explorer", Description: "Use 20 commands you've never used before", Type: ChallengeTypeWeekly, Category: "exploration", Icon: "🔍", XPReward: 250, Requirement: 20, Metric: "new_commands_ever", Difficulty: DifficultyHard},
	{ID: "weekly_git_week", Name: "Git Week", Description: "Execute 100 git commands", Type: ChallengeTypeWeekly, Category: "tool", Icon: "🌿", XPReward: 150, Requirement: 100, Metric: "git_commands", Difficulty: DifficultyMedium},

	// Productivity Challenges
	{ID: "weekly_alias_master", Name: "Alias Master", Description: "Use aliases 100 times this week", Type: ChallengeTypeWeekly, Category: "productivity", Icon: "🔗", XPReward: 200, Requirement: 100, Metric: "weekly_alias_usage", Difficulty: DifficultyMedium},
	{ID: "weekly_pipeline_pro", Name: "Pipeline Pro", Description: "Create 50 pipelines this week", Type: ChallengeTypeWeekly, Category: "productivity", Icon: "🔀", XPReward: 200, Requirement: 50, Metric: "weekly_pipelines", Difficulty: DifficultyMedium},
}

// SpecialChallenges contains special/seasonal challenges
var SpecialChallenges = []ChallengeDefinition{
	// Monthly Challenges
	{ID: "special_monthly_marathon", Name: "Monthly Marathon", Description: "30-day streak this month", Type: ChallengeTypeSpecial, Category: "streak", Icon: "🏅", XPReward: 1000, Requirement: 30, Metric: "monthly_streak", Difficulty: DifficultyHard},
	{ID: "special_10k_month", Name: "10K Month", Description: "Execute 10,000 commands this month", Type: ChallengeTypeSpecial, Category: "milestone", Icon: "🎯", XPReward: 1000, Requirement: 10000, Metric: "monthly_commands", Difficulty: DifficultyHard},

	// Seasonal Challenges
	{ID: "special_hacktoberfest", Name: "Hacktoberfest Hero", Description: "500 git commands in October", Type: ChallengeTypeSpecial, Category: "seasonal", Icon: "🎃", XPReward: 500, Requirement: 500, Metric: "october_git_commands", Difficulty: DifficultyMedium},
	{ID: "special_new_year", Name: "New Year New You", Description: "Complete 7-day streak starting Jan 1", Type: ChallengeTypeSpecial, Category: "seasonal", Icon: "🎆", XPReward: 300, Requirement: 7, Metric: "jan_streak", Difficulty: DifficultyMedium},

	// One-time Challenges
	{ID: "special_first_1000", Name: "First Thousand", Description: "Reach 1000 total commands", Type: ChallengeTypeSpecial, Category: "milestone", Icon: "🎊", XPReward: 200, Requirement: 1000, Metric: "total_commands", Difficulty: DifficultyEasy},
	{ID: "special_level_25", Name: "Quarter Century", Description: "Reach level 25", Type: ChallengeTypeSpecial, Category: "milestone", Icon: "🌟", XPReward: 300, Requirement: 25, Metric: "level", Difficulty: DifficultyMedium},
	{ID: "special_level_50", Name: "Halfway There", Description: "Reach level 50", Type: ChallengeTypeSpecial, Category: "milestone", Icon: "💫", XPReward: 500, Requirement: 50, Metric: "level", Difficulty: DifficultyHard},
	{ID: "special_level_100", Name: "Centurion Level", Description: "Reach level 100", Type: ChallengeTypeSpecial, Category: "milestone", Icon: "👑", XPReward: 1000, Requirement: 100, Metric: "level", Difficulty: DifficultyHard},
}

// GetRandomDailyChallenges returns a set of daily challenges for today
func GetRandomDailyChallenges(count int, seed int64) []ChallengeDefinition {
	if count > len(DailyChallenges) {
		count = len(DailyChallenges)
	}

	// Use date-based seed for consistent daily challenges
	r := rand.New(rand.NewSource(seed))

	// Shuffle and pick
	shuffled := make([]ChallengeDefinition, len(DailyChallenges))
	copy(shuffled, DailyChallenges)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	// Try to get a mix of difficulties
	var easy, medium, hard []ChallengeDefinition
	for _, c := range shuffled {
		switch c.Difficulty {
		case DifficultyEasy:
			easy = append(easy, c)
		case DifficultyMedium:
			medium = append(medium, c)
		case DifficultyHard:
			hard = append(hard, c)
		}
	}

	var result []ChallengeDefinition

	// Pick 1 easy, 2 medium, 1 hard (or adjust based on count)
	if count >= 4 {
		if len(easy) > 0 {
			result = append(result, easy[0])
		}
		if len(medium) > 1 {
			result = append(result, medium[0], medium[1])
		} else if len(medium) > 0 {
			result = append(result, medium[0])
		}
		if len(hard) > 0 {
			result = append(result, hard[0])
		}
	} else {
		result = shuffled[:count]
	}

	// Fill remaining slots if needed
	for len(result) < count && len(shuffled) > len(result) {
		found := false
		for _, c := range shuffled {
			exists := false
			for _, r := range result {
				if r.ID == c.ID {
					exists = true
					break
				}
			}
			if !exists {
				result = append(result, c)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	return result
}

// GetWeeklyChallenges returns weekly challenges
func GetWeeklyChallenges(count int, seed int64) []ChallengeDefinition {
	if count > len(WeeklyChallenges) {
		count = len(WeeklyChallenges)
	}

	r := rand.New(rand.NewSource(seed))

	shuffled := make([]ChallengeDefinition, len(WeeklyChallenges))
	copy(shuffled, WeeklyChallenges)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

// GetDailySeed returns a seed based on the current date for consistent daily challenges
func GetDailySeed() int64 {
	now := time.Now()
	return int64(now.Year()*10000 + int(now.Month())*100 + now.Day())
}

// GetWeeklySeed returns a seed based on the current week
func GetWeeklySeed() int64 {
	now := time.Now()
	year, week := now.ISOWeek()
	return int64(year*100 + week)
}

// GetDailyResetTime returns when daily challenges reset (midnight local time)
func GetDailyResetTime() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
}

// GetWeeklyResetTime returns when weekly challenges reset (Sunday midnight)
func GetWeeklyResetTime() time.Time {
	now := time.Now()
	daysUntilSunday := (7 - int(now.Weekday())) % 7
	if daysUntilSunday == 0 && now.Hour() >= 0 {
		daysUntilSunday = 7 // Next Sunday, not today
	}
	return time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 0, 0, 0, 0, now.Location())
}

// TimeUntilDailyReset returns duration until daily reset
func TimeUntilDailyReset() time.Duration {
	return time.Until(GetDailyResetTime())
}

// TimeUntilWeeklyReset returns duration until weekly reset
func TimeUntilWeeklyReset() time.Duration {
	return time.Until(GetWeeklyResetTime())
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return formatPlural(hours, "hour") + " " + formatPlural(minutes, "min")
	}
	return formatPlural(minutes, "min")
}

func formatPlural(n int, unit string) string {
	if n == 1 || n == -1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
