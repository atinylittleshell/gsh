package appupdate

import (
	"bytes"
	"context"

	"os"
	"testing"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) Open(name string) (*os.File, error) {
	args := m.Called(name)
	return args.Get(0).(*os.File), args.Error(1)
}

func (m *MockFileSystem) Create(name string) (*os.File, error) {
	args := m.Called(name)
	return args.Get(0).(*os.File), args.Error(1)
}

func (m *MockFileSystem) ReadFile(name string) (string, error) {
	args := m.Called(name)
	return args.String(0), args.Error(1)
}

func (m *MockFileSystem) WriteFile(name, content string) error {
	return m.Called(name, content).Error(0)
}

type MockFile struct {
	mock.Mock
	bytes.Buffer
}

func (m *MockFile) Close() error {
	return m.Called().Error(0)
}

type MockPrompter struct {
	mock.Mock
}

func (m *MockPrompter) Prompt(
	prompt string,
	historyValues []string,
	explanation string,
	predictor gline.Predictor,
	explainer gline.Explainer,
	analytics gline.PredictionAnalytics,
	logger *zap.Logger,
	options gline.Options,
) (string, error) {
	args := m.Called(prompt, historyValues, explanation, predictor, explainer, logger, options)
	return args.String(0), args.Error(1)
}

type MockUpdater struct {
	mock.Mock
}

func (m *MockUpdater) DetectLatest(ctx context.Context, repo string) (Release, bool, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).(Release), args.Bool(1), args.Error(2)
}

func (m *MockUpdater) UpdateTo(ctx context.Context, assetURL, assetName, exePath string) error {
	args := m.Called(ctx, assetURL, assetName, exePath)
	return args.Error(0)
}

type MockRelease struct {
	mock.Mock
}

func (m *MockRelease) Version() string {
	return m.Called().String(0)
}

func (m *MockRelease) AssetURL() string {
	return m.Called().String(0)
}

func (m *MockRelease) AssetName() string {
	return m.Called().String(0)
}

func TestReadLatestVersion(t *testing.T) {
	mockFS := new(MockFileSystem)
	mockFile, _ := os.CreateTemp("", "test-latest-version")
	defer os.Remove(mockFile.Name())

	mockFile.Write([]byte("1.2.3"))
	mockFile.Seek(0, 0)
	mockFS.On("Open", core.LatestVersionFile()).Return(mockFile, nil)

	result := readLatestVersion(mockFS)
	assert.Equal(t, "1.2.3", result)

	mockFS.AssertExpectations(t)
}

func TestHandleSelfUpdate_UpdateNeeded(t *testing.T) {
	mockFS := new(MockFileSystem)
	mockPrompter := new(MockPrompter)
	mockUpdater := new(MockUpdater)
	mockRemoteRelease := new(MockRelease)
	logger := zap.NewNop()

	mockFileForRead, _ := os.CreateTemp("", "test-latest-version-read")
	defer os.Remove(mockFileForRead.Name())
	mockFileForRead.Write([]byte("1.1.0"))
	mockFileForRead.Seek(0, 0)

	mockFileForWrite, _ := os.CreateTemp("", "test-latest-version-write")
	defer os.Remove(mockFileForWrite.Name())

	mockFS.On("Open", core.LatestVersionFile()).Return(mockFileForRead, nil)
	mockFS.On("Create", core.LatestVersionFile()).Return(mockFileForWrite, nil)

	// Use minor version bump instead of major to test normal update flow
	mockRemoteRelease.On("Version").Return("1.2.0")
	mockRemoteRelease.On("AssetURL").Return("https://github.com/test/url")
	mockRemoteRelease.On("AssetName").Return("test")

	mockUpdater.On("DetectLatest", mock.Anything, "atinylittleshell/gsh").Return(mockRemoteRelease, true, nil)
	mockUpdater.On("UpdateTo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockPrompter.
		On("Prompt", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("y", nil)

	resultChannel := HandleSelfUpdate("1.0.0", logger, mockFS, mockPrompter, mockUpdater)

	remoteVersion, ok := <-resultChannel

	assert.Equal(t, true, ok)
	assert.Equal(t, "1.2.0", remoteVersion)

	mockFS.AssertExpectations(t)
	mockRemoteRelease.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestHandleSelfUpdate_NoUpdateNeeded(t *testing.T) {
	mockFS := new(MockFileSystem)
	mockPrompter := new(MockPrompter)
	mockUpdater := new(MockUpdater)
	mockRemoteRelease := new(MockRelease)
	logger := zap.NewNop()

	mockFileForRead, _ := os.CreateTemp("", "test-latest-version-read")
	defer os.Remove(mockFileForRead.Name())
	mockFileForRead.Write([]byte("1.2.3"))
	mockFileForRead.Seek(0, 0)

	mockFileForWrite, _ := os.CreateTemp("", "test-latest-version-write")
	defer os.Remove(mockFileForWrite.Name())

	mockFS.On("Open", core.LatestVersionFile()).Return(mockFileForRead, nil)
	mockFS.On("Create", core.LatestVersionFile()).Return(mockFileForWrite, nil)

	mockRemoteRelease.On("Version").Return("1.2.4")
	mockUpdater.On("DetectLatest", mock.Anything, "atinylittleshell/gsh").Return(mockRemoteRelease, true, nil)

	resultChannel := HandleSelfUpdate("2.0.0", logger, mockFS, mockPrompter, mockUpdater)

	remoteVersion, ok := <-resultChannel

	assert.Equal(t, true, ok)
	assert.Equal(t, "1.2.4", remoteVersion)

	mockFS.AssertExpectations(t)
	mockRemoteRelease.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)

	mockPrompter.AssertNotCalled(t, "Prompt")
}

func TestHandleSelfUpdate_MajorVersionBoundary(t *testing.T) {
	mockFS := new(MockFileSystem)
	mockPrompter := new(MockPrompter)
	mockUpdater := new(MockUpdater)
	mockRemoteRelease := new(MockRelease)
	logger := zap.NewNop()

	mockFileForRead, _ := os.CreateTemp("", "test-latest-version-read")
	defer os.Remove(mockFileForRead.Name())
	mockFileForRead.Write([]byte("1.0.0"))
	mockFileForRead.Seek(0, 0)

	mockFileForWrite, _ := os.CreateTemp("", "test-latest-version-write")
	defer os.Remove(mockFileForWrite.Name())

	mockFS.On("Open", core.LatestVersionFile()).Return(mockFileForRead, nil)
	mockFS.On("Create", core.LatestVersionFile()).Return(mockFileForWrite, nil)

	// Major version bump: v0.9.0 -> v1.0.0
	mockRemoteRelease.On("Version").Return("1.0.0")
	mockUpdater.On("DetectLatest", mock.Anything, "atinylittleshell/gsh").Return(mockRemoteRelease, true, nil)

	resultChannel := HandleSelfUpdate("0.9.0", logger, mockFS, mockPrompter, mockUpdater)

	remoteVersion, ok := <-resultChannel

	assert.Equal(t, true, ok)
	assert.Equal(t, "1.0.0", remoteVersion)

	mockFS.AssertExpectations(t)
	mockRemoteRelease.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)

	// Should NOT prompt or update across major version boundary
	mockPrompter.AssertNotCalled(t, "Prompt")
	mockUpdater.AssertNotCalled(t, "UpdateTo")
}
