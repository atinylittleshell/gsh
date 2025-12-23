package appupdate

import (
	"bytes"
	"context"

	"os"
	"testing"

	"github.com/atinylittleshell/gsh/internal/core"
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
	mockUpdater := new(MockUpdater)
	mockRemoteRelease := new(MockRelease)
	logger := zap.NewNop()

	mockFileForWrite, _ := os.CreateTemp("", "test-latest-version-write")
	defer os.Remove(mockFileForWrite.Name())

	mockFS.On("Create", core.LatestVersionFile()).Return(mockFileForWrite, nil)

	mockRemoteRelease.On("Version").Return("2.0.0")

	mockUpdater.On("DetectLatest", mock.Anything, "atinylittleshell/gsh").Return(mockRemoteRelease, true, nil)

	resultChannel := HandleSelfUpdate("0.1.0", logger, mockFS, mockUpdater)

	remoteVersion, ok := <-resultChannel

	assert.Equal(t, true, ok)
	assert.Equal(t, "2.0.0", remoteVersion)

	mockFS.AssertExpectations(t)
	mockRemoteRelease.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestHandleSelfUpdate_NoUpdateNeeded(t *testing.T) {
	mockFS := new(MockFileSystem)
	mockUpdater := new(MockUpdater)
	mockRemoteRelease := new(MockRelease)
	logger := zap.NewNop()

	mockRemoteRelease.On("Version").Return("1.2.4")
	mockUpdater.On("DetectLatest", mock.Anything, "atinylittleshell/gsh").Return(mockRemoteRelease, true, nil)

	resultChannel := HandleSelfUpdate("2.0.0", logger, mockFS, mockUpdater)

	// Should not receive anything since version is not newer
	remoteVersion, ok := <-resultChannel

	assert.Equal(t, false, ok)
	assert.Equal(t, "", remoteVersion)

	mockFS.AssertExpectations(t)
	mockRemoteRelease.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

