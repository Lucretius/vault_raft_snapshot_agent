package snapshot_agent

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Uploader struct {
	uploader *manager.Uploader
	s3Client *s3.Client
	config   config.S3Config
	retain   int64
}

func NewS3Uploader(config *config.Configuration) (*S3Uploader, error) {

	s3Config, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(config.AWS.Region),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load default aws config: %w", err)
	}

	if config.AWS.AccessKeyID != "" && config.AWS.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(config.AWS.AccessKeyID, config.AWS.SecretAccessKey, "")
		s3Config.Credentials = creds
	}

	if config.AWS.Endpoint != "" {
		s3Config.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: config.AWS.Endpoint,
			}, nil
		})
	}

	s3Client := s3.NewFromConfig(s3Config, func(o *s3.Options) {
		o.UsePathStyle = config.AWS.S3ForcePathStyle
	})

	s3Uploader := &S3Uploader{
		s3Client: s3Client,
		uploader: manager.NewUploader(s3Client),
		config:   config.AWS,
		retain:   config.Retain,
	}

	return s3Uploader, nil
}

func (u *S3Uploader) Upload(ctx context.Context, reader io.Reader, currentTs int64) (string, error) {
	keyPrefix := u.keyPrefix()

	input := &s3.PutObjectInput{
		Bucket: &u.config.Bucket,
		Key:    aws.String(fmt.Sprintf("%sraft_snapshot-%d.snap", keyPrefix, currentTs)),
		Body:   reader,
	}

	if u.config.SSE == true {
		input.ServerSideEncryption = s3Types.ServerSideEncryptionAes256
	}

	o, err := u.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error uploading snapshot: %w", err)
	} else {
		if u.retain > 0 {

			existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, keyPrefix)

			if err != nil {
				return "", fmt.Errorf("error getting existing snapshots: %w", err)
			}

			if len(existingSnapshots)-int(u.retain) <= 0 {
				return o.Location, nil
			}
			snapshotsToDelete := existingSnapshots[0 : len(existingSnapshots)-int(u.retain)]

			for i := range snapshotsToDelete {
				_, err := u.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &u.config.Bucket,
					Key:    snapshotsToDelete[i].Key,
				})
				if err != nil {
					return "", fmt.Errorf("error deleting snapshot %s: %w", *snapshotsToDelete[i].Key, err)
				}
			}
		}
		return o.Location, nil
	}
}

func (u *S3Uploader) LastSuccessfulUpload(ctx context.Context) (time.Time, error) {
	keyPrefix := u.keyPrefix()

	existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, keyPrefix)

	if err != nil {
		return time.Time{}, fmt.Errorf("error getting existing snapshots: %w", err)
	}

	if len(existingSnapshots) == 0 {
		return time.Time{}, nil
	}

	lastSnapshot := existingSnapshots[len(existingSnapshots)-1]

	return *lastSnapshot.LastModified, nil
}

func (u *S3Uploader) keyPrefix() string {
	keyPrefix := ""
	if u.config.KeyPrefix != "" {
		keyPrefix = fmt.Sprintf("%s/", u.config.KeyPrefix)
	}

	return keyPrefix
}

func (u *S3Uploader) listUploadedSnapshotsAscending(ctx context.Context, keyPrefix string) ([]s3Types.Object, error) {

	var result []s3Types.Object

	existingSnapshotList, err := u.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &u.config.Bucket,
		Prefix: aws.String(keyPrefix),
	})

	if err != nil {
		return result, fmt.Errorf("error listing uploaded snapshots: %w", err)
	}

	for _, obj := range existingSnapshotList.Contents {
		if strings.HasSuffix(*obj.Key, ".snap") && strings.Contains(*obj.Key, "raft_snapshot-") {
			result = append(result, obj)
		}
	}

	timestamp := func(o1, o2 *s3Types.Object) bool {
		return o1.LastModified.Before(*o2.LastModified)
	}

	S3By(timestamp).Sort(result)

	return result, nil
}

// implementation of Sort interface for s3 objects
type S3By func(f1, f2 *s3Types.Object) bool

func (by S3By) Sort(objects []s3Types.Object) {
	fs := &s3ObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type s3ObjectSorter struct {
	objects []s3Types.Object
	by      func(f1, f2 *s3Types.Object) bool // Closure used in the Less method.
}

func (s *s3ObjectSorter) Len() int {
	return len(s.objects)
}

func (s *s3ObjectSorter) Less(i, j int) bool {
	return s.by(&s.objects[i], &s.objects[j])
}

func (s *s3ObjectSorter) Swap(i, j int) {
	s.objects[i], s.objects[j] = s.objects[j], s.objects[i]
}
