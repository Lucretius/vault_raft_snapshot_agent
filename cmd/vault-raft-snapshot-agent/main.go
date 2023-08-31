/*
Vault Raft Snapshot Agent periodically takes snapshots of Vault's raft database.
It uploads those snaphots to one or more storage locations like a local harddrive
or an AWS S3 Bucket.

Usage:

    vault-raft-snapshot-agent [flags] [options]

The flags are:

    -v, -version
		Prints version information and exits

The options are:

	-c -config <file>
		Specifies the config-file to use. 

If no config file is explicitly specified, the program looks for configuration-files
with the name `snapshot` and the extensions supported by [viper]
in the current working directory or in /etc/vault.d/snapshots.

For details on how to configure the program see the [README]

[viper]: https://github.com/spf13/viper
[README]: https://github.com/Argelbargel/vault-raft-snapshot-agent/README.md
*/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	internal "github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent"
)

var Version = "development"
var Platform = "linux/amd64"

type quietBoolFlag struct {
	cli.BoolFlag
}

func (qbf *quietBoolFlag) String() string {
	return cli.FlagStringer(qbf)
}

func (qbf *quietBoolFlag) GetDefaultText() string {
	return ""
}

func main() {
	cli.VersionPrinter = func(ctx *cli.Context) {
		fmt.Printf("%s (%s), version: %s\n", ctx.App.Name, Platform, ctx.App.Version)
	}

	cli.VersionFlag = &quietBoolFlag{
		cli.BoolFlag{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "print the version",
		},
	}

	app := &cli.App{
		Name:        "vault-raft-snapshot-agent",
		Version:     Version,
		Description: "takes periodic snapshot of vault's raft-db",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "/etc/vault.d/snapshot.json",
				Usage:   "Load configuration from `FILE`",
				EnvVars: []string{"VAULT_RAFT_SNAPSHOT_AGENT_CONFIG_FILE"},
			},
		},
		Action: func(ctx *cli.Context) error {
			startSnapshotter(ctx.Path("config"))
			return nil
		},
	}
	app.CustomAppHelpTemplate = `Usage: {{.HelpName}} [options]
{{.Description}}

Options:
{{range $index, $option := .VisibleFlags}}{{if $index}}
{{end}}{{$option}}{{end}}`

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startSnapshotter(configFile cli.Path) {
	config, err := internal.ReadConfig(configFile)
	if err != nil {
		log.Fatalf("Could not read configuration file %s: %s\n", configFile, err)
	}

	snapshotter, err := internal.CreateSnapshotter(config)
	if err != nil {
		log.Fatalf("Cannot instantiate snapshotter: %s\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	runSnapshotter(ctx, snapshotter)
}

func runSnapshotter(ctx context.Context, snapshotter *internal.Snapshotter) {
	internal.WatchConfigAndReconfigure(snapshotter)

	for {
		frequency, err := snapshotter.TakeSnapshot(ctx)
		if err != nil {
			log.Printf("Could not take snapshot or upload to all targets: %v\n", err)
		}
		select {
		case <-time.After(frequency):
			continue
		case <-ctx.Done():
			os.Exit(1)
		}
	}
}
