package coach

import (
	"database/sql"
	"encoding/json"
	"os/user"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/history"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mvdan.cc/sh/v3/interp"
)

// CoachManager coordinates all coach functionality
type CoachManager struct {
	db             *gorm.DB
	historyManager *history.HistoryManager
	runner         *interp.Runner
	logger         *zap.Logger

	profile  *CoachProfile
	tipCache *TipCache

	// Session tracking
	sessionStart       time.Time
	sessionCommands    int
	sessionErrors      int
	consecutiveSuccess int
	lastCommandTime    time.Time
	todayStats         *CoachDailyStats

	// Active challenges
	dailyChallenges  []CoachChallenge
	weeklyChallenges []CoachChallenge

	// Pending notifications
	pendingNotifications []CoachNotification
}

// NewCoachManager creates a new coach manager
func NewCoachManager(db *gorm.DB, historyManager *history.HistoryManager, runner *interp.Runner, zapLogger *zap.Logger) (*CoachManager, error) {
	// Configure GORM to use silent logger to avoid printing "record not found" messages
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})

	// Run migrations
	err := db.AutoMigrate(
		&CoachProfile{},
		&CoachAchievement{},
		&CoachChallenge{},
		&CoachDailyStats{},
		&CoachDismissedInsight{},
		&CoachTipHistory{},
		&CoachGeneratedTip{},
		&CoachTipFeedback{},
		&CoachNotification{},
	)
	if err != nil {
		return nil, err
	}

	// Get or create profile
	username := getUsername()
	profile := &CoachProfile{}
	result := db.Where("username = ?", username).First(profile)
	if result.Error == gorm.ErrRecordNotFound {
		profile = &CoachProfile{
			Username: username,
			Title:    "Shell Novice",
			Level:    1,
		}
		if err := db.Create(profile).Error; err != nil {
			return nil, err
		}
	} else if result.Error != nil {
		return nil, result.Error
	}

	manager := &CoachManager{
		db:             db,
		historyManager: historyManager,
		runner:         runner,
		logger:         zapLogger,
		profile:        profile,
		tipCache:       NewTipCache(50, 24*time.Hour),
		sessionStart:   time.Now(),
	}

	// Load today's stats
	manager.loadTodayStats()

	// Load active challenges
	manager.loadActiveChallenges()

	// Update streak on session start
	manager.updateStreak()

	return manager, nil
}

// getUsername returns the current username
func getUsername() string {
	u, err := user.Current()
	if err != nil {
		return "user"
	}
	return u.Username
}

// loadTodayStats loads or creates today's stats
func (m *CoachManager) loadTodayStats() {
	today := time.Now().Format("2006-01-02")
	stats := &CoachDailyStats{}

	result := m.db.Where("profile_id = ? AND date = ?", m.profile.ID, today).First(stats)
	if result.Error == gorm.ErrRecordNotFound {
		stats = &CoachDailyStats{
			ProfileID: m.profile.ID,
			Date:      today,
		}
		m.db.Create(stats)
	} else {
		// For existing records, initialize CommandCount if it's 0
		// and AvgCommandTimeMs > 0 (indicating there were commands)
		if stats.CommandCount == 0 && stats.AvgCommandTimeMs > 0 && stats.CommandsExecuted > 0 {
			// Estimate CommandCount from CommandsExecuted for backwards compatibility
			stats.CommandCount = stats.CommandsExecuted
			m.db.Save(stats)
		}
	}

	m.todayStats = stats
}

