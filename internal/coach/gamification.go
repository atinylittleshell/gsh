package coach

import (
	"math"
	"sort"
	"time"
)

// Level titles for each level range
var LevelTitles = map[int]string{
	1:   "Shell Novice",
	11:  "Command Apprentice",
	21:  "Terminal Journeyman",
	36:  "Shell Artisan",
	51:  "Command Master",
	71:  "Terminal Virtuoso",
	86:  "Shell Sage",
	100: "Terminal Transcendent",
}

// GetTitleForLevel returns the title for a given level
func GetTitleForLevel(level int) string {
	title := "Shell Novice"

	// Collect map keys into a slice
	keys := make([]int, 0, len(LevelTitles))
	for lvl := range LevelTitles {
		keys = append(keys, lvl)
	}

	// Sort keys in ascending order
	sort.Ints(keys)

	// Find the highest key that is <= level
	for _, lvl := range keys {
		if level >= lvl {
			title = LevelTitles[lvl]
		} else {
			break // Since keys are sorted, we can break early
		}
	}

	return title
}

// XPForLevel calculates the XP required to reach a specific level
// Uses an exponential curve: XP = 100 * (level^1.5)
func XPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	return int(100 * math.Pow(float64(level), 1.5))
}

// XPForNextLevel returns XP needed to reach next level from current
func XPForNextLevel(currentLevel int) int {
	return XPForLevel(currentLevel+1) - XPForLevel(currentLevel)
}

// LevelFromTotalXP calculates level from total XP
func LevelFromTotalXP(totalXP int) int {
	level := 1
	for XPForLevel(level+1) <= totalXP {
		level++
		if level >= 100 {
			break
		}
	}
	return level
}

// XPProgressInLevel returns progress towards next level (0.0 to 1.0)
func XPProgressInLevel(totalXP int, currentLevel int) float64 {
	if currentLevel >= 100 {
		return 1.0
	}
	currentLevelXP := XPForLevel(currentLevel)
	nextLevelXP := XPForLevel(currentLevel + 1)
	xpInLevel := totalXP - currentLevelXP
	xpNeeded := nextLevelXP - currentLevelXP
	if xpNeeded <= 0 {
		return 1.0
	}
	return float64(xpInLevel) / float64(xpNeeded)
}

// StreakMultiplier returns XP multiplier based on streak length
func StreakMultiplier(streakDays int) float64 {
	switch {
	case streakDays >= 365:
		return 5.0
	case streakDays >= 100:
		return 3.0
	case streakDays >= 60:
		return 2.5
	case streakDays >= 30:
		return 2.0
	case streakDays >= 14:
		return 1.5
	case streakDays >= 7:
		return 1.25
	case streakDays >= 3:
		return 1.1
	default:
		return 1.0
	}
}

// PrestigeMultiplier returns XP multiplier based on prestige level
func PrestigeMultiplier(prestige int) float64 {
	if prestige <= 0 {
		return 1.0
	}
	if prestige >= 10 {
		return 2.0
	}
	return 1.0 + float64(prestige)*0.1
}

// XPReward represents a breakdown of XP earned
type XPReward struct {
	Base             int
	StreakBonus      int
	PrestigeBonus    int
	Total            int
	StreakMultiplier float64
}

// CalculateXPReward calculates XP reward with all multipliers
func CalculateXPReward(baseXP int, streakDays int, prestige int) XPReward {
	streakMult := StreakMultiplier(streakDays)
	prestigeMult := PrestigeMultiplier(prestige)

	withStreak := int(float64(baseXP) * streakMult)
	streakBonus := withStreak - baseXP

	total := int(float64(withStreak) * prestigeMult)
	prestigeBonus := total - withStreak

	return XPReward{
		Base:             baseXP,
		StreakBonus:      streakBonus,
		PrestigeBonus:    prestigeBonus,
		Total:            total,
		StreakMultiplier: streakMult,
	}
}

// CommandXPValue returns base XP for various command activities
type CommandXPValue struct {
	SuccessfulCommand  int
	UsedAlias          int
	UsedPrediction     int
	FastCommand        int // Under 50ms
	FirstCommandOfDay  int
	FirstCommandOfHour int
	NewCommandUsed     int
	PipelineUsed       int
	NoErrorStreak10    int
}

