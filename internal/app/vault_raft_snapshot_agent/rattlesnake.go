package vault_raft_snapshot_agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/creasty/defaults"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

// a rattlesnake is a viper adapted to our needs ;-)
type rattlesnake struct {
	v *viper.Viper
}

func newRattlesnake(configName string, envPrefix string, configPaths ...string) rattlesnake {
	v := viper.New()
	v.SetConfigName(configName)
	v.SetEnvPrefix(envPrefix)
	for _, path := range configPaths {
		v.AddConfigPath(path)
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return rattlesnake{v}
}

func (r rattlesnake) BindEnv(input ...string) error {
	return r.v.BindEnv(input...)
}

func (r rattlesnake) BindAllEnv(env map[string]string) error {
	for k, v := range env {
		if err := r.BindEnv(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (r rattlesnake) SetConfigFile(file string) error {
	if file != "" {
		file, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("could not build absolute path to config-file %s: %s", file, err)
		}
	}

	r.v.SetConfigFile(file)
	return nil
}

func (r rattlesnake) ReadInConfig() error {
	return r.v.ReadInConfig()
}

func (r rattlesnake) ConfigFileUsed() string {
	return r.v.ConfigFileUsed()
}

func (r rattlesnake) Unmarshal(config interface{}, opts ...viper.DecoderConfigOption) error {
	if err := bindStruct(r.v, config); err != nil {
		return fmt.Errorf("could not bind env vars for configuration: %s", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current working directory: %s", err)
	}

	configDir := filepath.Dir(r.ConfigFileUsed())
	if err := os.Chdir(configDir); err != nil {
		return fmt.Errorf("could not switch working-directory to %s to parse configuration: %s", configDir, err)
	}

	defer func() {
		if err := os.Chdir(wd); err != nil {
			log.Fatalf("Could not switch back to working directory %s: %s\n", wd, err)
		}
	}()

	if err := r.v.Unmarshal(config, opts...); err != nil {
		return err
	}

	if err := defaults.Set(config); err != nil {
		return fmt.Errorf("could not set configuration's default-values: %s", err)
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return err
	}

	return nil
}

func (r rattlesnake) OnConfigChange(run func()) {
	r.v.OnConfigChange(func(in fsnotify.Event) {
		run()
	})
	r.v.WatchConfig()
}

func (r rattlesnake) IsConfigurationNotFoundError(err error) bool {
	_, notfound := err.(viper.ConfigFileNotFoundError)
	return notfound
}

// implements automatic unmarshalling from environment variables
// see https://github.com/spf13/viper/pull/1429
// can be removed if that pr is merged
func bindStruct(v *viper.Viper, input interface{}) error {
	envKeysMap := map[string]interface{}{}
	if err := mapstructure.Decode(input, &envKeysMap); err != nil {
		return err
	}

	structKeys := flattenAndMergeMap(map[string]bool{}, envKeysMap, "")
	for key := range structKeys {
		if err := v.BindEnv(key); err != nil {
			return err
		}
	}

	return nil
}

func flattenAndMergeMap(shadow map[string]bool, m map[string]interface{}, prefix string) map[string]bool {
	if shadow != nil && prefix != "" && shadow[prefix] {
		// prefix is shadowed => nothing more to flatten
		return shadow
	}
	if shadow == nil {
		shadow = make(map[string]bool)
	}

	var m2 map[string]interface{}
	if prefix != "" {
		prefix += "."
	}
	for k, val := range m {
		fullKey := prefix + k
		switch val := val.(type) {
		case map[string]interface{}:
			m2 = val
		case map[interface{}]interface{}:
			m2 = cast.ToStringMap(val)
		default:
			// immediate value
			shadow[strings.ToLower(fullKey)] = true
			continue
		}
		// recursively merge to shadow map
		shadow = flattenAndMergeMap(shadow, m2, fullKey)
	}
	return shadow
}
