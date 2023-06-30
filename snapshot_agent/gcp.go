package snapshot_agent

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	"google.golang.org/api/iterator"
)

type GCPUploader struct {
	gcpBucket *storage.BucketHandle
	retain    int64
}

func NewGCPUploader(config *config.Configuration) (*GCPUploader, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	bucket := client.Bucket(config.GCP.Bucket)

	return &GCPUploader{gcpBucket: bucket}, nil
}

func (u *GCPUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64) (string, error) {
	fileName := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)
	obj := u.gcpBucket.Object(fileName)
	w := obj.NewWriter(context.Background())

	_, err := io.Copy(w, reader)
	if err != nil {
		return "", fmt.Errorf("error writing snapshot to GCP: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("error closing GCP writer: %w", err)
	}

	if u.retain > 0 {

		existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, "raft_snapshot-")

		if err != nil {
			return "", fmt.Errorf("error getting existing snapshots: %w", err)
		}

		if len(existingSnapshots)-int(u.retain) <= 0 {
			return fileName, nil
		}
		snapshotsToDelete := existingSnapshots[0 : len(existingSnapshots)-int(u.retain)]

		for _, ss := range snapshotsToDelete {
			obj := u.gcpBucket.Object(ss.Name)
			err := obj.Delete(ctx)
			if err != nil {
				return "", fmt.Errorf("error deleting snapshot: %w", err)
			}
		}
	}
	return fileName, nil
}

func (u *GCPUploader) LastSuccessfulUpload(ctx context.Context) (time.Time, error) {
	existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, "raft_snapshot-")

	if err != nil {
		return time.Time{}, fmt.Errorf("error getting existing snapshots: %w", err)
	}

	if len(existingSnapshots) == 0 {
		return time.Time{}, nil
	}

	lastSnapshot := existingSnapshots[len(existingSnapshots)-1]

	return lastSnapshot.Updated, nil
}

func (u *GCPUploader) listUploadedSnapshotsAscending(ctx context.Context, keyPrefix string) ([]storage.ObjectAttrs, error) {

	var result []storage.ObjectAttrs

	query := &storage.Query{Prefix: keyPrefix}
	it := u.gcpBucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return result, fmt.Errorf("unable to iterate bucket: %w", err)
		}
		result = append(result, *attrs)
	}

	timestamp := func(o1, o2 *storage.ObjectAttrs) bool {
		return o1.Updated.Before(o2.Updated)
	}

	GCPBy(timestamp).Sort(result)

	return result, nil
}

// implementation of Sort interface for s3 objects
type GCPBy func(f1, f2 *storage.ObjectAttrs) bool

func (by GCPBy) Sort(objects []storage.ObjectAttrs) {
	fs := &gcpObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type gcpObjectSorter struct {
	objects []storage.ObjectAttrs
	by      func(f1, f2 *storage.ObjectAttrs) bool // Closure used in the Less method.
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