// loadActiveChallenges loads current challenges
func (m *CoachManager) loadActiveChallenges() {
	now := time.Now()

	// Load daily challenges
	var dailies []CoachChallenge
	m.db.Where("profile_id = ? AND type = ? AND end_time > ?", m.profile.ID, "daily", now).Find(&dailies)

	// If no active daily challenges, create new ones
	if len(dailies) == 0 {
		seed := GetDailySeed()
		defs := GetRandomDailyChallenges(4, seed)
		resetTime := GetDailyResetTime()

		for _, def := range defs {
			challenge := CoachChallenge{
				ProfileID:   m.profile.ID,
				ChallengeID: def.ID,
				Type:        string(def.Type),
				StartTime:   now,
				EndTime:     resetTime,
			}
			m.db.Create(&challenge)
			dailies = append(dailies, challenge)
		}
	}
	m.dailyChallenges = dailies

	// Load weekly challenges
	var weeklies []CoachChallenge
	m.db.Where("profile_id = ? AND type = ? AND end_time > ?", m.profile.ID, "weekly", now).Find(&weeklies)

	// If no active weekly challenges, create new ones
	if len(weeklies) == 0 {
		seed := GetWeeklySeed()
		defs := GetWeeklyChallenges(4, seed)
		resetTime := GetWeeklyResetTime()

		for _, def := range defs {
			challenge := CoachChallenge{
				ProfileID:   m.profile.ID,
				ChallengeID: def.ID,
				Type:        string(def.Type),
				StartTime:   now,
				EndTime:     resetTime,
			}
			m.db.Create(&challenge)
			weeklies = append(weeklies, challenge)
		}
	}
	m.weeklyChallenges = weeklies
}

// updateStreak updates the user's streak
func (m *CoachManager) updateStreak() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if m.profile.LastActiveDate.Valid {
		lastActive := m.profile.LastActiveDate.Time
		lastActiveDay := time.Date(lastActive.Year(), lastActive.Month(), lastActive.Day(), 0, 0, 0, 0, lastActive.Location())

		// Already active today
		if lastActiveDay.Equal(today) {
			return
		}

		// Calculate new streak
		newStreak, freezeUsed := CalculateNewStreak(
			m.profile.CurrentStreak,
			lastActive,
			m.profile.StreakFreezes > 0,
			m.profile.StreakFreezes,
		)

		if freezeUsed {
			m.profile.StreakFreezes--
		}

		// Check for streak milestone
		if newStreak > m.profile.CurrentStreak {
			if milestone := GetStreakMilestone(newStreak); milestone != nil {
				m.addNotification("streak", milestone.Message, "🔥", milestone.XPReward)
				m.addXP(milestone.XPReward, "streak_milestone")
			}
		}

		// Check if streak was reset
		if newStreak < m.profile.CurrentStreak {
			m.logger.Info("Streak reset", zap.Int("old", m.profile.CurrentStreak), zap.Int("new", newStreak))
		}

		m.profile.CurrentStreak = newStreak
		if newStreak > m.profile.LongestStreak {
			m.profile.LongestStreak = newStreak
		}

		// Update streak freezes earned
		earnedFreezes := StreakFreezesEarned(newStreak)
		if earnedFreezes > m.profile.StreakFreezes {
			m.profile.StreakFreezes = earnedFreezes
		}
	} else {
		// First time user
		m.profile.CurrentStreak = 1
		m.profile.StreakStartDate = sql.NullTime{Time: today, Valid: true}
	}

	m.profile.LastActiveDate = sql.NullTime{Time: now, Valid: true}
	m.db.Save(m.profile)
}

// RecordCommand records a command execution for gamification
func (m *CoachManager) RecordCommand(command string, exitCode int, durationMs int64) {
	m.sessionCommands++
	now := time.Now()

	// Track success/failure
	success := exitCode == 0
	if success {
		m.consecutiveSuccess++
	} else {
		m.sessionErrors++
		m.consecutiveSuccess = 0
	}

	// Calculate base XP
	xpValues := DefaultCommandXP()
	baseXP := 0

	if success {
		baseXP += xpValues.SuccessfulCommand

		// Speed bonus
		if durationMs < 50 {
			baseXP += xpValues.FastCommand
		}

		// Alias usage bonus
		if m.isAliasCommand(command) {
			baseXP += xpValues.UsedAlias
		}

		// Pipeline bonus
		if strings.Contains(command, "|") {
			baseXP += xpValues.PipelineUsed
		}

		// First command bonuses
		if m.sessionCommands == 1 {
			baseXP += xpValues.FirstCommandOfDay
		}
		if m.lastCommandTime.IsZero() || now.Hour() != m.lastCommandTime.Hour() {
			baseXP += xpValues.FirstCommandOfHour
		}

		// Consecutive success bonus
		if m.consecutiveSuccess > 0 && m.consecutiveSuccess%10 == 0 {
			baseXP += xpValues.NoErrorStreak10
		}
	}

	// Apply multipliers and add XP
	if baseXP > 0 {
		m.addXP(baseXP, "command")
	}

	// Update today's stats
	m.updateDailyStats(command, success, durationMs)

	// Update challenges
	m.updateChallengeProgress(command, success, durationMs)

	// Check achievements
	m.checkAchievements(command, success, durationMs)

	m.lastCommandTime = now
}

