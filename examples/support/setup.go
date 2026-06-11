package support

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

/*
RepoRoot walks upward from the working directory until it finds go.mod.
*/
func RepoRoot() (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for dir := workDir; ; {
		modPath := filepath.Join(dir, "go.mod")
		if _, statErr := os.Stat(modPath); statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("examples: go.mod not found from %q", workDir)
		}

		dir = parent
	}
}

/*
LoadViper reads cmd/cfg/config.yml from the repository root.
*/
func LoadViper() error {
	root, err := RepoRoot()
	if err != nil {
		return err
	}

	configPath := filepath.Join(root, "cmd", "cfg", "config.yml")
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	viper.SetDefault("project.name", "Animal")
	viper.SetDefault("project.description", "Multi-agent coordination harness.")

	return nil
}

/*
ConfigureSalonContext overrides project framing for the sentience salon example.
*/
func ConfigureSalonContext() {
	viper.Set("project.name", "Sentience panel")
	viper.Set("project.description", "An open philosophical discussion about conscious AI. This is not a product delivery meeting.")
}

/*
OpenAIConfig returns ai.endpoint, ai.apiKey, and ai.model from the active viper config.
*/
func OpenAIConfig() (endpoint, apiKey, model string) {
	endpoint = viper.GetString("ai.endpoint")
	apiKey = os.ExpandEnv(viper.GetString("ai.apiKey"))
	model = viper.GetString("ai.model")

	return endpoint, apiKey, model
}

/*
NewQPool returns a minimal qpool for examples and tests.
*/
func NewQPool(ctx context.Context) *qpool.Q[any] {
	return qpool.NewQ[any](ctx, 1, 2, &qpool.Config{Scaler: nil})
}

/*
DefaultSwarmOptions returns swarm settings matching cmd/cfg/config.yml defaults.
*/
func DefaultSwarmOptions(meshID string) swarm.Options {
	return swarm.Options{
		MeshID:    meshID,
		GossipTTL: 30 * time.Second,
		MeshTTL:   90 * time.Minute,
		Buffer:    64,
	}
}

/*
DefaultLeaseOptions returns path-prefix lease defaults for examples.
*/
func DefaultLeaseOptions() lease.Options {
	return lease.Options{
		KeySpace: lease.PathKeySpace{},
		IdleTTL:  15 * time.Minute,
	}
}

/*
DrainParticipant merges pending mesh rumors into the local view.
*/
func DrainParticipant(participant *swarm.Participant) error {
	return participant.Drain()
}

/*
WaitAnnounce blocks until the participant view records an announcement topic.
*/
func WaitAnnounce(participant *swarm.Participant, topic string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if drainErr := participant.Drain(); drainErr != nil {
			return drainErr
		}

		for _, record := range participant.View().RecentAnnounces() {
			if record.Topic == topic {
				return nil
			}
		}

		time.Sleep(5 * time.Millisecond)
	}

	return fmt.Errorf("examples: timed out waiting for announce topic %q", topic)
}
