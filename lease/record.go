package lease

import "time"

/*
leaseRecord tracks which actor holds an exclusive lease and when it was last touched.
Idle expiration in Coordinator compares lastUsed against Options.IdleTTL to reclaim stale keys.
*/
type leaseRecord struct {
	actorID  string
	prefix   string
	lastUsed time.Time
}
