package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/theapemachine/animal/swarm"
)

/*
SwarmSection configures gossip mesh and situational awareness defaults.
*/
type SwarmSection struct {
	Enabled          bool   `yaml:"enabled"`
	MeshID           string `yaml:"mesh_id"`
	GossipTTLSeconds int    `yaml:"gossip_ttl_seconds"`
	MeshTTLSeconds   int    `yaml:"mesh_ttl_seconds"`
	MeshBuffer       int    `yaml:"mesh_buffer"`
}

/*
Options builds swarm.Options from configured defaults.
*/
func (section SwarmSection) Options() (swarm.Options, error) {
	if section.MeshID == "" {
		return swarm.Options{}, fmt.Errorf("config: ai.swarm.mesh_id is required")
	}

	if section.GossipTTLSeconds <= 0 {
		return swarm.Options{}, fmt.Errorf("config: ai.swarm.gossip_ttl_seconds is required")
	}

	if section.MeshTTLSeconds <= 0 {
		return swarm.Options{}, fmt.Errorf("config: ai.swarm.mesh_ttl_seconds is required")
	}

	if section.MeshBuffer <= 0 {
		return swarm.Options{}, fmt.Errorf("config: ai.swarm.mesh_buffer is required")
	}

	return swarm.Options{
		MeshID:    section.MeshID,
		GossipTTL: time.Duration(section.GossipTTLSeconds) * time.Second,
		MeshTTL:   time.Duration(section.MeshTTLSeconds) * time.Second,
		Buffer:    section.MeshBuffer,
	}, nil
}

/*
SwarmOptionsFromViper reads ai.swarm from the active viper config.
*/
func SwarmOptionsFromViper() (swarm.Options, error) {
	section := SwarmSection{
		Enabled:          viper.GetBool("ai.swarm.enabled"),
		MeshID:           viper.GetString("ai.swarm.mesh_id"),
		GossipTTLSeconds: viper.GetInt("ai.swarm.gossip_ttl_seconds"),
		MeshTTLSeconds:   viper.GetInt("ai.swarm.mesh_ttl_seconds"),
		MeshBuffer:       viper.GetInt("ai.swarm.mesh_buffer"),
	}

	return section.Options()
}
