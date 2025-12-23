package appupdate

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"go.uber.org/zap"
)

func HandleSelfUpdate(
	currentVersion string,
	logger *zap.Logger,
	fs filesystem.FileSystem,
	updater Updater,
) chan string {
	resultChannel := make(chan string)

	currentSemVer, err := semver.NewVersion(currentVersion)
	if err != nil {
		logger.Debug("running a dev build, skipping self-update check")
		close(resultChannel)
		return resultChannel
	}

	// Check for newer versions from remote repository
	go fetchAndSaveLatestVersion(resultChannel, logger, fs, updater, currentSemVer)

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

func fetchAndSaveLatestVersion(resultChannel chan string, logger *zap.Logger, fs filesystem.FileSystem, updater Updater, currentSemVer *semver.Version) {
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

	// Check if there's a newer version
	latestSemVer, err := semver.NewVersion(latest.Version())
	if err != nil {
		logger.Error("failed to parse latest version", zap.Error(err))
		return
	}

	if latestSemVer.LessThanEqual(currentSemVer) {
		logger.Debug("already running the latest version")
		return
	}

	// Save the latest version for notification
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

	logger.Info("new version available", zap.String("current", currentSemVer.String()), zap.String("latest", latest.Version()))
	resultChannel <- latest.Version()
}
