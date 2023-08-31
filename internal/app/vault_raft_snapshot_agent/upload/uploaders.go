package upload

import (
	"context"
	"io"
)

type UploadersConfig struct {
	AWS   AWSConfig   `default:"{\"Empty\": true}" mapstructure:"aws"`
	Azure AzureConfig `default:"{\"Empty\": true}" mapstructure:"azure"`
	GCP   GCPConfig   `default:"{\"Empty\": true}" mapstructure:"google"`
	Local LocalConfig `default:"{\"Empty\": true}" mapstructure:"local"`
}

func (c UploadersConfig) HasUploaders() bool {
	return !(c.AWS.Empty && c.Azure.Empty && c.GCP.Empty && c.Local.Empty)
}

type Uploader interface {
	Upload(ctx context.Context, reader io.Reader, currentTs int64, retain int) error
}

func CreateUploaders(config UploadersConfig) ([]Uploader, error) {
	var uploaders []Uploader

	if !config.AWS.Empty {
		aws, err := newAWSUploader(config.AWS)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, aws)
	}

	if !config.Azure.Empty {
		azure, err := newAzureUploader(config.Azure)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, azure)
	}

	if !config.GCP.Empty {
		gcp, err := newGCPUploader(config.GCP)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, gcp)
	}

	if !config.Local.Empty {
		local, err := newLocalUploader(config.Local)
		if err != nil {
			return nil, err
		}
		uploaders = append(uploaders, local)
	}

	return uploaders, nil
}
