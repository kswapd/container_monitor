package container

import (
	"time"

	"github.com/google/cadvisor/client"
)

var containerClient client.Client

func RecentStats(start, end time.Time, maxStats int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	return stats, nil
}
