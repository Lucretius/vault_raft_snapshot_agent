package vault_raft_snapshot_agent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// CreateGCPSnapshot writes snapshot to google storage
func (s *Snapshotter) CreateGCPSnapshot(b *bytes.Buffer, config *Configuration, currentTs int64) (string, error) {
	fileName := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)
	obj := s.GCPBucket.Object(fileName)
	w := obj.NewWriter(context.Background())

	if _, err := w.Write(b.Bytes()); err != nil {
		return "", err
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	if config.Retain > 0 {
		deleteCtx := context.Background()
		query := &storage.Query{Prefix: "raft_snapshot-"}
		it := s.GCPBucket.Objects(deleteCtx, query)
		var files []storage.ObjectAttrs
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Println("Unable to iterate through bucket to find old snapshots to delete")
				return fileName, err
			}
			files = append(files, *attrs)
		}

		timestamp := func(o1, o2 *storage.ObjectAttrs) bool {
			return o1.Updated.Before(o2.Updated)
		}

		GCPBy(timestamp).Sort(files)
		if len(files)-int(config.Retain) <= 0 {
			return fileName, nil
		}
		snapshotsToDelete := files[0 : len(files)-int(config.Retain)]

		for _, ss := range snapshotsToDelete {
			obj := s.GCPBucket.Object(ss.Name)
			err := obj.Delete(deleteCtx)
			if err != nil {
				log.Println("Cannot delete old snapshot")
				return fileName, err
			}
		}
	}
	return fileName, nil
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
