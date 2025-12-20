package coach

import (
	"fmt"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/styles"
)

// RenderDashboard renders the main coach dashboard
func (m *CoachManager) RenderDashboard() string {
	var sb strings.Builder

	profile := m.profile
	stats := m.todayStats

	// Header
	sb.WriteString(styles.AGENT_MESSAGE("╔══════════════════════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  🎮 GSH PRODUCTIVITY COACH                                               ║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╠══════════════════════════════════════════════════════════════════════════╣\n"))

	// Welcome and streak
	streakStr := ""
	if profile.CurrentStreak > 0 {
		streakStr = fmt.Sprintf("🔥 %d-day streak!", profile.CurrentStreak)
	}
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  Welcome back, %s! %s\n", profile.Username, padRight(streakStr, 30))))

	// Level and title
	prestigeStr := ""
	if profile.Prestige > 0 {
		prestigeStr = strings.Repeat("★", profile.Prestige) + " "
	}
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s%s\n", prestigeStr, profile.Title)))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// XP Progress bar
	progress := XPProgressInLevel(profile.TotalXP, profile.Level)
	xpNeeded := XPForNextLevel(profile.Level)
	xpCurrent := profile.TotalXP - XPForLevel(profile.Level)
	progressBar := renderProgressBar(progress, 40)

	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  LEVEL %d %s ⭐ %d / %d XP\n", profile.Level, padRight("", 30), xpCurrent, xpNeeded)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s %.1f%%\n", progressBar, progress*100)))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║══════════════════════════════════════════════════════════════════════════║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Today's stats
	sb.WriteString(styles.AGENT_MESSAGE("║  📊 TODAY'S PROGRESS\n"))
	if stats != nil {
		accuracy := 0.0
		if stats.CommandsExecuted > 0 {
			accuracy = float64(stats.CommandsSuccessful) / float64(stats.CommandsExecuted) * 100
		}
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Commands: %d\n", stats.CommandsExecuted)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Accuracy: %.1f%%\n", accuracy)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Errors: %d\n", stats.CommandsFailed)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  └── XP Earned: %d\n", stats.XPEarned)))
	} else {
		sb.WriteString(styles.AGENT_MESSAGE("║  └── No activity yet today\n"))
	}
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║══════════════════════════════════════════════════════════════════════════║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Daily challenges
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  📋 DAILY CHALLENGES                              Resets in %s\n", formatDurationShort(TimeUntilDailyReset()))))
	for _, challenge := range m.dailyChallenges {
		def := getChallengeDefinition(challenge.ChallengeID)
		if def == nil {
			continue
		}

		status := "⬜"
		progressStr := fmt.Sprintf("%.0f%%", challenge.Progress*100)
		if challenge.Completed {
			status = "✅"
			progressStr = "DONE!"
		} else if challenge.Progress > 0 {
			status = "🔄"
		}

		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s %s - %s (%d/%d) %s\n",
			status, def.Icon, def.Name, challenge.CurrentValue, def.Requirement, progressStr)))
	}
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║══════════════════════════════════════════════════════════════════════════║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Weekly challenges
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  📅 WEEKLY CHALLENGES                            Resets in %s\n", formatDurationShort(TimeUntilWeeklyReset()))))
	for _, challenge := range m.weeklyChallenges {
		def := getChallengeDefinition(challenge.ChallengeID)
		if def == nil {
			continue
		}

		status := "⬜"
		progressStr := fmt.Sprintf("%.0f%%", challenge.Progress*100)
		if challenge.Completed {
			status = "✅"
			progressStr = "DONE!"
		} else if challenge.Progress > 0 {
			status = "🔄"
		}

		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s %s - %s (%d/%d) %s\n",
			status, def.Icon, def.Name, challenge.CurrentValue, def.Requirement, progressStr)))
	}
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Footer
	sb.WriteString(styles.AGENT_MESSAGE("╠══════════════════════════════════════════════════════════════════════════╣\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  @!coach [stats|achievements|challenges|tips|reset-tips]                 ║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╚══════════════════════════════════════════════════════════════════════════╝\n"))

	return sb.String()
}

