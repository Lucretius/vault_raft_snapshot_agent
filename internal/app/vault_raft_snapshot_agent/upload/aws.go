package upload

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type AWSConfig struct {
	Credentials             AWSCredentialsConfig `default:"{\"Empty\": true}"`
	Region                  string               `validate:"required_if=Empty false"`
	Bucket                  string               `validate:"required_if=Empty false"`
	KeyPrefix               string               `mapstructure:",omitifempty"`
	Endpoint                string               `mapstructure:",omitifempty"`
	UseServerSideEncryption bool
	ForcePathStyle          bool
	Empty                   bool
}

type AWSCredentialsConfig struct {
	Key    string `validate:"required_if=Empty false"`
	Secret string `validate:"required_if=Empty false"`
	Empty  bool
}

type awsUploader struct {
	client    *s3.Client
	uploader  *manager.Uploader
	keyPrefix string
	bucket    string
	sse       bool
}

func newAWSUploader(config AWSConfig) (*awsUploader, error) {
	clientConfig, err := awsConfig.LoadDefaultConfig(context.Background(), awsConfig.WithRegion(config.Region))

	if err != nil {
		return nil, fmt.Errorf("failed to load default aws config: %w", err)
	}

	if !config.Credentials.Empty {
		creds := credentials.NewStaticCredentialsProvider(config.Credentials.Key, config.Credentials.Secret, "")
		clientConfig.Credentials = creds
	}

	client := s3.NewFromConfig(clientConfig, func(o *s3.Options) {
		o.UsePathStyle = config.ForcePathStyle
		if config.Endpoint != "" {
			o.BaseEndpoint = aws.String(config.Endpoint)
		}
	})

	keyPrefix := ""
	if config.KeyPrefix != "" {
		keyPrefix = fmt.Sprintf("%s/", config.KeyPrefix)
	}

	return &awsUploader{
		client,
		manager.NewUploader(client),
		keyPrefix,
		config.Bucket,
		config.UseServerSideEncryption,
	}, nil
}

func (u *awsUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64, retain int) error {
	input := &s3.PutObjectInput{
		Bucket: &u.bucket,
		Key:    aws.String(fmt.Sprintf("%sraft_snapshot-%d.snap", u.keyPrefix, currentTs)),
		Body:   reader,
	}

	if u.sse {
		input.ServerSideEncryption = s3Types.ServerSideEncryptionAes256
	}

	_, err := u.uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("error uploading snapshot to aws s3: %w", err)
	} else {
		if retain > 0 {

			existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx)

			if err != nil {
				return fmt.Errorf("error getting existing snapshots from aws s3: %w", err)
			}

			if len(existingSnapshots)-int(retain) <= 0 {
				return nil
			}
			snapshotsToDelete := existingSnapshots[0 : len(existingSnapshots)-int(retain)]

			for i := range snapshotsToDelete {
				_, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &u.bucket,
					Key:    snapshotsToDelete[i].Key,
				})
				if err != nil {
					return fmt.Errorf("error deleting snapshot %s from aws s3: %w", *snapshotsToDelete[i].Key, err)
				}
			}
		}
		return nil
	}
}

func (u *awsUploader) listUploadedSnapshotsAscending(ctx context.Context) ([]s3Types.Object, error) {
	var result []s3Types.Object

	existingSnapshotList, err := u.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &u.bucket,
		Prefix: aws.String(u.keyPrefix),
	})

	if err != nil {
		return result, fmt.Errorf("error listing uploaded snapshots on aws s3: %w", err)
	}

	for _, obj := range existingSnapshotList.Contents {
		if strings.HasSuffix(*obj.Key, ".snap") && strings.Contains(*obj.Key, "raft_snapshot-") {
			result = append(result, obj)
		}
	}

	timestamp := func(o1, o2 *s3Types.Object) bool {
		return o1.LastModified.Before(*o2.LastModified)
	}

	s3By(timestamp).Sort(result)

	return result, nil
}

// implementation of Sort interface for s3 objects
type s3By func(f1, f2 *s3Types.Object) bool

func (by s3By) Sort(objects []s3Types.Object) {
	fs := &s3ObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type s3ObjectSorter struct {
	objects []s3Types.Object
	by      s3By
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
