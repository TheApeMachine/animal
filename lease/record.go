package lease

import "time"

type leaseRecord struct {
	actorID  string
	lastUsed time.Time
}
