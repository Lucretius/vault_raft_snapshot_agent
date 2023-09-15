package upload

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
)

type UploadersConfig struct {
	AWS   AWSUploaderConfig   `default:"{\"Empty\": true}"`
	Azure AzureUploaderConfig `default:"{\"Empty\": true}"`
	GCP   GCPUploaderConfig   `default:"{\"Empty\": true}"`
	Local LocalUploaderConfig `default:"{\"Empty\": true}"`
	Swift SwiftUploaderConfig `default:"{\"Empty\": true}"`
}

type Uploader interface {
	Destination() string
	Upload(ctx context.Context, snapshot io.Reader, prefix string, timestamp string, suffix string, retain int) error
}

func CreateUploaders(ctx context.Context, config UploadersConfig) ([]Uploader, error) {
	var uploaders []Uploader

	if !config.AWS.Empty {
		aws, err := createAWSUploader(ctx, config.AWS)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, aws)
	}

	if !config.Azure.Empty {
		azure, err := createAzureUploader(ctx, config.Azure)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, azure)
	}

	if !config.GCP.Empty {
		gcp, err := createGCPUploader(ctx, config.GCP)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, gcp)
	}

	if !config.Local.Empty {
		local, err := createLocalUploader(ctx, config.Local)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, local)
	}

	if !config.Swift.Empty {
		local, err := createSwiftUploader(ctx, config.Swift)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, local)
	}

	return uploaders, nil
}

type uploaderImpl[T any] interface {
	uploadSnapshot(ctx context.Context, name string, data io.Reader) error
	deleteSnapshot(ctx context.Context, snapshot T) error
	listSnapshots(ctx context.Context, prefix string, ext string) ([]T, error)
	compareSnapshots(a, b T) int
}

type uploader[T any] struct {
	impl uploaderImpl[T]
}

func (u uploader[T]) Destination() string {
	return ""
}

func (u uploader[T]) Upload(ctx context.Context, snapshot io.Reader, prefix string, timestamp string, suffix string, retain int) error {
	name := strings.Join([]string{prefix, timestamp, suffix}, "")
	if err := u.impl.uploadSnapshot(ctx, name, snapshot); err != nil {
		return fmt.Errorf("error uploading snapshot to %s: %w", u.Destination(), err)
	}

	if retain > 0 {
		return u.deleteSnapshots(ctx, prefix, suffix, retain)
	}

	return nil
}

func (u uploader[T]) deleteSnapshots(ctx context.Context, prefix string, suffix string, retain int) error {
	snapshots, err := u.impl.listSnapshots(ctx, prefix, suffix)
	if err != nil {
		return fmt.Errorf("error getting snapshots from %s: %w", u.Destination(), err)
	}

	if len(snapshots) > retain {
		slices.SortFunc(snapshots, func(a, b T) int { return u.impl.compareSnapshots(a, b) * -1 })

		for _, s := range snapshots[retain:] {
			if err := u.impl.deleteSnapshot(ctx, s); err != nil {
				return fmt.Errorf("error deleting snapshot from %s: %w", u.Destination(), err)
			}
		}
	}
	return nil
}
