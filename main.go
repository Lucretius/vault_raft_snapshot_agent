package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	"github.com/Lucretius/vault_raft_snapshot_agent/snapshot_agent"
)

func listenForInterruptSignals() chan bool {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		_ = <-sigs
		done <- true
	}()
	return done
}

func main() {
	done := listenForInterruptSignals()

	log.Println("Reading configuration...")
	c, err := config.ReadConfig()

	if err != nil {
		log.Fatalln("Configuration could not be found")
	}

	snapshotter, err := snapshot_agent.NewSnapshotter(c)
	if err != nil {
		log.Fatalln("Cannot instantiate snapshotter.", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configuredFrequency, err := time.ParseDuration(c.Frequency)

	if err != nil {
		configuredFrequency = time.Hour
	}

	snapshotTimeout := 60 * time.Second

	if c.SnapshotTimeout != "" {
		snapshotTimeout, err = time.ParseDuration(c.SnapshotTimeout)

		if err != nil {
			log.Fatalln("Unable to parse snapshot timeout", err)
		}
	}

	var lastSuccessfulUploads snapshot_agent.LastUpload

	for {
		frequency := configuredFrequency

		if snapshotter.TokenExpiration.Before(time.Now()) {
			switch c.VaultAuthMethod {
			case "k8s":
				err := snapshotter.SetClientTokenFromK8sAuth(c)
				if err != nil {
					log.Fatalln("Unable to get token from k8s auth")
					return
				}
			case "token":
				// Do nothing as vault agent will auto-renew the token
			default:
				err := snapshotter.SetClientTokenFromAppRole(c)
				if err != nil {
					log.Fatalln("Unable to get token from approle")
					return
				}
			}
		}
		leader, err := snapshotter.API.Sys().Leader()
		if err != nil {
			log.Println(err.Error())
			log.Fatalln("Unable to determine leader instance.  The snapshot agent will only run on the leader node.  Are you running this daemon on a Vault instance?")
		}
		leaderIsSelf := leader.IsSelf
		if !leaderIsSelf {
			log.Println("Not running on leader node, skipping.")
		} else {
			if lastSuccessfulUploads == nil {
				lastSuccessfulUploads, err = snapshotter.GetLastSuccessfulUploads(ctx)

				if err != nil {
					log.Fatalln("Unable to get last successful uploads", err)
				}

				frequency = lastSuccessfulUploads.NextBackupIn(configuredFrequency)
			} else {
				runBackup(ctx, snapshotter, snapshotTimeout)
			}
		}
		select {
		case <-time.After(frequency):
			continue
		case <-done:
			os.Exit(1)
		}
	}
}

func runBackup(ctx context.Context, snapshotter *snapshot_agent.Snapshotter, snapshotTimeout time.Duration) {
	log.Println("Starting backup.")
	snapshot, err := os.CreateTemp("", "snapshot")

	if err != nil {
		log.Printf("Unable to create temporary snapshot file: %s\n", err)
	}

	defer os.Remove(snapshot.Name())

	ctx, cancel := context.WithTimeout(ctx, snapshotTimeout)
	defer cancel()

	err = snapshotter.API.Sys().RaftSnapshotWithContext(ctx, snapshot)
	if err != nil {
		log.Printf("Unable to generate snapshot: %s\n", err)
	}

	_, err = snapshot.Seek(0, io.SeekStart)
	if err != nil {
		log.Printf("Unable to seek to start of snapshot file: %s\n", err)
	}

	now := time.Now().UnixNano()

	snapshotter.Lock()
	defer snapshotter.Unlock()

	for uploaderType, uploader := range snapshotter.Uploaders {
		snapshotPath, err := uploader.Upload(ctx, snapshot, now)

		if err != nil {
			log.Printf("Unable to upload %s snapshot (%s): %s\n", uploaderType, snapshotPath, err)
		}

		log.Printf("Successfully uploaded %s snapshot (%s)\n", uploaderType, snapshotPath)
	}

	log.Println("Backup completed.")
}
