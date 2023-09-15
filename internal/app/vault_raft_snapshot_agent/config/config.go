package config

import (
	"fmt"
	"log"
)

type Parser[T Configuration] struct {
	delegate rattlesnake
}

type Configuration interface {
	HasUploaders() bool
}

func NewParser[T Configuration](envPrefix string, configFilename string, configSearchPaths ...string) Parser[T] {
	return Parser[T]{newRattlesnake(envPrefix, configFilename, configSearchPaths...)}
}

// ReadConfig reads the configuration file
func (p Parser[T]) ReadConfig(config T, file string) error {
	err := p.delegate.BindAllEnv(
		map[string]string{
			"vault.url":                        "VAULT_ADDR",
			"uploaders.aws.credentials.key":    "AWS_ACCESS_KEY_ID",
			"uploaders.aws.credentials.secret": "AWS_SECRET_ACCESS_KEY",
		},
	)
	if err != nil {
		return fmt.Errorf("could not bind environment-variables: %s", err)
	}

	if file != "" {
		if err := p.delegate.SetConfigFile(file); err != nil {
			return err
		}
	}

	if err := p.delegate.ReadInConfig(); err != nil {
		if p.delegate.IsConfigurationNotFoundError(err) {
			log.Printf("Could not find any configuration file, will create configuration based solely on environment...")
		} else {
			return err
		}
	}

	if usedConfigFile := p.delegate.ConfigFileUsed(); usedConfigFile != "" {
		log.Printf("Using configuration from %s...\n", usedConfigFile)
	}

	if err := p.delegate.Unmarshal(config); err != nil {
		return fmt.Errorf("could not unmarshal configuration: %s", err)
	}

	if !config.HasUploaders() {
		return fmt.Errorf("no uploaders configured!")
	}

	return nil
}

func (p Parser[T]) OnConfigChange(config T, handler func(config T) error) <-chan error {
	ch := make(chan error, 1)

	p.delegate.OnConfigChange(func() {
		if err := p.delegate.Unmarshal(config); err != nil {
			log.Printf("Ignoring configuration change as configuration in %s is invalid: %v\n", p.delegate.ConfigFileUsed(), err)
			ch <- err
		} else {
			ch <- handler(config)
		}
	})

	return ch
}
