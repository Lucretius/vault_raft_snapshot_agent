package config

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	tagFieldName = "resolve-path"
)

var (
	errorInvalidType error = errors.New("subject must be a struct passed by pointer")
)

type pathResolver struct {
	baseDir string
}

func newPathResolver(baseDir string) pathResolver {
	return pathResolver{baseDir}
}

func (r pathResolver) Resolve(subject interface{}) error {
	if reflect.TypeOf(subject).Kind() != reflect.Ptr {
		return errorInvalidType
	}

	s := reflect.ValueOf(subject).Elem()

	return r.resolve(s)
}

func (r pathResolver) resolve(value reflect.Value) error {
	t := value.Type()

	if t.Kind() != reflect.Struct {
		return errorInvalidType
	}

	for i := 0; i < t.NumField(); i++ {
		f := value.Field(i)

		if !f.CanSet() {
			continue
		}

		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}

		if f.Kind() == reflect.Struct {
			if err := r.resolve(f); err != nil {
				return err
			}
		}

		if f.Kind() != reflect.String || f.String() == "" {
			continue
		}

		if baseDir, present := t.Field(i).Tag.Lookup(tagFieldName); present {
			if err := r.resolvePath(f, baseDir); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r pathResolver) resolvePath(field reflect.Value, baseDir string) error {
	path := field.String()
	if baseDir == "" {
		baseDir = r.baseDir
	}

	if !filepath.IsAbs(path) && !strings.HasPrefix(path, "/") {
		path = filepath.Join(baseDir, path)
		field.Set(reflect.ValueOf(path).Convert(field.Type()))
	}

	return nil
}
