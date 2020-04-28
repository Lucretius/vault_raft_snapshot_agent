package snapshot_agent

import (
	"fmt"
	"io"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// CreateS3Snapshot writes snapshot to s3 location
func (s *Snapshotter) CreateS3Snapshot(reader io.ReadWriter, config *config.Configuration, currentTs int64) (string, error) {
	keyPrefix := "raft_snapshots"
	if config.AWS.KeyPrefix != "" {
		keyPrefix = config.AWS.KeyPrefix
	}

	input := &s3manager.UploadInput{
		Bucket:               &config.AWS.Bucket,
		Key:                  aws.String(fmt.Sprintf("%s/raft_snapshot-%d.snap", keyPrefix, currentTs)),
		Body:                 reader,
		ServerSideEncryption: aws.String("AES256"),
	}

	if config.AWS.SSE == false {
		input.ServerSideEncryption = nil
	}

	if config.AWS.StaticSnapshotName != "" {
		input.Key = aws.String(fmt.Sprintf("%s/%s.snap", keyPrefix, config.AWS.StaticSnapshotName))
	}

	o, err := s.Uploader.Upload(input)
	if err != nil {
		return "", err
	} else {
		if config.Retain > 0 {
			// deal with retain logic later
		}
		return o.Location, nil
	}
}
