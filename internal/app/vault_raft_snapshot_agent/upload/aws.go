package upload

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type AWSUploaderConfig struct {
	Credentials             AWSUploaderCredentialsConfig `default:"{\"Empty\": true}"`
	Bucket                  string                       `validate:"required_if=Empty false"`
	KeyPrefix               string                       `mapstructure:",omitifempty"`
	Endpoint                string                       `mapstructure:",omitifempty"`
	Region                  string
	UseServerSideEncryption bool
	ForcePathStyle          bool
	Empty                   bool
}

type AWSUploaderCredentialsConfig struct {
	Key    string `validate:"required_if=Empty false"`
	Secret string `validate:"required_if=Empty false"`
	Empty  bool
}

type awsUploaderImpl struct {
	client    *s3.Client
	uploader  *manager.Uploader
	keyPrefix string
	bucket    string
	sse       bool
}

func createAWSUploader(ctx context.Context, config AWSUploaderConfig) (*uploader[s3Types.Object], error) {
	clientConfig, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(config.Region))

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

	return &uploader[s3Types.Object]{
		awsUploaderImpl{
			client:    client,
			uploader:  manager.NewUploader(client),
			keyPrefix: keyPrefix,
			bucket:    config.Bucket,
			sse:       config.UseServerSideEncryption,
		},
	}, nil
}

func (u awsUploaderImpl) Destination() string {
	return fmt.Sprintf("aws s3 bucket %s ", u.bucket)
}

// nolint:unused
// implements interface uploaderImpl
func (u awsUploaderImpl) uploadSnapshot(ctx context.Context, name string, data io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket: &u.bucket,
		Key:    aws.String(u.keyPrefix + name),
		Body:   data,
	}

	if u.sse {
		input.ServerSideEncryption = s3Types.ServerSideEncryptionAes256
	}

	if _, err := u.uploader.Upload(ctx, input); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u awsUploaderImpl) deleteSnapshot(ctx context.Context, snapshot s3Types.Object) error {
	input := &s3.DeleteObjectInput{
		Bucket: &u.bucket,
		Key:    snapshot.Key,
	}

	if _, err := u.client.DeleteObject(ctx, input); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u awsUploaderImpl) listSnapshots(ctx context.Context, prefix string, ext string) ([]s3Types.Object, error) {
	var result []s3Types.Object

	existingSnapshotList, err := u.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &u.bucket,
		Prefix: aws.String(u.keyPrefix),
	})

	if err != nil {
		return result, err
	}

	for _, obj := range existingSnapshotList.Contents {
		if strings.HasSuffix(*obj.Key, ext) && strings.Contains(*obj.Key, prefix) {
			result = append(result, obj)
		}
	}

	return result, nil
}

// nolint:unused
// implements interface uploaderImpl
func (u awsUploaderImpl) compareSnapshots(a, b s3Types.Object) int {
	return a.LastModified.Compare(*b.LastModified)
}