// addXP adds XP with multipliers and checks for level up
func (m *CoachManager) addXP(baseXP int, source string) {
	reward := CalculateXPReward(baseXP, m.profile.CurrentStreak, m.profile.Prestige)

	oldTotalXP := m.profile.TotalXP
	m.profile.CurrentXP += reward.Total
	m.profile.TotalXP += reward.Total

	// Check for level up
	newLevel := LevelFromTotalXP(m.profile.TotalXP)
	if newLevel > m.profile.Level {
		levelUp := CheckLevelUp(oldTotalXP, m.profile.TotalXP)
		if levelUp != nil {
			m.profile.Level = newLevel
			m.profile.Title = GetTitleForLevel(newLevel)
			m.addNotification("level_up",
				"You are now Level "+formatInt(newLevel)+" - "+m.profile.Title,
				"⭐", 0)
		}
	}

	// Update daily XP
	if m.todayStats != nil {
		m.todayStats.XPEarned += reward.Total
		m.db.Save(m.todayStats)
	}

	m.db.Save(m.profile)
}

// updateDailyStats updates today's statistics
func (m *CoachManager) updateDailyStats(command string, success bool, durationMs int64) {
	if m.todayStats == nil {
		m.loadTodayStats()
	}

	m.todayStats.CommandsExecuted++
	if success {
		m.todayStats.CommandsSuccessful++
	} else {
		m.todayStats.CommandsFailed++
	}

	// Track pipelines
	if strings.Contains(command, "|") {
		m.todayStats.PipelinesUsed++
	}

	// Track aliases
	if m.isAliasCommand(command) {
		m.todayStats.AliasesUsed++
	}

	// Update average command time using incremental formula
	m.todayStats.CommandCount++
	if m.todayStats.CommandCount == 1 {
		// First command, set average to its duration
		m.todayStats.AvgCommandTimeMs = int(durationMs)
	} else {
		// Incremental average: new_avg = old_avg + (new_value - old_avg) / count
		oldAvg := float64(m.todayStats.AvgCommandTimeMs)
		newValue := float64(durationMs)
		count := float64(m.todayStats.CommandCount)
		newAvg := oldAvg + (newValue-oldAvg)/count
		m.todayStats.AvgCommandTimeMs = int(newAvg)
	}

	// Track fastest command
	if m.todayStats.FastestCommandMs == 0 || int(durationMs) < m.todayStats.FastestCommandMs {
		m.todayStats.FastestCommandMs = int(durationMs)
	}

	// Update hourly activity
	hour := time.Now().Hour()
	var hourly [24]int
	if m.todayStats.HourlyActivity != "" {
		_ = json.Unmarshal([]byte(m.todayStats.HourlyActivity), &hourly)
	}
	hourly[hour]++
	hourlyJSON, _ := json.Marshal(hourly)
	m.todayStats.HourlyActivity = string(hourlyJSON)

	m.db.Save(m.todayStats)
}