// DefaultCommandXP returns default XP values
func DefaultCommandXP() CommandXPValue {
	return CommandXPValue{
		SuccessfulCommand:  1,
		UsedAlias:          2,
		UsedPrediction:     3,
		FastCommand:        1,
		FirstCommandOfDay:  10,
		FirstCommandOfHour: 2,
		NewCommandUsed:     25,
		PipelineUsed:       2,
		NoErrorStreak10:    15,
	}
}

// StreakMilestone represents a streak milestone with rewards
type StreakMilestone struct {
	Days       int
	XPReward   int
	BadgeID    string
	Message    string
	Multiplier float64
}

// StreakMilestones defines all streak milestones
var StreakMilestones = []StreakMilestone{
	{Days: 3, XPReward: 25, BadgeID: "", Message: "3-day streak! You're building a habit!", Multiplier: 1.1},
	{Days: 7, XPReward: 100, BadgeID: "streak_weekly", Message: "1 week streak! Multiplier now 1.25x!", Multiplier: 1.25},
	{Days: 14, XPReward: 200, BadgeID: "streak_dedicated", Message: "2 weeks! You're dedicated! Multiplier now 1.5x!", Multiplier: 1.5},
	{Days: 30, XPReward: 500, BadgeID: "streak_monthly", Message: "30 days! Incredible! Multiplier now 2x!", Multiplier: 2.0},
	{Days: 60, XPReward: 1000, BadgeID: "streak_unstoppable", Message: "60 days! You're unstoppable! Multiplier now 2.5x!", Multiplier: 2.5},
	{Days: 100, XPReward: 2000, BadgeID: "streak_quarterly", Message: "100 DAYS! LEGENDARY! Multiplier now 3x!", Multiplier: 3.0},
	{Days: 180, XPReward: 3000, BadgeID: "streak_half_year", Message: "Half a year! You're in the elite!", Multiplier: 3.0},
	{Days: 365, XPReward: 10000, BadgeID: "streak_legendary", Message: "ONE FULL YEAR! Multiplier now 5x!", Multiplier: 5.0},
}

// GetStreakMilestone returns the milestone for a given streak day if it's a milestone
func GetStreakMilestone(days int) *StreakMilestone {
	for _, m := range StreakMilestones {
		if m.Days == days {
			return &m
		}
	}
	return nil
}

// GetCurrentStreakMilestone returns the highest achieved milestone
func GetCurrentStreakMilestone(days int) *StreakMilestone {
	var highest *StreakMilestone
	for i := range StreakMilestones {
		if days >= StreakMilestones[i].Days {
			highest = &StreakMilestones[i]
		}
	}
	return highest
}

// GetNextStreakMilestone returns the next milestone to achieve
func GetNextStreakMilestone(days int) *StreakMilestone {
	for i := range StreakMilestones {
		if days < StreakMilestones[i].Days {
			return &StreakMilestones[i]
		}
	}
	return nil
}

// DaysToNextMilestone returns days until next streak milestone
func DaysToNextMilestone(currentStreak int) int {
	next := GetNextStreakMilestone(currentStreak)
	if next == nil {
		return 0
	}
	return next.Days - currentStreak
}

