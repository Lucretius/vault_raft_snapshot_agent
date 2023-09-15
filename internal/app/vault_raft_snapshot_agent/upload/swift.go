package upload

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ncw/swift/v2"
)

type SwiftUploaderConfig struct {
	Container string `validate:"required_if=Empty false"`
	UserName  string `validate:"required_if=Empty false"`
	ApiKey    string `validate:"required_if=Empty false"`
	AuthUrl   string `validate:"required_if=Empty false,omitempty,http_url"`
	Domain    string `validate:"omitempty,http_url"`
	Region    string
	TenantId  string
	Timeout   time.Duration `default:"60s"`
	Empty     bool
}

type swiftUploaderImpl struct {
	connection *swift.Connection
	container  string
}

func createSwiftUploader(ctx context.Context, config SwiftUploaderConfig) (*uploader[swift.Object], error) {
	conn := swift.Connection{
		UserName: config.UserName,
		ApiKey:   config.ApiKey,
		AuthUrl:  config.AuthUrl,
		Region:   config.Region,
		TenantId: config.TenantId,
		Domain:   config.Domain,
		Timeout:  config.Timeout,
	}

	if err := conn.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("invalid credentials: %s", err)
	}

	if _, _, err := conn.Container(ctx, config.Container); err != nil {
		return nil, fmt.Errorf("invalid container %s: %s", config.Container, err)
	}

	return &uploader[swift.Object]{
		swiftUploaderImpl{
			connection: &conn,
			container:  config.Container,
		},
	}, nil
}

// nolint:unused
// implements interface uploaderImpl
func (u swiftUploaderImpl) Destination() string {
	return fmt.Sprintf("swift container %s", u.container)
}

// nolint:unused
// implements interface uploaderImpl
func (u swiftUploaderImpl) uploadSnapshot(ctx context.Context, name string, data io.Reader) error {
	_, header, err := u.connection.Container(ctx, u.container)
	if err != nil {
		return err
	}

	object, err := u.connection.ObjectCreate(ctx, u.container, name, false, "", "", header)
	if err != nil {
		return err
	}

	if _, err := io.Copy(object, data); err != nil {
		return err
	}

	if err := object.Close(); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u swiftUploaderImpl) deleteSnapshot(ctx context.Context, snapshot swift.Object) error {
	if err := u.connection.ObjectDelete(ctx, u.container, snapshot.Name); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u swiftUploaderImpl) listSnapshots(ctx context.Context, prefix string, ext string) ([]swift.Object, error) {
	return u.connection.ObjectsAll(ctx, u.container, &swift.ObjectsOpts{Prefix: prefix})
}

// nolint:unused
// implements interface uploaderImpl
func (u swiftUploaderImpl) compareSnapshots(a, b swift.Object) int {
	return a.LastModified.Compare(a.LastModified)
}
