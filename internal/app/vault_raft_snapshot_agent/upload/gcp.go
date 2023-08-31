package upload

import (
	"context"
	"fmt"
	"io"
	"sort"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCPConfig struct {
	Bucket string `validate:"required_if=Empty false"`
	Empty bool
}

type gcpUploader struct {
	gcpBucket *storage.BucketHandle
}

func newGCPUploader(config GCPConfig) (*gcpUploader, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &gcpUploader{
		client.Bucket(config.Bucket),
	}, nil
}

func (u *gcpUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64, retain int) error {
	fileName := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)
	obj := u.gcpBucket.Object(fileName)
	w := obj.NewWriter(context.Background())

	_, err := io.Copy(w, reader)
	if err != nil {
		return fmt.Errorf("error writing snapshot to gcp: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("error closing gcp writer: %w", err)
	}

	if retain > 0 {

		existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, "raft_snapshot-")

		if err != nil {
			return fmt.Errorf("error getting existing snapshots from gcp: %w", err)
		}

		if len(existingSnapshots)-int(retain) <= 0 {
			return nil
		}
		snapshotsToDelete := existingSnapshots[0 : len(existingSnapshots)-int(retain)]

		for _, ss := range snapshotsToDelete {
			obj := u.gcpBucket.Object(ss.Name)
			err := obj.Delete(ctx)
			if err != nil {
				return fmt.Errorf("error deleting snapshot from gcp: %w", err)
			}
		}
	}
	return nil
}

func (u *gcpUploader) listUploadedSnapshotsAscending(ctx context.Context, keyPrefix string) ([]storage.ObjectAttrs, error) {

	var result []storage.ObjectAttrs

	query := &storage.Query{Prefix: keyPrefix}
	it := u.gcpBucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return result, fmt.Errorf("unable to iterate bucket on gcp: %w", err)
		}
		result = append(result, *attrs)
	}

	timestamp := func(o1, o2 *storage.ObjectAttrs) bool {
		return o1.Updated.Before(o2.Updated)
	}

	gcpBy(timestamp).Sort(result)

	return result, nil
}

// implementation of Sort interface for s3 objects
type gcpBy func(f1, f2 *storage.ObjectAttrs) bool

func (by gcpBy) Sort(objects []storage.ObjectAttrs) {
	fs := &gcpObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type gcpObjectSorter struct {
	objects []storage.ObjectAttrs
	by      gcpBy
}

func (s *gcpObjectSorter) Len() int {
	return len(s.objects)
}

func (s *gcpObjectSorter) Less(i, j int) bool {
	return s.by(&s.objects[i], &s.objects[j])
}

func (s *gcpObjectSorter) Swap(i, j int) {
	s.objects[i], s.objects[j] = s.objects[j], s.objects[i]
}