// updateChallengeProgress updates progress on active challenges
func (m *CoachManager) updateChallengeProgress(command string, success bool, durationMs int64) {
	// Update daily challenges
	for i := range m.dailyChallenges {
		if m.dailyChallenges[i].Completed {
			continue
		}

		def := getChallengeDefinition(m.dailyChallenges[i].ChallengeID)
		if def == nil {
			continue
		}

		// Update based on metric
		updated := false
		switch def.Metric {
		case "daily_commands":
			m.dailyChallenges[i].CurrentValue = m.todayStats.CommandsExecuted
			updated = true
		case "consecutive_success":
			if success && m.consecutiveSuccess > m.dailyChallenges[i].CurrentValue {
				m.dailyChallenges[i].CurrentValue = m.consecutiveSuccess
				updated = true
			}
		case "fast_commands_30ms":
			if success && durationMs < 30 {
				m.dailyChallenges[i].CurrentValue++
				updated = true
			}
		case "pipelines_used":
			if strings.Contains(command, "|") {
				m.dailyChallenges[i].CurrentValue++
				updated = true
			}
		case "alias_usage_count":
			if m.isAliasCommand(command) {
				m.dailyChallenges[i].CurrentValue++
				updated = true
			}
		case "unique_directories":
			m.dailyChallenges[i].CurrentValue = m.countUniqueDirectories()
			updated = true
		}

		if updated {
			m.dailyChallenges[i].Progress = float64(m.dailyChallenges[i].CurrentValue) / float64(def.Requirement)
			if m.dailyChallenges[i].Progress >= 1.0 {
				m.completeChallenge(&m.dailyChallenges[i], def)
			}
			m.db.Save(&m.dailyChallenges[i])
		}
	}

	// Update weekly challenges similarly
	for i := range m.weeklyChallenges {
		if m.weeklyChallenges[i].Completed {
			continue
		}

		def := getChallengeDefinition(m.weeklyChallenges[i].ChallengeID)
		if def == nil {
			continue
		}

		updated := false
		switch def.Metric {
		case "weekly_commands":
			m.weeklyChallenges[i].CurrentValue = m.countWeeklyCommands()
			updated = true
		case "active_days":
			m.weeklyChallenges[i].CurrentValue = m.countActiveDaysThisWeek()
			updated = true
		case "git_commands":
			if strings.HasPrefix(command, "git ") {
				m.weeklyChallenges[i].CurrentValue++
				updated = true
			}
		}

		if updated {
			m.weeklyChallenges[i].Progress = float64(m.weeklyChallenges[i].CurrentValue) / float64(def.Requirement)
			if m.weeklyChallenges[i].Progress >= 1.0 {
				m.completeChallenge(&m.weeklyChallenges[i], def)
			}
			m.db.Save(&m.weeklyChallenges[i])
		}
	}
}

// completeChallenge handles challenge completion
func (m *CoachManager) completeChallenge(challenge *CoachChallenge, def *ChallengeDefinition) {
	challenge.Completed = true
	m.addNotification("challenge",
		def.Name+" completed! +"+formatInt(def.XPReward)+" XP",
		def.Icon, def.XPReward)
	m.addXP(def.XPReward, "challenge")
}

// checkAchievements checks for newly unlocked achievements
func (m *CoachManager) checkAchievements(command string, success bool, durationMs int64) {
	// Get all achievements
	for _, def := range AllAchievements {
		// Check if already unlocked
		var existing CoachAchievement
		result := m.db.Where("profile_id = ? AND achievement_id = ?", m.profile.ID, def.ID).First(&existing)
		if result.Error == nil && existing.UnlockedAt.Valid {
			continue // Already unlocked
		}

		// Calculate progress based on metric
		var currentValue int
		switch def.Metric {
		case "total_commands":
			currentValue = m.getTotalCommands()
		case "current_streak":
			currentValue = m.profile.CurrentStreak
		case "consecutive_success":
			currentValue = m.consecutiveSuccess
		case "unique_commands":
			currentValue = m.countUniqueCommands()
		case "pipelines_used":
			currentValue = m.countTotalPipelines()
		case "alias_usage_count":
			currentValue = m.countTotalAliasUsage()
		case "time_saved_seconds":
			currentValue = m.getTotalTimeSaved()
		case "fast_commands_50ms":
			if success && durationMs < 50 {
				currentValue = m.countFastCommands(50)
			}
		case "git_commits":
			currentValue = m.countGitCommits()
		// Add more metrics as needed
		default:
			continue
		}

		// Update or create achievement progress
		progress := float64(currentValue) / float64(def.Requirement)
		if progress > 1.0 {
			progress = 1.0
		}

		if result.Error == gorm.ErrRecordNotFound {
			existing = CoachAchievement{
				ProfileID:     m.profile.ID,
				AchievementID: def.ID,
				CurrentValue:  currentValue,
				Progress:      progress,
			}
		} else {
			existing.CurrentValue = currentValue
			existing.Progress = progress
		}

		// Check for unlock
		if currentValue >= def.Requirement && !existing.UnlockedAt.Valid {
			existing.UnlockedAt = sql.NullTime{Time: time.Now(), Valid: true}
			m.addNotification("achievement",
				def.Name+" - "+def.Description,
				def.Icon, def.XPReward)
			m.addXP(def.XPReward, "achievement")
		}

		m.db.Save(&existing)
	}
}

