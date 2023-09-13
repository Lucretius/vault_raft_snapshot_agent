package vault_raft_snapshot_agent

import (
	"fmt"
	"log"
)

var parser rattlesnake = newRattlesnake("snapshot", "VRSA", "/etc/vault.d/", ".")

// ReadConfig reads the configuration file
func ReadConfig(file string) (config SnapshotterConfig, err error) {
	config = SnapshotterConfig{}

	err = parser.BindAllEnv(
		map[string]string{
			"vault.url":                        "VAULT_ADDR",
			"uploaders.aws.credentials.key":    "AWS_ACCESS_KEY_ID",
			"uploaders.aws.credentials.secret": "SECRET_ACCESS_KEY",
		},
	)
	if err != nil {
		return config, fmt.Errorf("could not bind environment-variables: %s", err)
	}

	if file != "" {
		if err := parser.SetConfigFile(file); err != nil {
			return config, err
		}
	}

	if err := parser.ReadInConfig(); err != nil {
		if parser.IsConfigurationNotFoundError(err) {
			if file != "" {
				return config, err
			} else {
				log.Printf("Could not find any configuration file, will create configuration based solely on environment...")
			}
		} else {
			return config, err
		}
	}

	if usedConfigFile := parser.ConfigFileUsed(); usedConfigFile != "" {
		log.Printf("Using configuration from %s...\n", usedConfigFile)
	}

	if err := parser.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("could not unmarshal configuration: %s", err)
	}

	if !config.Uploaders.HasUploaders() {
		return config, fmt.Errorf("no uploaders configured!")
	}

	return config, nil
}

func WatchConfigAndReconfigure(snapshotter *Snapshotter) <-chan error {
	ch := make(chan error, 1)

	parser.OnConfigChange(func() {
		config := SnapshotterConfig{}

		if err := parser.Unmarshal(&config); err != nil {
			log.Printf("Ignoring configuration change as configuration in %s is invalid: %v\n", parser.ConfigFileUsed(), err)
			ch <- err
		} else if err := snapshotter.Reconfigure(config); err != nil {
			log.Printf("Could not reconfigure snapshotter from %s: %v\n", parser.ConfigFileUsed(), err)
			ch <- err
		} else {
			ch <- nil
		}
	})

	return ch
}
