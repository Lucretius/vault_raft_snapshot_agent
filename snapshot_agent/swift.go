package snapshot_agent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	"github.com/ncw/swift/v2"
)

func (s *Snapshotter) CreateSwiftSnapshot(b *bytes.Buffer, config *config.Configuration, currentTs int64) (string, error) {
	ctx := context.Background()
	fileName := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)

	_, header, _ := s.SwiftConnection.Container(ctx, config.Swift.Container)

	object, err := s.SwiftConnection.ObjectCreate(ctx, config.Swift.Container, fileName, false, "", "", header)
	if err != nil {
		log.Fatal("Can't create Object in Swift: " + err.Error())
	}

	if _, err := object.Write(b.Bytes()); err != nil {
		return "", err
	}

	if err := object.Close(); err != nil {
		return "", err
	}

	if config.Retain > 0 {
		deleteCtx := context.Background()
		opts := &swift.ObjectsOpts{Prefix: "raft_snapshot-"}

		objects, err := s.SwiftConnection.ObjectsAll(deleteCtx, config.Swift.Container, opts)
		if err != nil {
			return "", err
		}

		timestamp := func(obj1, obj2 *swift.Object) bool {
			return obj1.LastModified.Before(obj2.LastModified)
		}

		SwiftBy(timestamp).Sort(objects)
		if len(objects)-int(config.Retain) <= 0 {
			return fileName, nil
		}

		objectsToDelete := objects[0 : len(objects)-int(config.Retain)]
		for _, objToDelete := range objectsToDelete {
			log.Printf("Delete snapshot: " + objToDelete.Name)
			err := s.SwiftConnection.ObjectDelete(deleteCtx, config.Swift.Container, objToDelete.Name)
			if err != nil {
				log.Println("Cannot delete snapshot")
				return fileName, err
			}
		}
	}

	return fileName, nil
}

type SwiftBy func(f1, f2 *swift.Object) bool

func (by SwiftBy) Sort(objects []swift.Object) {
	fs := &swiftObjectSorter{
		objects: objects,
		by:      by,
	}
	sort.Sort(fs)
}

type swiftObjectSorter struct {
	objects []swift.Object
	by      func(f1, f2 *swift.Object) bool
}

func (s *swiftObjectSorter) Len() int {
	return len(s.objects)
}

func (s *swiftObjectSorter) Less(i, j int) bool {
	return s.by(&s.objects[i], &s.objects[j])
}

func (s *swiftObjectSorter) Swap(i, j int) {
	s.objects[i], s.objects[j] = s.objects[j], s.objects[i]
}
