package upload

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

type LocalUploaderConfig struct {
	Path  string `validate:"required_if=Empty false,omitempty,dir"`
	Empty bool
}

type localUploaderImpl struct {
	path string
}

func createLocalUploader(ctx context.Context, config LocalUploaderConfig) (uploader[os.FileInfo], error) {
	return uploader[os.FileInfo]{
		localUploaderImpl{
			path: config.Path,
		},
	}, nil
}

func (u localUploaderImpl) Destination() string {
	return fmt.Sprintf("local path %s", u.path)
}

func (u localUploaderImpl) uploadSnapshot(ctx context.Context, name string, data io.Reader) error {
	fileName := fmt.Sprintf("%s/%s", u.path, name)

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer func() {
		_ = file.Close()
	}()

	if _, err = io.Copy(file, data); err != nil {
		return err
	}

	return nil
}

func (u localUploaderImpl) deleteSnapshot(ctx context.Context, snapshot os.FileInfo) error {
	if err := os.Remove(fmt.Sprintf("%s/%s", u.path, snapshot.Name())); err != nil {
		return err
	}

	return nil
}

func (u localUploaderImpl) listSnapshots(ctx context.Context, prefix string, ext string) ([]os.FileInfo, error) {
	var snapshots []os.FileInfo

	files, err := os.ReadDir(u.path)
	if err != nil {
		return snapshots, err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) && strings.HasSuffix(file.Name(), ext) {
			info, err := file.Info()
			if err != nil {
				return snapshots, err
			}
			snapshots = append(snapshots, info)
		}
	}

	return snapshots, nil
}

func (u localUploaderImpl) compareSnapshots(a, b os.FileInfo) int {
	return a.ModTime().Compare(b.ModTime())
}