// addNotification adds a pending notification
func (m *CoachManager) addNotification(notifType, content, icon string, xp int) {
	notif := CoachNotification{
		ProfileID: m.profile.ID,
		Type:      notifType,
		Title:     getTitleForNotificationType(notifType),
		Content:   content,
		Icon:      icon,
		XPGain:    xp,
	}
	m.db.Create(&notif)
	m.pendingNotifications = append(m.pendingNotifications, notif)
}

// GetPendingNotifications returns and clears pending notifications
func (m *CoachManager) GetPendingNotifications() []CoachNotification {
	notifs := m.pendingNotifications
	m.pendingNotifications = nil

	// Mark as shown
	for _, n := range notifs {
		m.db.Model(&n).Update("shown", true)
	}

	return notifs
}

// GetProfile returns the user profile
func (m *CoachManager) GetProfile() *CoachProfile {
	return m.profile
}

// GetTodayStats returns today's statistics
func (m *CoachManager) GetTodayStats() *CoachDailyStats {
	return m.todayStats
}

// GetDailyChallenges returns active daily challenges
func (m *CoachManager) GetDailyChallenges() []CoachChallenge {
	return m.dailyChallenges
}

// GetWeeklyChallenges returns active weekly challenges
func (m *CoachManager) GetWeeklyChallenges() []CoachChallenge {
	return m.weeklyChallenges
}

// GetDisplayContent returns content for the Assistant Box
func (m *CoachManager) GetDisplayContent() *CoachDisplayContent {
	// Priority 1: Pending notifications
	if len(m.pendingNotifications) > 0 {
		notif := m.pendingNotifications[0]
		return &CoachDisplayContent{
			Type:     notif.Type,
			Icon:     notif.Icon,
			Title:    notif.Title,
			Content:  notif.Content,
			Priority: 10,
		}
	}

	// Priority 2: Near-complete challenges
	for _, c := range m.dailyChallenges {
		if !c.Completed && c.Progress >= 0.8 {
			def := getChallengeDefinition(c.ChallengeID)
			if def != nil {
				return &CoachDisplayContent{
					Type:     "challenge_progress",
					Icon:     def.Icon,
					Title:    "Almost there!",
					Content:  def.Name + ": " + formatInt(c.CurrentValue) + "/" + formatInt(def.Requirement),
					Progress: c.Progress,
					Priority: 8,
				}
			}
		}
	}

	// Priority 3: Static tip
	tip := GetRandomStaticTip()
	if tip != nil {
		return ConvertStaticTipToDisplay(tip)
	}

	return nil
}

