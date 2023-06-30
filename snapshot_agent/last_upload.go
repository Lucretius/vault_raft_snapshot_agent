package snapshot_agent

import (
	"time"
)

type LastUpload map[UploaderType]time.Time

func (l *LastUpload) NextBackupIn(frequency time.Duration) time.Duration {

	var nextBackupIn time.Duration
	initial := true

	for _, lastUpload := range *l {
		since := time.Since(lastUpload)

		if since > frequency && initial {
			nextBackupIn = 0
		} else if since < frequency {
			nextBackupIn = frequency - since
		}
		initial = false
	}

	if nextBackupIn < 0 {
		nextBackupIn = 0
	}

	return nextBackupIn
}
