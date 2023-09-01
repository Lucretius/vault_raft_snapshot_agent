package upload

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCPConfig struct {
	Bucket string `validate:"required_if=Empty false"`
	Empty  bool
}

type gcpUploaderImpl struct {
	destination string
	bucket      *storage.BucketHandle
}

func createGCPUploader(config GCPConfig) (*uploader[storage.ObjectAttrs], error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &uploader[storage.ObjectAttrs]{
		gcpUploaderImpl{
			destination: fmt.Sprintf("gcp bucket %s", config.Bucket),
			bucket:      client.Bucket(config.Bucket),
		},
	}, nil
}

func (u gcpUploaderImpl) Destination() string {
	return u.destination
}

// nolint:unused
// implements interface uploaderImpl
func (u gcpUploaderImpl) uploadSnapshot(ctx context.Context, name string, data io.Reader) error {
	obj := u.bucket.Object(name)
	w := obj.NewWriter(context.Background())

	if _, err := io.Copy(w, data); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u gcpUploaderImpl) deleteSnapshot(ctx context.Context, snapshot storage.ObjectAttrs) error {
	obj := u.bucket.Object(snapshot.Name)
	if err := obj.Delete(ctx); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u gcpUploaderImpl) listSnapshots(ctx context.Context, prefix string, ext string) ([]storage.ObjectAttrs, error) {
	var result []storage.ObjectAttrs

	query := &storage.Query{Prefix: prefix}
	it := u.bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return result, err
		}
		result = append(result, *attrs)
	}

	return result, nil
}

// nolint:unused
// implements interface uploaderImpl
func (u gcpUploaderImpl) compareSnapshots(a, b storage.ObjectAttrs) int {
	return a.Updated.Compare(b.Updated)
}
