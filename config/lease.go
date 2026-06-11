package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/theapemachine/animal/lease"
)

/*
IdleTTL returns the configured lease idle expiration duration.
*/
func (section LeaseSection) IdleTTL() (time.Duration, error) {
	if section.IdleTTLSeconds <= 0 {
		return 0, fmt.Errorf("config: ai.lease.idle_ttl_seconds is required")
	}

	return time.Duration(section.IdleTTLSeconds) * time.Second, nil
}

/*
CoordinatorOptions builds lease.Options from the configured defaults.
*/
func (section LeaseSection) CoordinatorOptions(keySpace lease.KeySpace) (lease.Options, error) {
	idleTTL, err := section.IdleTTL()
	if err != nil {
		return lease.Options{}, err
	}

	if keySpace == nil {
		return lease.Options{}, fmt.Errorf("config: lease key space is required")
	}

	return lease.Options{
		KeySpace: keySpace,
		IdleTTL:  idleTTL,
	}, nil
}

/*
CoordinatorOptions builds lease.Options from the loaded config document.
*/
func (cfg *Config) CoordinatorOptions(keySpace lease.KeySpace) (lease.Options, error) {
	return cfg.AI.Lease.CoordinatorOptions(keySpace)
}

/*
LeaseCoordinatorOptionsFromViper reads ai.lease from the active viper config.
*/
func LeaseCoordinatorOptionsFromViper(keySpace lease.KeySpace) (lease.Options, error) {
	section := LeaseSection{
		IdleTTLSeconds: viper.GetInt("ai.lease.idle_ttl_seconds"),
	}

	return section.CoordinatorOptions(keySpace)
}