// RenderStats renders detailed statistics
func (m *CoachManager) RenderStats() string {
	var sb strings.Builder

	profile := m.profile
	stats := m.todayStats

	sb.WriteString(styles.AGENT_MESSAGE("╔══════════════════════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  📊 DETAILED STATISTICS                                                  ║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╠══════════════════════════════════════════════════════════════════════════╣\n"))

	// Profile stats
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  👤 PROFILE\n"))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Level: %d (%s)\n", profile.Level, profile.Title)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Total XP: %d\n", profile.TotalXP)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Current Streak: %d days\n", profile.CurrentStreak)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Longest Streak: %d days\n", profile.LongestStreak)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  └── Streak Freezes: %d available\n", profile.StreakFreezes)))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Multipliers
	streakMult := StreakMultiplier(profile.CurrentStreak)
	prestigeMult := PrestigeMultiplier(profile.Prestige)
	totalMult := streakMult * prestigeMult

	sb.WriteString(styles.AGENT_MESSAGE("║  ⚡ MULTIPLIERS\n"))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Streak Bonus: %.2fx\n", streakMult)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Prestige Bonus: %.2fx\n", prestigeMult)))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  └── Total: %.2fx XP\n", totalMult)))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Today's stats
	if stats != nil {
		sb.WriteString(styles.AGENT_MESSAGE("║  📈 TODAY\n"))
		accuracy := 0.0
		if stats.CommandsExecuted > 0 {
			accuracy = float64(stats.CommandsSuccessful) / float64(stats.CommandsExecuted) * 100
		}
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Commands: %d\n", stats.CommandsExecuted)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Successful: %d\n", stats.CommandsSuccessful)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Failed: %d\n", stats.CommandsFailed)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Accuracy: %.1f%%\n", accuracy)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Pipelines Used: %d\n", stats.PipelinesUsed)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Aliases Used: %d\n", stats.AliasesUsed)))
		if stats.AvgCommandTimeMs > 0 {
			sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Avg Command Time: %dms\n", stats.AvgCommandTimeMs)))
		}
		if stats.FastestCommandMs > 0 {
			sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Fastest Command: %dms\n", stats.FastestCommandMs)))
		}
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  └── XP Earned: %d\n", stats.XPEarned)))
	}

	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╚══════════════════════════════════════════════════════════════════════════╝\n"))

	return sb.String()
}