// GetStartupContent returns content for startup display
func (m *CoachManager) GetStartupContent() *CoachDisplayContent {
	greeting := GetTimeBasedGreeting()
	icon := GetTimeBasedIcon()

	streakInfo := ""
	if m.profile.CurrentStreak > 0 {
		mult := StreakMultiplier(m.profile.CurrentStreak)
		streakInfo = "🔥 Day " + formatInt(m.profile.CurrentStreak) + " streak"
		if mult > 1.0 {
			streakInfo += " (" + formatFloat(mult) + "x XP)"
		}
	}

	content := streakInfo
	if m.todayStats != nil && m.todayStats.CommandsExecuted > 0 {
		content += "\n📊 Today: " + formatInt(m.todayStats.CommandsExecuted) + " commands"
		if m.todayStats.XPEarned > 0 {
			content += ", +" + formatInt(m.todayStats.XPEarned) + " XP"
		}
	}

	// Add daily challenge summary
	incomplete := 0
	for _, c := range m.dailyChallenges {
		if !c.Completed {
			incomplete++
		}
	}
	if incomplete > 0 {
		content += "\n🎯 " + formatInt(incomplete) + " daily challenges remaining"
	}

	return &CoachDisplayContent{
		Type:    "startup",
		Icon:    icon,
		Title:   greeting + ", " + m.profile.Username + "!",
		Content: content,
		Action:  "Type @!coach for your dashboard",
	}
}

// Helper methods

func (m *CoachManager) isAliasCommand(command string) bool {
	// Check if the first word is an alias
	words := strings.Fields(command)
	if len(words) == 0 {
		return false
	}
	// This is a simplified check - in practice you'd check runner.alias
	return false
}

func (m *CoachManager) getTotalCommands() int {
	var count int64
	m.db.Model(&history.HistoryEntry{}).Count(&count)
	return int(count)
}

func (m *CoachManager) countUniqueCommands() int {
	var count int64
	m.db.Model(&history.HistoryEntry{}).Distinct("command").Count(&count)
	return int(count)
}

func (m *CoachManager) countUniqueDirectories() int {
	today := time.Now().Format("2006-01-02")
	var count int64
	m.db.Model(&history.HistoryEntry{}).
		Where("DATE(created_at) = ?", today).
		Distinct("directory").Count(&count)
	return int(count)
}

func (m *CoachManager) countWeeklyCommands() int {
	weekAgo := time.Now().AddDate(0, 0, -7)
	var count int64
	m.db.Model(&history.HistoryEntry{}).
		Where("created_at > ?", weekAgo).Count(&count)
	return int(count)
}

func (m *CoachManager) countActiveDaysThisWeek() int {
	weekAgo := time.Now().AddDate(0, 0, -7)
	var dates []string
	m.db.Model(&history.HistoryEntry{}).
		Where("created_at > ?", weekAgo).
		Distinct("DATE(created_at)").
		Pluck("DATE(created_at)", &dates)
	return len(dates)
}

func (m *CoachManager) countTotalPipelines() int {
	var count int64
	m.db.Model(&history.HistoryEntry{}).
		Where("command LIKE ?", "%|%").Count(&count)
	return int(count)
}

func (m *CoachManager) countTotalAliasUsage() int {
	// This would need more sophisticated tracking
	return 0
}

func (m *CoachManager) getTotalTimeSaved() int {
	// This would need keystroke tracking
	return 0
}

func (m *CoachManager) countFastCommands(thresholdMs int) int {
	// This would need command duration tracking
	return 0
}

func (m *CoachManager) countGitCommits() int {
	var count int64
	m.db.Model(&history.HistoryEntry{}).
		Where("command LIKE ?", "git commit%").Count(&count)
	return int(count)
}

func getChallengeDefinition(id string) *ChallengeDefinition {
	for _, c := range DailyChallenges {
		if c.ID == id {
			return &c
		}
	}
	for _, c := range WeeklyChallenges {
		if c.ID == id {
			return &c
		}
	}
	for _, c := range SpecialChallenges {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

func getTitleForNotificationType(t string) string {
	switch t {
	case "achievement":
		return "Achievement Unlocked!"
	case "level_up":
		return "Level Up!"
	case "challenge":
		return "Challenge Complete!"
	case "streak":
		return "Streak Milestone!"
	default:
		return "Notification"
	}
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return formatInt(n/10) + string(rune('0'+n%10))
}

func formatFloat(f float64) string {
	whole := int(f)
	frac := int((f - float64(whole)) * 10)
	return formatInt(whole) + "." + string(rune('0'+frac))
}
