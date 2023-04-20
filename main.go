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
	frequency, err := time.ParseDuration(c.Frequency)

	if err != nil {
		frequency = time.Hour
	}

	snapshotTimeout := 60 * time.Second

	if c.SnapshotTimeout != "" {
		snapshotTimeout, err = time.ParseDuration(c.SnapshotTimeout)

		if err != nil {
			log.Fatalln("Unable to parse snapshot timeout", err)
		}
	}

	for {
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

			func() {
				log.Println("Starting backup.")
				snapshot, err := os.CreateTemp("", "snapshot")

				if err != nil {
					log.Fatalln("Unable to create temporary snapshot file", err.Error())
				}

				defer os.Remove(snapshot.Name())

				ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
				defer cancel()

				err = snapshotter.API.Sys().RaftSnapshotWithContext(ctx, snapshot)
				if err != nil {
					log.Fatalln("Unable to generate snapshot", err.Error())
				}

				_, err = snapshot.Seek(0, io.SeekStart)
				if err != nil {
					log.Fatalln("Unable to seek to start of snapshot file", err.Error())
				}

				now := time.Now().UnixNano()
				if c.Local.Path != "" {
					snapshotPath, err := snapshotter.CreateLocalSnapshot(snapshot, c, now)
					logSnapshotError("local", snapshotPath, err)
				}
				if c.AWS.Bucket != "" {
					snapshotPath, err := snapshotter.CreateS3Snapshot(snapshot, c, now)
					logSnapshotError("aws", snapshotPath, err)
				}
				if c.GCP.Bucket != "" {
					snapshotPath, err := snapshotter.CreateGCPSnapshot(snapshot, c, now)
					logSnapshotError("gcp", snapshotPath, err)
				}
				if c.Azure.ContainerName != "" {
					snapshotPath, err := snapshotter.CreateAzureSnapshot(snapshot, c, now)
					logSnapshotError("azure", snapshotPath, err)
				}
				log.Println("Backup completed.")
			}()
		}
		select {
		case <-time.After(frequency):
			continue
		case <-done:
			os.Exit(1)
		}
	}
}

func logSnapshotError(dest, snapshotPath string, err error) {
	if err != nil {
		log.Printf("Failed to generate %s snapshot to %s: %v\n", dest, snapshotPath, err)
	} else {
		log.Printf("Successfully created %s snapshot to %s\n", dest, snapshotPath)
	}
}