// RenderAchievements renders achievements browser
func (m *CoachManager) RenderAchievements() string {
	var sb strings.Builder

	sb.WriteString(styles.AGENT_MESSAGE("╔══════════════════════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  🏆 ACHIEVEMENTS                                                          ║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╠══════════════════════════════════════════════════════════════════════════╣\n"))

	// Count unlocked
	var unlocked, total int
	var achievements []CoachAchievement
	m.db.Where("profile_id = ?", m.profile.ID).Find(&achievements)

	achievementMap := make(map[string]*CoachAchievement)
	for i := range achievements {
		achievementMap[achievements[i].AchievementID] = &achievements[i]
		if achievements[i].UnlockedAt.Valid {
			unlocked++
		}
	}
	total = len(AllAchievements)

	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %d / %d Unlocked (%.0f%%)\n", unlocked, total, float64(unlocked)/float64(total)*100)))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Group by category
	categories := []AchievementCategory{
		CategoryStreak, CategoryMilestone, CategoryAccuracy, CategorySpeed,
		CategoryProductivity, CategoryLearning, CategoryGit, CategorySpecial,
	}

	categoryNames := map[AchievementCategory]string{
		CategoryStreak:       "🔥 STREAK",
		CategoryMilestone:    "🏆 MILESTONE",
		CategoryAccuracy:     "🎯 ACCURACY",
		CategorySpeed:        "⚡ SPEED",
		CategoryProductivity: "🛠️ PRODUCTIVITY",
		CategoryLearning:     "📚 LEARNING",
		CategoryGit:          "🌿 GIT",
		CategorySpecial:      "🎪 SPECIAL",
	}

	for _, cat := range categories {
		catAchievements := GetAchievementsByCategory(cat)
		if len(catAchievements) == 0 {
			continue
		}

		catUnlocked := 0
		for _, a := range catAchievements {
			if ua, ok := achievementMap[a.ID]; ok && ua.UnlockedAt.Valid {
				catUnlocked++
			}
		}

		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s (%d/%d)\n", categoryNames[cat], catUnlocked, len(catAchievements))))

		// Show top 3 achievements (prioritize unlocked and near-unlock)
		shown := 0
		for _, a := range catAchievements {
			if shown >= 3 {
				break
			}

			ua := achievementMap[a.ID]
			status := "🔒"
			progressStr := ""

			if ua != nil && ua.UnlockedAt.Valid {
				status = "✨"
				progressStr = "UNLOCKED"
			} else if ua != nil && ua.Progress > 0 {
				status = "⏳"
				progressStr = fmt.Sprintf("%.0f%%", ua.Progress*100)
			}

			tierIcon := getTierIcon(a.Tier)
			sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  │ %s %s %s - %s %s\n",
				status, tierIcon, a.Name, truncate(a.Description, 30), progressStr)))
			shown++
		}
		sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	}

	sb.WriteString(styles.AGENT_MESSAGE("╚══════════════════════════════════════════════════════════════════════════╝\n"))

	return sb.String()
}

// RenderChallenges renders challenges view
func (m *CoachManager) RenderChallenges() string {
	var sb strings.Builder

	sb.WriteString(styles.AGENT_MESSAGE("╔══════════════════════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  🎯 CHALLENGES                                                            ║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╠══════════════════════════════════════════════════════════════════════════╣\n"))

	// Daily challenges
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  📋 DAILY CHALLENGES                         Resets in %s\n", formatDurationShort(TimeUntilDailyReset()))))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	for _, challenge := range m.dailyChallenges {
		def := getChallengeDefinition(challenge.ChallengeID)
		if def == nil {
			continue
		}

		status := "⬜"
		if challenge.Completed {
			status = "✅"
		} else if challenge.Progress > 0 {
			status = "🔄"
		}

		progressBar := renderProgressBar(challenge.Progress, 20)
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s %s %s\n", status, def.Icon, def.Name)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║     %s\n", def.Description)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║     %s %d/%d  +%d XP\n", progressBar, challenge.CurrentValue, def.Requirement, def.XPReward)))
		sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	}

	// Weekly challenges
	sb.WriteString(styles.AGENT_MESSAGE("║──────────────────────────────────────────────────────────────────────────║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  📅 WEEKLY CHALLENGES                       Resets in %s\n", formatDurationShort(TimeUntilWeeklyReset()))))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	for _, challenge := range m.weeklyChallenges {
		def := getChallengeDefinition(challenge.ChallengeID)
		if def == nil {
			continue
		}

		status := "⬜"
		if challenge.Completed {
			status = "✅"
		} else if challenge.Progress > 0 {
			status = "🔄"
		}

		progressBar := renderProgressBar(challenge.Progress, 20)
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s %s %s\n", status, def.Icon, def.Name)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║     %s\n", def.Description)))
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║     %s %d/%d  +%d XP\n", progressBar, challenge.CurrentValue, def.Requirement, def.XPReward)))
		sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	}

	sb.WriteString(styles.AGENT_MESSAGE("╚══════════════════════════════════════════════════════════════════════════╝\n"))

	return sb.String()
}

// Helper functions

func renderProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(progress * float64(width))
	empty := width - filled

	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDurationShort(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		return fmt.Sprintf("%dd %dh", days, hours%24)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func getTierIcon(tier AchievementTier) string {
	switch tier {
	case TierBronze:
		return "🥉"
	case TierSilver:
		return "🥈"
	case TierGold:
		return "🥇"
	case TierPlatinum:
		return "💎"
	case TierDiamond:
		return "👑"
	default:
		return "⭐"
	}
}

// RenderAllTips renders a view of all tips in the database
func (m *CoachManager) RenderAllTips() string {
	var sb strings.Builder

	sb.WriteString(styles.AGENT_MESSAGE("╔══════════════════════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  💡 ALL TIPS                                                              ║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╠══════════════════════════════════════════════════════════════════════════╣\n"))

	// Get all tips from database
	var tips []CoachDatabaseTip
	m.db.Where("active = ?", true).Order("category, priority DESC").Find(&tips)

	// Count by source
	staticCount := 0
	llmCount := 0
	for _, tip := range tips {
		if tip.Source == "static" {
			staticCount++
		} else {
			llmCount++
		}
	}

	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  Total: %d tips (%d static, %d AI-generated)\n", len(tips), staticCount, llmCount)))
	sb.WriteString(styles.AGENT_MESSAGE("║\n"))

	// Group by category
	categories := make(map[string][]CoachDatabaseTip)
	categoryOrder := []string{}
	for _, tip := range tips {
		if _, exists := categories[tip.Category]; !exists {
			categoryOrder = append(categoryOrder, tip.Category)
		}
		categories[tip.Category] = append(categories[tip.Category], tip)
	}

	categoryIcons := map[string]string{
		"productivity": "💡",
		"shortcut":     "⌨️",
		"command":      "📚",
		"git":          "🌿",
		"fun_fact":     "🎲",
		"motivation":   "🚀",
		"efficiency":   "⚡",
		"learning":     "📖",
		"error_fix":    "🔧",
		"workflow":     "🔄",
		"alias":        "⌨️",
		"tool_discovery": "🔍",
		"security":     "🔒",
		"time_management": "⏰",
		"encouragement": "💪",
	}

	for _, cat := range categoryOrder {
		catTips := categories[cat]
		icon := categoryIcons[cat]
		if icon == "" {
			icon = "📌"
		}

		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  %s %s (%d tips)\n", icon, strings.ToUpper(cat), len(catTips))))

		// Show up to 5 tips per category
		showCount := len(catTips)
		if showCount > 5 {
			showCount = 5
		}

		for i := 0; i < showCount; i++ {
			tip := catTips[i]
			sourceTag := ""
			if tip.Source == "llm" {
				sourceTag = " [AI]"
			}
			shownInfo := ""
			if tip.ShownCount > 0 {
				shownInfo = fmt.Sprintf(" (shown %dx)", tip.ShownCount)
			}
			sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  │ %s%s%s\n", truncate(tip.Title+": "+tip.Content, 60), sourceTag, shownInfo)))
		}

		if len(catTips) > 5 {
			sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  │ ... and %d more\n", len(catTips)-5)))
		}
		sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	}

	// Show tip generation status
	sb.WriteString(styles.AGENT_MESSAGE("║──────────────────────────────────────────────────────────────────────────║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("║  📊 TIP GENERATION STATUS\n"))
	sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  ├── Commands since last generation: %d / 1000\n", m.profile.CommandsSinceLastTipGen)))
	if m.profile.LastTipGenTime.Valid {
		sb.WriteString(styles.AGENT_MESSAGE(fmt.Sprintf("║  └── Last generated: %s\n", m.profile.LastTipGenTime.Time.Format("2006-01-02 15:04"))))
	} else {
		sb.WriteString(styles.AGENT_MESSAGE("║  └── Last generated: Never\n"))
	}

	sb.WriteString(styles.AGENT_MESSAGE("║\n"))
	sb.WriteString(styles.AGENT_MESSAGE("╚══════════════════════════════════════════════════════════════════════════╝\n"))

	return sb.String()
}
