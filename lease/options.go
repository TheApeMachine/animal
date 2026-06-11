package lease

import "time"

/*
Options configures a Coordinator.
IdleTTL must be supplied from config (see config.LeaseSection).
*/
type Options struct {
	KeySpace KeySpace
	IdleTTL  time.Duration
}
