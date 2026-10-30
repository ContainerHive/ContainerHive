package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/build"
	"github.com/ContainerHive/ContainerHive/pkg/changes"
	"github.com/ContainerHive/ContainerHive/pkg/discovery"
	"github.com/urfave/cli/v3"
)

const (
	changesFormatText = "text"
	changesFormatJSON = "json"
)

func changesCmd() *cli.Command {
	return &cli.Command{
		Name:  "changes",
		Usage: "Print the images affected by changes since a base revision",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "base",
				Usage: "Git revision to compare HEAD against, via their merge base",
			},
			&cli.StringFlag{
				Name:  "files-from",
				Usage: "Read changed files, relative to the project root, one per line from this file (- for stdin) instead of git",
			},
			&cli.StringFlag{
				Name:  "unmatched",
				Usage: "Files outside every image directory: " + changes.UnmatchedAll + " rebuilds every image, " + changes.UnmatchedIgnore + " drops them",
				Value: changes.UnmatchedAll,
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format: " + changesFormatText + " or " + changesFormatJSON,
				Value: changesFormatText,
			},
			&cli.BoolFlag{
				Name:  "write",
				Usage: "Store the result as dist/" + changes.FileName + ", which limits build, test, finalize and sbom to affected images",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			format := cmd.String("format")
			if err := validateChangesFormat(format); err != nil {
				return err
			}
			if err := changes.ParseUnmatched(cmd.String("unmatched")); err != nil {
				return err
			}
			base, filesFrom := cmd.String("base"), cmd.String("files-from")
			if (base == "") == (filesFrom == "") {
				return errors.New("set exactly one of --base or --files-from")
			}

			// Change detection only needs image directories and the
			// dependency graph, so it must not fail on an unreachable
			// version source.
			project, err := discoverProject(ctx, cmd, discovery.WithSkipTagRanges())
			if err != nil {
				return err
			}

			opts := changes.Options{Unmatched: cmd.String("unmatched")}
			var set *changes.Set
			if filesFrom != "" {
				files, err := readChangedFiles(cmd.Root().Reader, filesFrom)
				if err != nil {
					return err
				}
				set, err = changes.Detect(project, files, opts)
				if err != nil {
					return err
				}
			} else {
				set, err = changes.DetectFromGit(project, base, opts)
				if err != nil {
					return err
				}
			}

			if set.Global {
				slog.Info("All images affected", "reason", set.Reason)
			}
			if cmd.Bool("write") {
				if err := changes.Write(getDistPath(cmd), set); err != nil {
					return fmt.Errorf("failed to write %s: %w", changes.FileName, err)
				}
			}
			return writeChanges(cmd.Root().Writer, set, format)
		},
	}
}

func validateChangesFormat(format string) error {
	switch format {
	case changesFormatText, changesFormatJSON:
		return nil
	}
	return fmt.Errorf("unknown changes format %q, want %s or %s", format, changesFormatText, changesFormatJSON)
}

// writeChanges prints the affected images, one per line for text.
func writeChanges(w io.Writer, set *changes.Set, format string) error {
	if err := validateChangesFormat(format); err != nil {
		return err
	}
	if format == changesFormatJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(set)
	}
	for _, name := range set.Affected {
		if _, err := fmt.Fprintln(w, name); err != nil {
			return err
		}
	}
	return nil
}

// readChangedFiles reads one file path per line from path, or from stdin for
// "-". Blank lines are skipped.
func readChangedFiles(stdin io.Reader, path string) ([]string, error) {
	reader := stdin
	if path != "-" {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open changed files: %w", err)
		}
		defer file.Close()
		reader = file
	}

	var files []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			files = append(files, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read changed files: %w", err)
	}
	return files, nil
}

// limitToChanges narrows filters to the images affected according to
// dist/changes.json. Without the file, filters are returned unchanged and the
// set is nil. skip reports that no filtered image is affected.
func limitToChanges(distPath string, filters []build.Filter) (limited []build.Filter, set *changes.Set, skip bool, err error) {
	set, err = changes.Load(distPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to load %s: %w", changes.FileName, err)
	}
	if set == nil {
		return filters, nil, false, nil
	}

	if len(filters) == 0 {
		filters = []build.Filter{{}}
	}
	for _, filter := range filters {
		if filter.ImageName != "" {
			if set.IsAffected(filter.ImageName) {
				limited = append(limited, filter)
			}
			continue
		}
		for _, name := range set.Affected {
			limited = append(limited, build.Filter{ImageName: name, TagName: filter.TagName})
		}
	}

	if len(limited) == 0 {
		slog.Info("No affected images to process, skipping", "changes", changes.FileName)
		return nil, set, true, nil
	}
	slog.Info("Limited to affected images", "affected", set.Affected)
	return limited, set, false, nil
}
