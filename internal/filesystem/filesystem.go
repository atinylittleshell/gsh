package filesystem

import "os"

type FileSystem interface {
	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	ReadFile(name string) (string, error)
	WriteFile(name string, content string) error
}

type DefaultFileSystem struct{}

func (fs DefaultFileSystem) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (fs DefaultFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (fs DefaultFileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (fs DefaultFileSystem) ReadFile(name string) (string, error) {
	content, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (fs DefaultFileSystem) WriteFile(name string, content string) error {
	return os.WriteFile(name, []byte(content), 0644)
}