// IsStreakActive checks if streak should continue based on last active date
func IsStreakActive(lastActive time.Time) bool {
	if lastActive.IsZero() {
		return false
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastActiveDay := time.Date(lastActive.Year(), lastActive.Month(), lastActive.Day(), 0, 0, 0, 0, lastActive.Location())

	// Streak is active if last active was today or yesterday
	daysSince := 0
	for d := lastActiveDay; d.Before(today); d = d.AddDate(0, 0, 1) {
		daysSince++
	}
	return daysSince <= 1
}

// CanContinueStreak checks if streak can still be continued (grace period until noon)
func CanContinueStreak(lastActive time.Time) bool {
	if lastActive.IsZero() {
		return true // Can start new streak
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastActiveDay := time.Date(lastActive.Year(), lastActive.Month(), lastActive.Day(), 0, 0, 0, 0, lastActive.Location())

	daysSince := 0
	for d := lastActiveDay; d.Before(today); d = d.AddDate(0, 0, 1) {
		daysSince++
	}

	if daysSince == 0 {
		return true // Same day
	}
	if daysSince == 1 {
		return true // Yesterday, can continue
	}
	if daysSince == 2 && now.Hour() < 12 {
		return true // Day before yesterday but before noon (grace period)
	}

	return false
}

// CalculateNewStreak calculates what the streak should be
func CalculateNewStreak(currentStreak int, lastActive time.Time, useFreeze bool, freezesAvailable int) (newStreak int, freezeUsed bool) {
	if lastActive.IsZero() {
		return 1, false // Starting fresh
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastActiveDay := time.Date(lastActive.Year(), lastActive.Month(), lastActive.Day(), 0, 0, 0, 0, lastActive.Location())

	daysSince := 0
	for d := lastActiveDay; d.Before(today); d = d.AddDate(0, 0, 1) {
		daysSince++
	}

	switch daysSince {
	case 0:
		return currentStreak, false // Same day, no change
	case 1:
		return currentStreak + 1, false // Yesterday, increment
	case 2:
		if useFreeze && freezesAvailable > 0 {
			return currentStreak + 1, true // Use freeze to maintain
		}
		if now.Hour() < 12 {
			return currentStreak + 1, false // Grace period
		}
		return 1, false // Reset
	default:
		if useFreeze && freezesAvailable > 0 && daysSince <= 3 {
			return currentStreak + 1, true
		}
		return 1, false // Reset
	}
}

// StreakFreezesEarned calculates how many freezes have been earned based on streak
// Earn 1 freeze per week of streak (max 3)
func StreakFreezesEarned(currentStreak int) int {
	earned := currentStreak / 7
	if earned > 3 {
		return 3
	}
	return earned
}

// LevelUpInfo contains information about a level up
type LevelUpInfo struct {
	OldLevel int
	NewLevel int
	OldTitle string
	NewTitle string
	XPToNext int
	Unlocks  []string // Features unlocked at this level
}

// CheckLevelUp checks if user leveled up and returns info
func CheckLevelUp(oldTotalXP, newTotalXP int) *LevelUpInfo {
	oldLevel := LevelFromTotalXP(oldTotalXP)
	newLevel := LevelFromTotalXP(newTotalXP)

	if newLevel <= oldLevel {
		return nil
	}

	return &LevelUpInfo{
		OldLevel: oldLevel,
		NewLevel: newLevel,
		OldTitle: GetTitleForLevel(oldLevel),
		NewTitle: GetTitleForLevel(newLevel),
		XPToNext: XPForNextLevel(newLevel),
		Unlocks:  getUnlocksForLevel(newLevel),
	}
}

func getUnlocksForLevel(level int) []string {
	unlocks := []string{}

	switch level {
	case 5:
		unlocks = append(unlocks, "Daily challenges unlocked")
	case 10:
		unlocks = append(unlocks, "Weekly challenges unlocked")
	case 25:
		unlocks = append(unlocks, "Advanced analytics")
	case 50:
		unlocks = append(unlocks, "Custom dashboard themes")
	case 75:
		unlocks = append(unlocks, "Extended statistics")
	case 100:
		unlocks = append(unlocks, "Prestige mode unlocked")
	}

	return unlocks
}

// PrestigeInfo contains information for prestige
type PrestigeInfo struct {
	CurrentPrestige int
	NewPrestige     int
	BonusMultiplier float64
	StarPrefix      string
}

// CanPrestige checks if user can prestige (level 100)
func CanPrestige(level, currentPrestige int) bool {
	return level >= 100 && currentPrestige < 10
}

// GetPrestigeInfo returns info about prestiging
func GetPrestigeInfo(currentPrestige int) *PrestigeInfo {
	newPrestige := currentPrestige + 1
	if newPrestige > 10 {
		newPrestige = 10
	}

	stars := ""
	for i := 0; i < newPrestige; i++ {
		stars += "★"
	}

	return &PrestigeInfo{
		CurrentPrestige: currentPrestige,
		NewPrestige:     newPrestige,
		BonusMultiplier: PrestigeMultiplier(newPrestige),
		StarPrefix:      stars,
	}
}
