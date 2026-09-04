package cli

import (
	"github.com/ContainerHive/ContainerHive/pkg/shard"
	"github.com/urfave/cli/v3"
)

// shardFlags returns the --max-shards/--current-shard flags shared by
// build, sbom and test. They fall back to CI_NODE_TOTAL/CI_NODE_INDEX -
// the variables GitLab sets for `parallel: N` - so a project gets
// sharding for free under `parallel:` without passing any flags. The
// names are vendor-neutral: other CI providers without native fan-out
// can export the same two variables from a wrapper script.
func shardFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "max-shards",
			Usage:   "Total number of shards to split work across",
			Sources: cli.EnvVars("CI_NODE_TOTAL"),
			Value:   1,
		},
		&cli.IntFlag{
			Name:    "current-shard",
			Usage:   "This shard's 1-based index (must be between 1 and --max-shards)",
			Sources: cli.EnvVars("CI_NODE_INDEX"),
			Value:   1,
		},
	}
}

// resolveShard reads and validates the shard flags for the given command.
func resolveShard(cmd *cli.Command) (shard.Shard, error) {
	s := shard.Shard{
		Current: cmd.Int("current-shard"),
		Max:     cmd.Int("max-shards"),
	}
	if err := s.Validate(); err != nil {
		return shard.Shard{}, err
	}
	return s, nil
}
