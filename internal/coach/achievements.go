package coach

// AchievementTier represents the tier/rarity of an achievement
type AchievementTier string

const (
	TierBronze   AchievementTier = "bronze"
	TierSilver   AchievementTier = "silver"
	TierGold     AchievementTier = "gold"
	TierPlatinum AchievementTier = "platinum"
	TierDiamond  AchievementTier = "diamond"
)

// AchievementCategory represents the category of an achievement
type AchievementCategory string

const (
	CategorySpeed       AchievementCategory = "speed"
	CategoryAccuracy    AchievementCategory = "accuracy"
	CategoryLearning    AchievementCategory = "learning"
	CategoryStreak      AchievementCategory = "streak"
	CategoryProductivity AchievementCategory = "productivity"
	CategoryTimeBased   AchievementCategory = "time_based"
	CategoryMilestone   AchievementCategory = "milestone"
	CategoryGit         AchievementCategory = "git"
	CategorySpecial     AchievementCategory = "special"
)

// AchievementDefinition defines a single achievement
type AchievementDefinition struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Category    AchievementCategory
	Tier        AchievementTier
	Requirement int
	XPReward    int
	Secret      bool   // Hidden until unlocked
	Metric      string // What to measure
}

// TierXPRewards returns XP for each tier
func TierXPRewards() map[AchievementTier]int {
	return map[AchievementTier]int{
		TierBronze:   100,
		TierSilver:   250,
		TierGold:     500,
		TierPlatinum: 1000,
		TierDiamond:  2000,
	}
}

