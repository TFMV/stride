package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	stride "github.com/TFMV/stride/internal/walk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var findCmd = &cobra.Command{
	Use:   "find [options] <path>",
	Short: "Find files with advanced filtering",
	Long: `Find files with advanced filtering capabilities.
Supports pattern matching, time-based filtering, size constraints, and more.
Can execute commands for each matched file or format output using templates.

Examples:
  stride find /path/to/search --name="*.go"
  stride find /path/to/search --regex=".*\\.txt$" --larger-than=1MB
  stride find /path/to/search --exec="echo Processing: {}"
  stride find /path/to/search --format="{base} ({size} bytes)"
  stride find /path/to/search --older-than=7d --watch`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		return runFind(path)
	},
}

func init() {
	rootCmd.AddCommand(findCmd)

	// Pattern matching options
	findCmd.Flags().StringP("name", "n", "", "Match by file name (supports wildcards)")
	findCmd.Flags().StringP("path", "p", "", "Match by path (supports wildcards)")
	findCmd.Flags().String("ignore", "", "Skip paths matching this pattern")
	findCmd.Flags().StringP("regex", "r", "", "Match by regular expression")

	// Time-based filtering
	findCmd.Flags().String("older-than", "", "Files older than this duration (e.g. 7d, 24h, 30m)")
	findCmd.Flags().String("newer-than", "", "Files newer than this duration (e.g. 7d, 24h, 30m)")

	// Size-based filtering
	findCmd.Flags().String("larger-than", "", "Files larger than this size (e.g. 1MB, 500KB)")
	findCmd.Flags().String("smaller-than", "", "Files smaller than this size (e.g. 1MB, 500KB)")

	// Metadata and tag filtering
	findCmd.Flags().StringSlice("meta", []string{}, "Metadata key-value patterns to match (key=regex)")
	findCmd.Flags().StringSlice("tag", []string{}, "Tag key-value patterns to match (key=regex)")

	// Execution options
	findCmd.Flags().String("exec", "", "Command to execute for each match")
	findCmd.Flags().String("format", "", "Format string for output")

	// Traversal options
	findCmd.Flags().UintP("max-depth", "d", 0, "Maximum directory depth to traverse")
	findCmd.Flags().Bool("follow-symlinks", false, "Follow symbolic links")
	findCmd.Flags().Bool("include-hidden", false, "Include hidden files")
	findCmd.Flags().Bool("with-versions", false, "Include file versions")

	// Watch options
	findCmd.Flags().BoolP("watch", "w", false, "Watch for changes")
	findCmd.Flags().StringSlice("watch-events", []string{"create", "modify"}, "Events to watch for")

	// Bind flags to viper
	viper.BindPFlag("find.name", findCmd.Flags().Lookup("name"))
	viper.BindPFlag("find.path", findCmd.Flags().Lookup("path"))
	viper.BindPFlag("find.ignore", findCmd.Flags().Lookup("ignore"))
	viper.BindPFlag("find.regex", findCmd.Flags().Lookup("regex"))
	viper.BindPFlag("find.older-than", findCmd.Flags().Lookup("older-than"))
	viper.BindPFlag("find.newer-than", findCmd.Flags().Lookup("newer-than"))
	viper.BindPFlag("find.larger-than", findCmd.Flags().Lookup("larger-than"))
	viper.BindPFlag("find.smaller-than", findCmd.Flags().Lookup("smaller-than"))
	viper.BindPFlag("find.meta", findCmd.Flags().Lookup("meta"))
	viper.BindPFlag("find.tag", findCmd.Flags().Lookup("tag"))
	viper.BindPFlag("find.exec", findCmd.Flags().Lookup("exec"))
	viper.BindPFlag("find.format", findCmd.Flags().Lookup("format"))
	viper.BindPFlag("find.max-depth", findCmd.Flags().Lookup("max-depth"))
	viper.BindPFlag("find.follow-symlinks", findCmd.Flags().Lookup("follow-symlinks"))
	viper.BindPFlag("find.include-hidden", findCmd.Flags().Lookup("include-hidden"))
	viper.BindPFlag("find.with-versions", findCmd.Flags().Lookup("with-versions"))
	viper.BindPFlag("find.watch", findCmd.Flags().Lookup("watch"))
	viper.BindPFlag("find.watch-events", findCmd.Flags().Lookup("watch-events"))
}

