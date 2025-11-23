package appupdate

import (
	"bytes"
	"context"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"io"
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
	resultChannel := make(chan string)

	currentSemVer, err := semver.NewVersion(currentVersion)
	if err != nil {
		logger.Debug("running a dev build, skipping self-update check")
		close(resultChannel)
		return resultChannel
	}

	// Check if we have previously detected a newer version
	updateToLatestVersion(currentSemVer, logger, fs, prompter, updater)

	// Check for newer versions from remote repository
	go fetchAndSaveLatestVersion(resultChannel, logger, fs, updater)

	return resultChannel
}

func readLatestVersion(fs filesystem.FileSystem) string {
	file, err := fs.Open(core.LatestVersionFile())
	if err != nil {
		return ""
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(buf.String())
}

func updateToLatestVersion(currentSemVer *semver.Version, logger *zap.Logger, fs filesystem.FileSystem, prompter core.UserPrompter, updater Updater) {
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

	confirm, _ := prompter.Prompt(
		styles.AGENT_QUESTION("New version of gsh available. Update now? (Y/n) "),
		[]string{},
		latestVersion,
		nil,
		nil,
		nil,
		logger,
		gline.NewOptions(),
	)

	if strings.ToLower(confirm) == "n" {
		return
	}

	latest, found, err := updater.DetectLatest(
		context.Background(),
		"atinylittleshell/gsh",
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

func fetchAndSaveLatestVersion(resultChannel chan string, logger *zap.Logger, fs filesystem.FileSystem, updater Updater) {
	defer close(resultChannel)

	latest, found, err := updater.DetectLatest(
		context.Background(),
		"atinylittleshell/gsh",
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
	file, err := fs.Create(recordFilePath)
	if err != nil {
		logger.Error("failed to save latest version", zap.Error(err))
		return
	}
	defer file.Close()

	_, err = file.WriteString(latest.Version())
	if err != nil {
		logger.Error("failed to save latest version", zap.Error(err))
		return
	}

	resultChannel <- latest.Version()
}
