package collectors

import "github.com/fireharp/tinkershop/internal/storage"

type Result struct {
	Projects     []storage.Project
	Observations []storage.Observation
}
