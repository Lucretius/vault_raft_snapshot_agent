package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
)

func TestUploaderUpload(t *testing.T) {
	implStub := uploaderImplStub{}
	uploader := uploader[int]{&implStub}
	snapshotData := []byte("test")

	ctx := context.Background()
	err := uploader.Upload(ctx, bytes.NewReader(snapshotData), "test-", "time", ".snap", 0)

	assert.NoError(t, err, "Upload failed unexpectedly")
	assert.Equal(t, ctx, implStub.uploadCtx)
	assert.Equal(t, snapshotData, implStub.uploadData)
	assert.Equal(t, "test-time.snap", implStub.uploadName)
}

func TestUploaderDeletesSnapshotsIfRetainIsSet(t *testing.T) {
	implStub := uploaderImplStub{snapshots: []int{3, 1, 4, 2}}
	uploader := uploader[int]{&implStub}

	err := uploader.Upload(context.Background(), &bytes.Buffer{}, "", "", "", 2)

	assert.NoError(t, err, "Upload failed unexpectedly")
	assert.Equal(t, []int{4, 3}, implStub.snapshots)
}

func TestUploaderUploadFailsIfImplUploadFails(t *testing.T) {
	implStub := uploaderImplStub{snapshots: []int{3, 1}, uploadFails: true}
	uploader := uploader[int]{&implStub}

	err := uploader.Upload(context.Background(), &bytes.Buffer{}, "", "", "", 1)

	assert.Error(t, err, "Upload did not fail although implementation failed")
	assert.True(t, implStub.uploaded)
	assert.False(t, implStub.listed)
	assert.False(t, implStub.deleted)
}

func TestUploaderUploadFailsIfImplListFails(t *testing.T) {
	implStub := uploaderImplStub{snapshots: []int{3, 1}, listFails: true}
	uploader := uploader[int]{&implStub}

	err := uploader.Upload(context.Background(), &bytes.Buffer{}, "", "", "", 1)

	assert.Error(t, err, "Upload did not fail although implementation failed")
	assert.True(t, implStub.uploaded)
	assert.True(t, implStub.listed)
	assert.False(t, implStub.deleted)
}

func TestUploaderUploadFailsIfImplDeleteFails(t *testing.T) {
	implStub := uploaderImplStub{snapshots: []int{3, 1}, deleteFails: true}
	uploader := uploader[int]{&implStub}

	err := uploader.Upload(context.Background(), &bytes.Buffer{}, "", "", "", 1)

	assert.Error(t, err, "Upload did not fail although implementation failed")
	assert.True(t, implStub.uploaded)
	assert.True(t, implStub.listed)
	assert.True(t, implStub.deleted)
}

type uploaderImplStub struct {
	uploadFails bool
	deleteFails bool
	listFails   bool
	uploaded    bool
	listed      bool
	deleted     bool
	snapshots   []int
	uploadCtx   context.Context
	uploadName  string
	uploadData  []byte
}

// nolint:unused
// implements interface uploaderImpl
func (stub *uploaderImplStub) uploadSnapshot(ctx context.Context, name string, reader io.Reader) error {
	stub.uploaded = true
	if stub.uploadFails {
		return errors.New("upload failed")
	}
	stub.uploadCtx = ctx
	stub.uploadName = name
	bytes, _ := io.ReadAll(reader)
	stub.uploadData = bytes
	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (stub *uploaderImplStub) deleteSnapshot(ctx context.Context, snapshot int) error {
	stub.deleted = true
	if stub.deleteFails {
		return errors.New("delete failed")
	}
	stub.snapshots = funk.FilterInt(stub.snapshots, func(x int) bool { return x != snapshot })
	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (stub *uploaderImplStub) listSnapshots(ctx context.Context, prefix string, ext string) ([]int, error) {
	stub.listed = true
	if stub.listFails {
		return []int{}, errors.New("list failed")
	}

	return stub.snapshots, nil
}

// nolint:unused
// implements interface uploaderImpl
func (stub *uploaderImplStub) compareSnapshots(a, b int) int {
	return b - a
}
