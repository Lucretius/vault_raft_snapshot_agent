package vault_raft_snapshot_agent

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/upload"
	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault"
	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault/auth"
	"github.com/stretchr/testify/assert"
)

func TestSnapshotterLocksTakeSnapshot(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{
		leader:          true,
		snapshotRuntime: time.Millisecond * 500,
	}
	uploaderStub := uploaderStub{}
	config := SnapshotConfig{
		Timeout: clientAPIStub.snapshotRuntime * 3,
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})

	start := time.Now()

	errs := make(chan error, 1)
	go func() {
		_, err := snapshotter.TakeSnapshot(context.Background())
		errs <- err
	}()

	go func() {
		_, err := snapshotter.TakeSnapshot(context.Background())
		errs <- err
	}()

	for i := 0; i < 2; i++ {
		err := <-errs
		assert.NoError(t, err, "TakeSnapshot failed unexpectedly")
	}

	assert.GreaterOrEqual(t, time.Since(start), clientAPIStub.snapshotRuntime*2, "TakeSnapshot did not prevent synchronous snapshots")
}

func TestSnapshotterLocksConfigure(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{
		leader:          true,
		snapshotRuntime: time.Millisecond * 500,
	}
	uploaderStub := uploaderStub{}
	config := SnapshotConfig{
		Timeout: clientAPIStub.snapshotRuntime * 3,
	}

	newConfig := SnapshotConfig{
		Frequency: time.Minute,
		Timeout:   time.Millisecond,
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})

	start := time.Now()

	errs := make(chan error, 1)
	running := make(chan bool, 1)
	go func() {
		running <- true
		_, err := snapshotter.TakeSnapshot(context.Background())
		errs <- err
	}()

	go func() {
		<-running
		snapshotter.Configure(newConfig, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})
		errs <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errs
		assert.NoError(t, err, "TakeSnapshot failed unexpectedly")
	}

	assert.GreaterOrEqual(t, time.Since(start), clientAPIStub.snapshotRuntime+250, "TakeSnapshot did not prevent re-configuration during snapshots")

	frequency, err := snapshotter.TakeSnapshot(context.Background())

	assert.NoError(t, err, "TakeSnapshot failed unexpectedly")
	assert.Equal(t, newConfig.Frequency, frequency, "Snaphotter did not re-configure propertly")
}

func TestSnapshotterAbortsAfterTimeout(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{
		leader:          true,
		snapshotRuntime: time.Second * 5,
	}
	uploaderStub := uploaderStub{}
	config := SnapshotConfig{
		Timeout: time.Second,
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})

	start := time.Now()

	errs := make(chan error, 1)
	go func() {
		_, err := snapshotter.TakeSnapshot(context.Background())
		errs <- err
	}()

	assert.NoErrorf(t, <-errs, "TakeSnapshot failed unexpectedly")
	// config.Timeout * 2 is quite less than clientAPIStub.snapshotRuntime
	// and big enough so that the test does not flicker
	assert.LessOrEqual(t, time.Since(start), config.Timeout*2, "TakeSnapshot did not abort at timeout")
}

func TestSnapshotterFailsIfSnapshottingFails(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{
		leader: false,
	}
	uploaderStub := uploaderStub{}
	config := SnapshotConfig{
		Timeout: time.Second,
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})

	_, err := snapshotter.TakeSnapshot(context.Background())

	assert.Error(t, err, "TakeSnaphot did not fail although snapshotting failed")
	assert.False(t, uploaderStub.uploaded, "TakeSnapshot uploaded although snapshotting failed")
}

func TestSnapshotterUploadsDataFromSnapshot(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{
		leader:       true,
		snapshotData: "test-snapshot",
	}
	uploaderStub := uploaderStub{}
	config := SnapshotConfig{
		Timeout:         time.Second,
		NamePrefix:      "test-",
		NameSuffix:      ".test",
		TimestampFormat: "2006-01-02T15-04Z-0700",
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})

	_, err := snapshotter.TakeSnapshot(context.Background())

	assert.NoError(t, err, "TakeSnaphot failed unexpectedly")
	assert.True(t, uploaderStub.uploaded, "TakeSnapshot did not upload")
	assert.Equal(t, clientAPIStub.snapshotData, uploaderStub.uploadData, "TakeSnapshot did upload false data")
	assert.Equal(t, config.NamePrefix, uploaderStub.uploadPrefix)
	assert.Equal(t, config.NameSuffix, uploaderStub.uploadSuffix)
	assert.Equal(t, time.Now().Format(config.TimestampFormat), uploaderStub.uploadTimestamp)
}

func TestSnapshotterContinuesUploadingIfUploadFails(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{
		leader:       true,
		snapshotData: "test-snapshot",
	}
	uploaderStub1 := uploaderStub{
		uploadFails: true,
	}
	uploaderStub2 := uploaderStub{
		uploadFails: false,
	}

	config := SnapshotConfig{
		Timeout: time.Second,
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub1, &uploaderStub2})

	_, err := snapshotter.TakeSnapshot(context.Background())
	assert.Error(t, err, "TakeSnaphot did not fail although one of the uploaders failed")

	assert.True(t, uploaderStub1.uploaded, "TakeSnapshot did not upload to first uploader")
	assert.True(t, uploaderStub2.uploaded, "TakeSnapshot did not upload to second uploader")
}

func TestSnapshotterReturnsFrequency(t *testing.T) {
	clientAPIStub := snapshotterVaultClientAPIStub{}
	uploaderStub := uploaderStub{}

	config := SnapshotConfig{
		Frequency: time.Minute,
	}

	snapshotter := Snapshotter{}
	snapshotter.Configure(config, vault.NewClient("http://127.0.0.1:8200", &clientAPIStub, nil), []upload.Uploader{&uploaderStub})

	frequency, _ := snapshotter.TakeSnapshot(context.Background())

	assert.Equal(t, config.Frequency, frequency)
}

type snapshotterVaultClientAPIStub struct {
	leader          bool
	snapshotRuntime time.Duration
	snapshotData    string
}

func (stub *snapshotterVaultClientAPIStub) TakeSnapshot(ctx context.Context, writer io.Writer) error {
	if stub.snapshotData != "" {
		if _, err := writer.Write([]byte(stub.snapshotData)); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
	case <-time.After(stub.snapshotRuntime):
	}

	return nil
}

func (stub *snapshotterVaultClientAPIStub) IsLeader() (bool, error) {
	return stub.leader, nil
}

func (stub *snapshotterVaultClientAPIStub) AuthAPI() auth.VaultAuthAPI {
	return nil
}

type uploaderStub struct {
	uploaded        bool
	uploadPrefix    string
	uploadTimestamp string
	uploadSuffix    string
	uploadData      string
	uploadFails     bool
}

func (stub *uploaderStub) Destination() string {
	return ""
}

func (stub *uploaderStub) Upload(ctx context.Context, reader io.Reader, prefix string, timestamp string, suffix string, retain int) error {
	stub.uploaded = true
	if stub.uploadFails {
		return errors.New("upload failed")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	stub.uploadData = string(data)
	stub.uploadPrefix = prefix
	stub.uploadTimestamp = timestamp
	stub.uploadSuffix = suffix
	return nil
}