func runFind(root string) error {
	// Create find options
	opts := stride.FindOptions{
		NamePattern:    viper.GetString("find.name"),
		PathPattern:    viper.GetString("find.path"),
		IgnorePattern:  viper.GetString("find.ignore"),
		MaxDepth:       viper.GetUint("find.max-depth"),
		FollowSymlinks: viper.GetBool("find.follow-symlinks"),
		IncludeHidden:  viper.GetBool("find.include-hidden"),
		WithVersions:   viper.GetBool("find.with-versions"),
		Watch:          viper.GetBool("find.watch"),
		WatchEvents:    viper.GetStringSlice("find.watch-events"),
	}

	// Parse regex pattern
	if regexStr := viper.GetString("find.regex"); regexStr != "" {
		var err error
		opts.RegexPattern, err = regexp.Compile(regexStr)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Parse time durations
	if olderThanStr := viper.GetString("find.older-than"); olderThanStr != "" {
		duration, err := parseDuration(olderThanStr)
		if err != nil {
			return fmt.Errorf("invalid older-than value: %w", err)
		}
		opts.OlderThan = duration
	}

	if newerThanStr := viper.GetString("find.newer-than"); newerThanStr != "" {
		duration, err := parseDuration(newerThanStr)
		if err != nil {
			return fmt.Errorf("invalid newer-than value: %w", err)
		}
		opts.NewerThan = duration
	}

	// Parse size constraints
	if largerThanStr := viper.GetString("find.larger-than"); largerThanStr != "" {
		size, err := parseSize(largerThanStr)
		if err != nil {
			return fmt.Errorf("invalid larger-than value: %w", err)
		}
		opts.LargerSize = size
	}

	if smallerThanStr := viper.GetString("find.smaller-than"); smallerThanStr != "" {
		size, err := parseSize(smallerThanStr)
		if err != nil {
			return fmt.Errorf("invalid smaller-than value: %w", err)
		}
		opts.SmallerSize = size
	}

	// Parse metadata and tag patterns
	if metaPatterns := viper.GetStringSlice("find.meta"); len(metaPatterns) > 0 {
		metaMap, err := parseKeyValuePatterns(metaPatterns)
		if err != nil {
			return fmt.Errorf("invalid metadata pattern: %w", err)
		}
		opts.MatchMeta, err = stride.CompileRegexMap(metaMap)
		if err != nil {
			return fmt.Errorf("invalid metadata regex: %w", err)
		}
	}

	if tagPatterns := viper.GetStringSlice("find.tag"); len(tagPatterns) > 0 {
		tagMap, err := parseKeyValuePatterns(tagPatterns)
		if err != nil {
			return fmt.Errorf("invalid tag pattern: %w", err)
		}
		opts.MatchTags, err = stride.CompileRegexMap(tagMap)
		if err != nil {
			return fmt.Errorf("invalid tag regex: %w", err)
		}
	}

	// Execute the find operation
	ctx := context.Background()

	// If exec command is specified, use it
	if execCmd := viper.GetString("find.exec"); execCmd != "" {
		return stride.FindWithExec(ctx, root, opts, execCmd)
	}

	// If format is specified, use it
	if format := viper.GetString("find.format"); format != "" {
		return stride.FindWithFormat(ctx, root, opts, format)
	}

	// Otherwise, use default handler
	return stride.Find(ctx, root, opts, nil)
}

// parseDuration parses a duration string with support for days (d)
func parseDuration(s string) (time.Duration, error) {
	// Handle days specially
	if strings.HasSuffix(s, "d") {
		days, err := parseFloat(s[:len(s)-1])
		if err != nil {
			return 0, err
		}
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}

	// Use standard duration parsing for other units
	return time.ParseDuration(s)
}

// parseSize parses a size string with support for KB, MB, GB, TB
func parseSize(s string) (int64, error) {
	s = strings.ToUpper(s)

	multiplier := int64(1)

	if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-2]
	}

	size, err := parseFloat(s)
	if err != nil {
		return 0, err
	}

	return int64(size * float64(multiplier)), nil
}

// parseFloat parses a float from a string
func parseFloat(s string) (float64, error) {
	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}

// parseKeyValuePatterns parses key=value patterns from a string slice
func parseKeyValuePatterns(patterns []string) (map[string]string, error) {
	result := make(map[string]string, len(patterns))

	for _, pattern := range patterns {
		parts := strings.SplitN(pattern, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid pattern format: %s (expected key=value)", pattern)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		result[key] = value
	}

	return result, nil
}
