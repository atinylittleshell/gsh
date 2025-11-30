package appupdate

import (
	"bytes"
	"context"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"io"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/styles"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"github.com/creativeprojects/go-selfupdate"
	"go.uber.org/zap"
)

func HandleSelfUpdate(
	currentVersion string,
	logger *zap.Logger,
	fs filesystem.FileSystem,
	prompter core.UserPrompter,
	updater Updater,
) chan string {
	const repoSlug = "robottwo/gsh_prime"

	resultChannel := make(chan string)

	// Trim any whitespace or newlines from version strings
	currentVersion = strings.TrimSpace(currentVersion)

	currentSemVer, err := semver.NewVersion(currentVersion)
	if err != nil {
		logger.Debug("running a dev build, skipping self-update check")
		close(resultChannel)
		return resultChannel
	}

	// Check if we have previously detected a newer version
	updateToLatestVersion(repoSlug, currentSemVer, logger, fs, prompter, updater)

	// Check for newer versions from remote repository
	go fetchAndSaveLatestVersion(repoSlug, resultChannel, logger, fs, updater)

	return resultChannel
}

func readLatestVersion(fs filesystem.FileSystem) string {
	file, err := fs.Open(core.LatestVersionFile())
	if err != nil {
		return ""
	}
	defer func() {
		_ = file.Close()
	}()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(buf.String())
}

func updateToLatestVersion(repoSlug string, currentSemVer *semver.Version, logger *zap.Logger, fs filesystem.FileSystem, prompter core.UserPrompter, updater Updater) {
	latestVersion := readLatestVersion(fs)
	if latestVersion == "" {
		return
	}

	latestSemVer, err := semver.NewVersion(latestVersion)
	if err != nil {
		logger.Error("failed to parse latest version", zap.Error(err))
		return
	}
	if latestSemVer.LessThanEqual(currentSemVer) {
		return
	}

	// Check GSH_DEFAULT_TO_YES environment variable
	defaultToYes := strings.ToLower(os.Getenv("GSH_DEFAULT_TO_YES"))
	isDefaultYes := defaultToYes == "1" || defaultToYes == "true"

	promptText := "New version of gsh available. Update now? (y/N) "
	if isDefaultYes {
		promptText = "New version of gsh available. Update now? (Y/n) "
	}

	confirm, _ := prompter.Prompt(
		styles.AGENT_QUESTION(promptText),
		[]string{},
		latestVersion,
		nil,
		nil,
		nil,
		logger,
		gline.NewOptions(),
	)

	confirmLower := strings.ToLower(strings.TrimSpace(confirm))
	// If default is yes, only "n" or "no" declines. If default is no, only "y" or "yes" confirms.
	if isDefaultYes {
		if confirmLower == "n" || confirmLower == "no" {
			return
		}
	} else {
		if confirmLower != "y" && confirmLower != "yes" {
			return
		}
	}

	latest, found, err := updater.DetectLatest(
		context.Background(),
		repoSlug,
	)
	if err != nil {
		logger.Warn("error occurred while detecting latest version", zap.Error(err))
		return
	}
	if !found {
		logger.Warn("latest version could not be detected")
		return
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		logger.Error("failed to get executable path to update", zap.Error(err))
		return
	}
	if err := updater.UpdateTo(context.Background(), latest.AssetURL(), latest.AssetName(), exe); err != nil {
		logger.Error("failed to update to latest version", zap.Error(err))
		return
	}

	logger.Info("successfully updated to latest version", zap.String("version", latest.Version()))
}

func fetchAndSaveLatestVersion(repoSlug string, resultChannel chan string, logger *zap.Logger, fs filesystem.FileSystem, updater Updater) {
	defer close(resultChannel)

	latest, found, err := updater.DetectLatest(
		context.Background(),
		repoSlug,
	)
	if err != nil {
		logger.Warn("error occurred while getting latest version from remote", zap.Error(err))
		return
	}
	if !found {
		logger.Warn("latest version could not be found")
		return
	}

	recordFilePath := core.LatestVersionFile()

	// Use OpenFile to set specific permissions (0600)
	file, err := fs.OpenFile(recordFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Error("failed to save latest version", zap.Error(err))
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Trim whitespace before writing
	versionToWrite := strings.TrimSpace(latest.Version())
	_, err = file.WriteString(versionToWrite)
	if err != nil {
		logger.Error("failed to save latest version", zap.Error(err))
		return
	}

	resultChannel <- latest.Version()
}