// AllAchievements contains all achievement definitions
var AllAchievements = []AchievementDefinition{
	// ====== SPEED ACHIEVEMENTS ======
	{ID: "speed_quick_start", Name: "Quick Start", Description: "Execute 10 commands under 100ms", Icon: "⚡", Category: CategorySpeed, Tier: TierBronze, Requirement: 10, XPReward: 100, Metric: "fast_commands_100ms"},
	{ID: "speed_getting_faster", Name: "Getting Faster", Description: "Average command time under 200ms for a day", Icon: "🏃", Category: CategorySpeed, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "day_avg_under_200ms"},
	{ID: "speed_quick_draw", Name: "Quick Draw", Description: "Execute 100 commands under 50ms average", Icon: "🔫", Category: CategorySpeed, Tier: TierGold, Requirement: 100, XPReward: 500, Metric: "fast_commands_50ms"},
	{ID: "speed_lightning", Name: "Lightning Fingers", Description: "Type at 80+ WPM in commands", Icon: "⚡", Category: CategorySpeed, Tier: TierGold, Requirement: 80, XPReward: 500, Metric: "typing_wpm"},
	{ID: "speed_millisecond", Name: "Millisecond Master", Description: "1000 commands under 30ms", Icon: "⏱️", Category: CategorySpeed, Tier: TierPlatinum, Requirement: 1000, XPReward: 1000, Metric: "fast_commands_30ms"},
	{ID: "speed_time_lord", Name: "Time Lord", Description: "Save 1 hour cumulative via aliases", Icon: "⏰", Category: CategorySpeed, Tier: TierDiamond, Requirement: 3600, XPReward: 2000, Metric: "time_saved_seconds"},
	{ID: "speed_blink", Name: "Blink of an Eye", Description: "Execute a command in under 5ms", Icon: "👁️", Category: CategorySpeed, Tier: TierPlatinum, Requirement: 1, XPReward: 1000, Metric: "command_under_5ms"},
	{ID: "speed_rapid_fire", Name: "Rapid Fire", Description: "Execute 50 commands in 10 minutes", Icon: "🔥", Category: CategorySpeed, Tier: TierSilver, Requirement: 50, XPReward: 250, Metric: "commands_in_10min"},

	// ====== ACCURACY ACHIEVEMENTS ======
	{ID: "accuracy_steady", Name: "Steady Aim", Description: "80% accuracy over 50 commands", Icon: "🎯", Category: CategoryAccuracy, Tier: TierBronze, Requirement: 50, XPReward: 100, Metric: "accuracy_80_over_50"},
	{ID: "accuracy_careful", Name: "Careful Typist", Description: "10 commands in a row without error", Icon: "✍️", Category: CategoryAccuracy, Tier: TierBronze, Requirement: 10, XPReward: 100, Metric: "consecutive_success"},
	{ID: "accuracy_sharp", Name: "Sharp Eye", Description: "90% accuracy over 200 commands", Icon: "👁️", Category: CategoryAccuracy, Tier: TierSilver, Requirement: 200, XPReward: 250, Metric: "accuracy_90_over_200"},
	{ID: "accuracy_consistent", Name: "Consistent", Description: "25 commands in a row without error", Icon: "📊", Category: CategoryAccuracy, Tier: TierSilver, Requirement: 25, XPReward: 250, Metric: "consecutive_success"},
	{ID: "accuracy_sharpshooter", Name: "Sharpshooter", Description: "95% accuracy over 1000 commands", Icon: "🎯", Category: CategoryAccuracy, Tier: TierGold, Requirement: 1000, XPReward: 500, Metric: "accuracy_95_over_1000"},
	{ID: "accuracy_perfect_session", Name: "Perfect Session", Description: "100 commands without error in one session", Icon: "💯", Category: CategoryAccuracy, Tier: TierGold, Requirement: 100, XPReward: 500, Metric: "session_no_errors_100"},
	{ID: "accuracy_surgical", Name: "Surgical Precision", Description: "98% accuracy over 5000 commands", Icon: "🔬", Category: CategoryAccuracy, Tier: TierPlatinum, Requirement: 5000, XPReward: 1000, Metric: "accuracy_98_over_5000"},
	{ID: "accuracy_perfect_week", Name: "Perfect Week", Description: "7 days with zero command errors", Icon: "📅", Category: CategoryAccuracy, Tier: TierPlatinum, Requirement: 7, XPReward: 1000, Metric: "error_free_days"},
	{ID: "accuracy_flawless", Name: "Flawless", Description: "99% accuracy over 10,000 commands", Icon: "💎", Category: CategoryAccuracy, Tier: TierDiamond, Requirement: 10000, XPReward: 2000, Metric: "accuracy_99_over_10000"},
	{ID: "accuracy_infallible", Name: "Infallible", Description: "1000 commands in a row without error", Icon: "🏆", Category: CategoryAccuracy, Tier: TierDiamond, Requirement: 1000, XPReward: 2000, Metric: "consecutive_success"},

	// ====== LEARNING ACHIEVEMENTS ======
	{ID: "learn_curious", Name: "Curious Cat", Description: "Use 10 unique commands", Icon: "🐱", Category: CategoryLearning, Tier: TierBronze, Requirement: 10, XPReward: 100, Metric: "unique_commands"},
	{ID: "learn_help_seeker", Name: "Help Seeker", Description: "Access --help for 5 commands", Icon: "❓", Category: CategoryLearning, Tier: TierBronze, Requirement: 5, XPReward: 100, Metric: "help_accessed"},
	{ID: "learn_explorer", Name: "Explorer", Description: "Navigate to 10 different directories", Icon: "🧭", Category: CategoryLearning, Tier: TierBronze, Requirement: 10, XPReward: 100, Metric: "unique_directories"},
	{ID: "learn_collector", Name: "Command Collector", Description: "Use 50 unique commands", Icon: "📚", Category: CategoryLearning, Tier: TierSilver, Requirement: 50, XPReward: 250, Metric: "unique_commands"},
	{ID: "learn_polyglot", Name: "Polyglot", Description: "Use 10 different CLI tools", Icon: "🗣️", Category: CategoryLearning, Tier: TierSilver, Requirement: 10, XPReward: 250, Metric: "unique_tools"},
	{ID: "learn_manual", Name: "Manual Reader", Description: "Access --help for 25 commands", Icon: "📖", Category: CategoryLearning, Tier: TierSilver, Requirement: 25, XPReward: 250, Metric: "help_accessed"},
	{ID: "learn_connoisseur", Name: "Command Connoisseur", Description: "Use 100 unique commands", Icon: "🎓", Category: CategoryLearning, Tier: TierGold, Requirement: 100, XPReward: 500, Metric: "unique_commands"},
	{ID: "learn_tool_master", Name: "Tool Master", Description: "Use 25 different CLI tools", Icon: "🛠️", Category: CategoryLearning, Tier: TierGold, Requirement: 25, XPReward: 500, Metric: "unique_tools"},
	{ID: "learn_encyclopedia", Name: "Living Encyclopedia", Description: "Use 200 unique commands", Icon: "📚", Category: CategoryLearning, Tier: TierPlatinum, Requirement: 200, XPReward: 1000, Metric: "unique_commands"},
	{ID: "learn_swiss_army", Name: "Swiss Army Knife", Description: "Use 50 different CLI tools", Icon: "🔧", Category: CategoryLearning, Tier: TierPlatinum, Requirement: 50, XPReward: 1000, Metric: "unique_tools"},
	{ID: "learn_deity", Name: "Command Deity", Description: "Use 500 unique commands", Icon: "👑", Category: CategoryLearning, Tier: TierDiamond, Requirement: 500, XPReward: 2000, Metric: "unique_commands"},

	// ====== STREAK ACHIEVEMENTS ======
	{ID: "streak_started", Name: "Getting Started", Description: "3-day usage streak", Icon: "🌱", Category: CategoryStreak, Tier: TierBronze, Requirement: 3, XPReward: 100, Metric: "current_streak"},
	{ID: "streak_habit", Name: "Habit Forming", Description: "5-day usage streak", Icon: "🌿", Category: CategoryStreak, Tier: TierBronze, Requirement: 5, XPReward: 100, Metric: "current_streak"},
	{ID: "streak_weekly", Name: "Weekly Warrior", Description: "7-day usage streak", Icon: "🔥", Category: CategoryStreak, Tier: TierBronze, Requirement: 7, XPReward: 100, Metric: "current_streak"},
	{ID: "streak_dedicated", Name: "Dedicated", Description: "14-day usage streak", Icon: "💪", Category: CategoryStreak, Tier: TierSilver, Requirement: 14, XPReward: 250, Metric: "current_streak"},
	{ID: "streak_committed", Name: "Committed", Description: "21-day usage streak", Icon: "🎯", Category: CategoryStreak, Tier: TierSilver, Requirement: 21, XPReward: 250, Metric: "current_streak"},
	{ID: "streak_monthly", Name: "Monthly Master", Description: "30-day usage streak", Icon: "🏆", Category: CategoryStreak, Tier: TierGold, Requirement: 30, XPReward: 500, Metric: "current_streak"},
	{ID: "streak_unstoppable", Name: "Unstoppable", Description: "60-day usage streak", Icon: "⚡", Category: CategoryStreak, Tier: TierGold, Requirement: 60, XPReward: 500, Metric: "current_streak"},
	{ID: "streak_relentless", Name: "Relentless", Description: "90-day usage streak", Icon: "🔥", Category: CategoryStreak, Tier: TierPlatinum, Requirement: 90, XPReward: 1000, Metric: "current_streak"},
	{ID: "streak_quarterly", Name: "Quarterly Champion", Description: "100-day usage streak", Icon: "💎", Category: CategoryStreak, Tier: TierPlatinum, Requirement: 100, XPReward: 1000, Metric: "current_streak"},
	{ID: "streak_half_year", Name: "Half-Year Hero", Description: "180-day usage streak", Icon: "🌟", Category: CategoryStreak, Tier: TierPlatinum, Requirement: 180, XPReward: 1000, Metric: "current_streak"},
	{ID: "streak_legendary", Name: "Legendary", Description: "365-day usage streak", Icon: "👑", Category: CategoryStreak, Tier: TierDiamond, Requirement: 365, XPReward: 2000, Metric: "current_streak"},

	// ====== PRODUCTIVITY ACHIEVEMENTS ======
	{ID: "prod_alias_apprentice", Name: "Alias Apprentice", Description: "Create your first alias", Icon: "🔗", Category: CategoryProductivity, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "aliases_created"},
	{ID: "prod_pipeline_beginner", Name: "Pipeline Beginner", Description: "Use your first pipe command", Icon: "🔀", Category: CategoryProductivity, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "pipelines_used"},
	{ID: "prod_time_saver", Name: "Time Saver", Description: "Save 1 minute via aliases", Icon: "⏱️", Category: CategoryProductivity, Tier: TierBronze, Requirement: 60, XPReward: 100, Metric: "time_saved_seconds"},
	{ID: "prod_alias_artisan", Name: "Alias Artisan", Description: "Use aliases 100 times", Icon: "🎨", Category: CategoryProductivity, Tier: TierSilver, Requirement: 100, XPReward: 250, Metric: "alias_usage_count"},
	{ID: "prod_pipeline_pro", Name: "Pipeline Pro", Description: "Use 100 piped commands", Icon: "🔀", Category: CategoryProductivity, Tier: TierSilver, Requirement: 100, XPReward: 250, Metric: "pipelines_used"},
	{ID: "prod_efficiency", Name: "Efficiency Expert", Description: "Save 30 minutes via aliases", Icon: "💎", Category: CategoryProductivity, Tier: TierSilver, Requirement: 1800, XPReward: 250, Metric: "time_saved_seconds"},
	{ID: "prod_alias_architect", Name: "Alias Architect", Description: "Use aliases 500 times", Icon: "🏗️", Category: CategoryProductivity, Tier: TierGold, Requirement: 500, XPReward: 500, Metric: "alias_usage_count"},
	{ID: "prod_pipeline_wizard", Name: "Pipeline Wizard", Description: "Use 500 piped commands", Icon: "🧙", Category: CategoryProductivity, Tier: TierGold, Requirement: 500, XPReward: 500, Metric: "pipelines_used"},
	{ID: "prod_hour_saver", Name: "Hour Saver", Description: "Save 1 hour via efficiency features", Icon: "⏰", Category: CategoryProductivity, Tier: TierGold, Requirement: 3600, XPReward: 500, Metric: "time_saved_seconds"},
	{ID: "prod_alias_legend", Name: "Alias Legend", Description: "Use aliases 2000 times", Icon: "🌟", Category: CategoryProductivity, Tier: TierPlatinum, Requirement: 2000, XPReward: 1000, Metric: "alias_usage_count"},
	{ID: "prod_pipeline_god", Name: "Pipeline God", Description: "Use 2000 piped commands", Icon: "⚡", Category: CategoryProductivity, Tier: TierPlatinum, Requirement: 2000, XPReward: 1000, Metric: "pipelines_used"},
	{ID: "prod_automation", Name: "Automation Wizard", Description: "Save 100,000 keystrokes", Icon: "🤖", Category: CategoryProductivity, Tier: TierDiamond, Requirement: 100000, XPReward: 2000, Metric: "keystrokes_saved"},
	{ID: "prod_deity", Name: "Productivity Deity", Description: "Save 100 hours cumulative", Icon: "👑", Category: CategoryProductivity, Tier: TierDiamond, Requirement: 360000, XPReward: 2000, Metric: "time_saved_seconds"},

	// ====== TIME-BASED ACHIEVEMENTS ======
	{ID: "time_night_owl", Name: "Night Owl", Description: "50 commands after midnight", Icon: "🦉", Category: CategoryTimeBased, Tier: TierBronze, Requirement: 50, XPReward: 100, Metric: "commands_after_midnight"},
	{ID: "time_early_bird", Name: "Early Bird", Description: "50 commands before 6 AM", Icon: "🐦", Category: CategoryTimeBased, Tier: TierBronze, Requirement: 50, XPReward: 100, Metric: "commands_before_6am"},
	{ID: "time_weekend", Name: "Weekend Coder", Description: "Active on 5 weekends", Icon: "📅", Category: CategoryTimeBased, Tier: TierBronze, Requirement: 5, XPReward: 100, Metric: "active_weekends"},
	{ID: "time_nocturnal", Name: "Nocturnal", Description: "500 commands after midnight", Icon: "🌙", Category: CategoryTimeBased, Tier: TierSilver, Requirement: 500, XPReward: 250, Metric: "commands_after_midnight"},
	{ID: "time_dawn", Name: "Dawn Patrol", Description: "500 commands before 6 AM", Icon: "🌅", Category: CategoryTimeBased, Tier: TierSilver, Requirement: 500, XPReward: 250, Metric: "commands_before_6am"},
	{ID: "time_weekend_warrior", Name: "Weekend Warrior", Description: "Active on 26 weekends", Icon: "⚔️", Category: CategoryTimeBased, Tier: TierSilver, Requirement: 26, XPReward: 250, Metric: "active_weekends"},
	{ID: "time_night_shift", Name: "Night Shift", Description: "5000 commands after midnight", Icon: "🌃", Category: CategoryTimeBased, Tier: TierGold, Requirement: 5000, XPReward: 500, Metric: "commands_after_midnight"},
	{ID: "time_all_nighter", Name: "All-Nighter", Description: "Active for 8+ hours in one session", Icon: "☕", Category: CategoryTimeBased, Tier: TierGold, Requirement: 1, XPReward: 500, Metric: "session_8_hours"},
	{ID: "time_vampire", Name: "Vampire", Description: "50% of commands after midnight", Icon: "🧛", Category: CategoryTimeBased, Tier: TierPlatinum, Requirement: 50, XPReward: 1000, Metric: "night_command_percentage"},
	{ID: "time_24_7", Name: "24/7", Description: "Commands in all 24 hours of a single day", Icon: "🕐", Category: CategoryTimeBased, Tier: TierPlatinum, Requirement: 24, XPReward: 1000, Metric: "all_hours_in_day"},
	{ID: "time_timeless", Name: "Timeless", Description: "Active every hour of every day in a week", Icon: "♾️", Category: CategoryTimeBased, Tier: TierDiamond, Requirement: 168, XPReward: 2000, Metric: "all_hours_in_week"},

	// ====== MILESTONE ACHIEVEMENTS ======
	{ID: "milestone_hello", Name: "Hello World", Description: "Execute your first command", Icon: "👋", Category: CategoryMilestone, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "total_commands"},
	{ID: "milestone_10", Name: "Getting Going", Description: "Execute 10 commands", Icon: "🚶", Category: CategoryMilestone, Tier: TierBronze, Requirement: 10, XPReward: 100, Metric: "total_commands"},
	{ID: "milestone_50", Name: "Warming Up", Description: "Execute 50 commands", Icon: "🏃", Category: CategoryMilestone, Tier: TierBronze, Requirement: 50, XPReward: 100, Metric: "total_commands"},
	{ID: "milestone_100", Name: "Centurion", Description: "Execute 100 commands", Icon: "💯", Category: CategoryMilestone, Tier: TierSilver, Requirement: 100, XPReward: 250, Metric: "total_commands"},
	{ID: "milestone_500", Name: "Active User", Description: "Execute 500 commands", Icon: "⭐", Category: CategoryMilestone, Tier: TierSilver, Requirement: 500, XPReward: 250, Metric: "total_commands"},
	{ID: "milestone_1000", Name: "Regular", Description: "Execute 1,000 commands", Icon: "🎖️", Category: CategoryMilestone, Tier: TierSilver, Requirement: 1000, XPReward: 250, Metric: "total_commands"},
	{ID: "milestone_5000", Name: "Thousand Club", Description: "Execute 5,000 commands", Icon: "🏅", Category: CategoryMilestone, Tier: TierGold, Requirement: 5000, XPReward: 500, Metric: "total_commands"},
	{ID: "milestone_10000", Name: "Power User", Description: "Execute 10,000 commands", Icon: "💪", Category: CategoryMilestone, Tier: TierGold, Requirement: 10000, XPReward: 500, Metric: "total_commands"},
	{ID: "milestone_25000", Name: "Dedicated", Description: "Execute 25,000 commands", Icon: "🌟", Category: CategoryMilestone, Tier: TierGold, Requirement: 25000, XPReward: 500, Metric: "total_commands"},
	{ID: "milestone_50000", Name: "Command Veteran", Description: "Execute 50,000 commands", Icon: "🎖️", Category: CategoryMilestone, Tier: TierPlatinum, Requirement: 50000, XPReward: 1000, Metric: "total_commands"},
	{ID: "milestone_100000", Name: "Master", Description: "Execute 100,000 commands", Icon: "👑", Category: CategoryMilestone, Tier: TierPlatinum, Requirement: 100000, XPReward: 1000, Metric: "total_commands"},
	{ID: "milestone_250000", Name: "Terminal Titan", Description: "Execute 250,000 commands", Icon: "🏆", Category: CategoryMilestone, Tier: TierDiamond, Requirement: 250000, XPReward: 2000, Metric: "total_commands"},
	{ID: "milestone_500000", Name: "Shell Legend", Description: "Execute 500,000 commands", Icon: "🌟", Category: CategoryMilestone, Tier: TierDiamond, Requirement: 500000, XPReward: 2000, Metric: "total_commands"},
	{ID: "milestone_1000000", Name: "The One", Description: "Execute 1,000,000 commands", Icon: "💎", Category: CategoryMilestone, Tier: TierDiamond, Requirement: 1000000, XPReward: 2000, Metric: "total_commands"},

	// ====== GIT ACHIEVEMENTS ======
	{ID: "git_first_commit", Name: "First Commit", Description: "Make your first git commit", Icon: "📝", Category: CategoryGit, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "git_commits"},
	{ID: "git_branch_explorer", Name: "Branch Explorer", Description: "Create your first branch", Icon: "🌿", Category: CategoryGit, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "git_branches_created"},
	{ID: "git_status_checker", Name: "Status Checker", Description: "Run git status 50 times", Icon: "📊", Category: CategoryGit, Tier: TierBronze, Requirement: 50, XPReward: 100, Metric: "git_status_count"},
	{ID: "git_commit_regular", Name: "Commit Regular", Description: "Make 100 commits", Icon: "💾", Category: CategoryGit, Tier: TierSilver, Requirement: 100, XPReward: 250, Metric: "git_commits"},
	{ID: "git_branch_manager", Name: "Branch Manager", Description: "Create 20 branches", Icon: "🌳", Category: CategoryGit, Tier: TierSilver, Requirement: 20, XPReward: 250, Metric: "git_branches_created"},
	{ID: "git_merge_master", Name: "Merge Master", Description: "Successfully merge 25 branches", Icon: "🔀", Category: CategoryGit, Tier: TierSilver, Requirement: 25, XPReward: 250, Metric: "git_merges"},
	{ID: "git_commit_champion", Name: "Commit Champion", Description: "Make 500 commits", Icon: "🏆", Category: CategoryGit, Tier: TierGold, Requirement: 500, XPReward: 500, Metric: "git_commits"},
	{ID: "git_rebase_warrior", Name: "Rebase Warrior", Description: "Successfully rebase 50 times", Icon: "⚔️", Category: CategoryGit, Tier: TierGold, Requirement: 50, XPReward: 500, Metric: "git_rebases"},
	{ID: "git_conflict_resolver", Name: "Conflict Resolver", Description: "Resolve 25 merge conflicts", Icon: "🔧", Category: CategoryGit, Tier: TierGold, Requirement: 25, XPReward: 500, Metric: "git_conflicts_resolved"},
	{ID: "git_guru", Name: "Git Guru", Description: "Make 2000 commits", Icon: "🧙", Category: CategoryGit, Tier: TierPlatinum, Requirement: 2000, XPReward: 1000, Metric: "git_commits"},
	{ID: "git_history_rewriter", Name: "History Rewriter", Description: "Use interactive rebase 100 times", Icon: "📜", Category: CategoryGit, Tier: TierPlatinum, Requirement: 100, XPReward: 1000, Metric: "git_interactive_rebases"},
	{ID: "git_cherry_picker", Name: "Cherry Picker", Description: "Cherry-pick 100 commits", Icon: "🍒", Category: CategoryGit, Tier: TierPlatinum, Requirement: 100, XPReward: 1000, Metric: "git_cherry_picks"},
	{ID: "git_god", Name: "Version Control God", Description: "Make 10,000 commits", Icon: "👑", Category: CategoryGit, Tier: TierDiamond, Requirement: 10000, XPReward: 2000, Metric: "git_commits"},
	{ID: "git_ninja", Name: "Git Ninja", Description: "Use 50 different git subcommands", Icon: "🥷", Category: CategoryGit, Tier: TierDiamond, Requirement: 50, XPReward: 2000, Metric: "unique_git_subcommands"},

	// ====== SPECIAL & SECRET ACHIEVEMENTS ======
	{ID: "special_sudo_sandwich", Name: "sudo make me a sandwich", Description: "Use sudo appropriately", Icon: "🥪", Category: CategorySpecial, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "sudo_sandwich", Secret: true},
	{ID: "special_over_9000", Name: "It's Over 9000!", Description: "Execute command #9001", Icon: "💥", Category: CategorySpecial, Tier: TierSilver, Requirement: 9001, XPReward: 250, Metric: "total_commands", Secret: true},
	{ID: "special_lucky_7", Name: "Lucky 7", Description: "Execute 777 commands", Icon: "🎰", Category: CategorySpecial, Tier: TierSilver, Requirement: 777, XPReward: 250, Metric: "total_commands", Secret: true},
	{ID: "special_nice", Name: "Nice", Description: "Execute the 69th command of the day", Icon: "😏", Category: CategorySpecial, Tier: TierBronze, Requirement: 69, XPReward: 100, Metric: "daily_command_69", Secret: true},
	{ID: "special_binary", Name: "Binary Master", Description: "Execute 1024 commands in a day", Icon: "🔢", Category: CategorySpecial, Tier: TierGold, Requirement: 1024, XPReward: 500, Metric: "daily_commands", Secret: true},
	{ID: "special_challenge", Name: "Challenge Accepted", Description: "Complete your first challenge", Icon: "🎯", Category: CategorySpecial, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "challenges_completed"},
	{ID: "special_weekly_victor", Name: "Weekly Victor", Description: "Complete all weekly challenges", Icon: "🏆", Category: CategorySpecial, Tier: TierGold, Requirement: 1, XPReward: 500, Metric: "all_weekly_challenges"},
	{ID: "special_challenge_addict", Name: "Challenge Addict", Description: "Complete 100 challenges", Icon: "🎮", Category: CategorySpecial, Tier: TierPlatinum, Requirement: 100, XPReward: 1000, Metric: "challenges_completed"},
	{ID: "special_perfectionist", Name: "Perfectionist", Description: "Complete 30 daily challenges in a row", Icon: "✨", Category: CategorySpecial, Tier: TierDiamond, Requirement: 30, XPReward: 2000, Metric: "consecutive_daily_challenges"},
	{ID: "special_self_aware", Name: "Self Aware", Description: "View your own achievements", Icon: "🪞", Category: CategorySpecial, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "achievements_viewed"},
	{ID: "special_stat_nerd", Name: "Stat Nerd", Description: "View stats 50 times", Icon: "📊", Category: CategorySpecial, Tier: TierSilver, Requirement: 50, XPReward: 250, Metric: "stats_viewed"},
	{ID: "special_reporter", Name: "Reporter", Description: "View weekly report 10 times", Icon: "📰", Category: CategorySpecial, Tier: TierSilver, Requirement: 10, XPReward: 250, Metric: "reports_viewed"},
	{ID: "special_sl", Name: "Choo Choo!", Description: "Run the sl command", Icon: "🚂", Category: CategorySpecial, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "sl_command", Secret: true},
	{ID: "special_cowsay", Name: "Moo!", Description: "Use the cowsay command", Icon: "🐄", Category: CategorySpecial, Tier: TierBronze, Requirement: 1, XPReward: 100, Metric: "cowsay_command", Secret: true},
	{ID: "special_fortune_cow", Name: "Wise Cow", Description: "Use fortune | cowsay pipeline", Icon: "🔮", Category: CategorySpecial, Tier: TierSilver, Requirement: 1, XPReward: 250, Metric: "fortune_cowsay", Secret: true},
	{ID: "special_new_year", Name: "New Year's Resolution", Description: "First command of the new year", Icon: "🎆", Category: CategorySpecial, Tier: TierGold, Requirement: 1, XPReward: 500, Metric: "new_year_command", Secret: true},
	{ID: "special_midnight", Name: "Midnight Coder", Description: "Execute command exactly at midnight", Icon: "🕛", Category: CategorySpecial, Tier: TierSilver, Requirement: 1, XPReward: 250, Metric: "midnight_command", Secret: true},
	{ID: "special_friday_deploy", Name: "Friday Deploy", Description: "Deploy on a Friday (brave!)", Icon: "💀", Category: CategorySpecial, Tier: TierGold, Requirement: 1, XPReward: 500, Metric: "friday_deploy", Secret: true},
	{ID: "special_hacktoberfest", Name: "Hacktoberfest Hero", Description: "50+ git commands in October", Icon: "🎃", Category: CategorySpecial, Tier: TierGold, Requirement: 50, XPReward: 500, Metric: "october_git_commands"},
}

// GetAchievementByID returns an achievement definition by ID
func GetAchievementByID(id string) *AchievementDefinition {
	for _, a := range AllAchievements {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

// GetAchievementsByCategory returns all achievements in a category
func GetAchievementsByCategory(category AchievementCategory) []AchievementDefinition {
	var result []AchievementDefinition
	for _, a := range AllAchievements {
		if a.Category == category {
			result = append(result, a)
		}
	}
	return result
}

// GetAchievementsByTier returns all achievements of a tier
func GetAchievementsByTier(tier AchievementTier) []AchievementDefinition {
	var result []AchievementDefinition
	for _, a := range AllAchievements {
		if a.Tier == tier {
			result = append(result, a)
		}
	}
	return result
}

// GetVisibleAchievements returns non-secret achievements
func GetVisibleAchievements() []AchievementDefinition {
	var result []AchievementDefinition
	for _, a := range AllAchievements {
		if !a.Secret {
			result = append(result, a)
		}
	}
	return result
}
