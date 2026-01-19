# VCS commit message prediction helpers.
# Provides context-aware commit message suggestions for git and jj (Jujutsu).

# Shared instructions for conventional commit format
export commitMessageInstructions = `
# Writing a command with a generated commit message
* Follow conventional commits format
* Always double-quote the commit message
* Format: "<type>(<scope>): <description>"
  * Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore
  * Scope is optional but recommended (e.g., the component or file affected)
  * Description should be imperative mood, lowercase, no period at end
* Keep the message under 72 characters
* Focus on WHAT changed and WHY, not HOW
* Analyze the diff to understand the actual changes made`

# =============================================================================
# Helper functions
# =============================================================================

# Check if a string contains a message flag (-m or --message)
# Handles: -m, --message, -am, -ma (combined flags for git)
tool __hasMessageFlag(input) {
    return input.includes(" -m ") || input.includes(" -m\"") || input.includes(" -m'") || input.includes(" --message ") || input.includes(" --message=") || input.includes(" -am ") || input.includes(" -am\"") || input.includes(" -am'") || input.includes(" -ma ") || input.includes(" -ma\"") || input.includes(" -ma'") || input.endsWith(" -m") || input.endsWith(" --message") || input.endsWith(" -am") || input.endsWith(" -ma")
}

# Check if a string contains the -a/--all flag for git commit
# Handles: -a, --all, -am, -ma (combined flags)
tool __hasAutoStageFlag(input) {
    return input.includes(" -a ") || input.includes(" -a\"") || input.includes(" --all ") || input.includes(" -am ") || input.includes(" -am\"") || input.includes(" -ma ") || input.includes(" -ma\"") || input.endsWith(" -a") || input.endsWith(" --all") || input.endsWith(" -am") || input.endsWith(" -ma")
}

# =============================================================================
# Git Support
# =============================================================================

# Check if input looks like a git commit message command
# Returns an object with { autoStage: bool } or null if not a commit message
export tool parseGitCommit(input) {
    if (input == null) {
        return null
    }
    
    # Check for "git commit " prefix and message flag
    if (!input.startsWith("git commit ")) {
        return null
    }
    
    if (!__hasMessageFlag(input)) {
        return null
    }
    
    # Check if -a/--all flag is present (auto-stage tracked files)
    autoStage = __hasAutoStageFlag(input)
    
    return { autoStage: autoStage }
}

# Get changes for git commit message context
# If autoStage is true, include both staged and unstaged changes (for -a flag)
# Otherwise get only staged changes
export tool getGitChanges(autoStage) {
    # -a/--all stages all tracked modified files, so we need both:
    # - git diff --staged (already staged)
    # - git diff (unstaged, which -a will auto-stage)
    # Using HEAD shows the combined diff of what will be committed
    diffFlag = "--staged"
    if (autoStage) {
        diffFlag = "HEAD"
    }
    
    # Get a summary of changes (files changed, insertions, deletions)
    statResult = exec(`git diff ${diffFlag} --stat 2>/dev/null`)
    if (statResult.exitCode != 0 || statResult.stdout == null || statResult.stdout.trim() == "") {
        return null
    }
    
    # Get the actual diff (limited to avoid huge outputs)
    diffResult = exec(`git diff ${diffFlag} -U15 2>/dev/null | head -1000`)
    if (diffResult.exitCode != 0 || diffResult.stdout == null || diffResult.stdout.trim() == "") {
        return statResult.stdout.trim()
    }
    
    return `${statResult.stdout.trim()}\n\n${diffResult.stdout.trim()}`
}

# =============================================================================
# Jujutsu (jj) Support
# =============================================================================

# Check if input looks like a jj commit/describe command with message flag
# Returns an object with {} or null if not a commit/describe message command
export tool parseJjCommitOrDescribe(input) {
    if (input == null) {
        return null
    }
    
    # Check for "jj commit " or "jj describe " prefix
    isCommit = input.startsWith("jj commit ")
    isDescribe = input.startsWith("jj describe ")
    
    if (!isCommit && !isDescribe) {
        return null
    }
    
    if (!__hasMessageFlag(input)) {
        return null
    }
    
    return {}
}

# Get changes for jj describe message context
# jj always shows the working copy changes (no staging area concept)
export tool getJjChanges() {
    # Get a summary of changes
    statResult = exec(`jj diff --stat 2>/dev/null`)
    if (statResult.exitCode != 0 || statResult.stdout == null || statResult.stdout.trim() == "") {
        return null
    }
    
    # Get the actual diff (limited to avoid huge outputs)
    diffResult = exec(`jj diff --context 15 2>/dev/null | head -1000`)
    if (diffResult.exitCode != 0 || diffResult.stdout == null || diffResult.stdout.trim() == "") {
        return statResult.stdout.trim()
    }
    
    return `${statResult.stdout.trim()}\n\n${diffResult.stdout.trim()}`
}

# =============================================================================
# Unified Interface
# =============================================================================

# Fast check if input is any VCS commit/describe message command (no diff execution)
# Returns true if it matches a VCS commit message pattern, false otherwise
export tool isVcsCommitMessage(input) {
    if (parseGitCommit(input) != null) {
        return true
    }
    if (parseJjCommitOrDescribe(input) != null) {
        return true
    }
    return false
}

# Check if input is any VCS commit/describe message command AND get changes
# Returns { vcs: "git" | "jj", changes: string | null } or null if not a commit message
# WARNING: This runs expensive diff commands - only call when you need the changes!
export tool parseVcsCommitMessage(input) {
    # Try git
    gitInfo = parseGitCommit(input)
    if (gitInfo != null) {
        changes = getGitChanges(gitInfo.autoStage)
        return { vcs: "git", changes: changes }
    }

    # Try jj (commit or describe)
    jjInfo = parseJjCommitOrDescribe(input)
    if (jjInfo != null) {
        changes = getJjChanges()
        return { vcs: "jj", changes: changes }
    }

    return null
}
