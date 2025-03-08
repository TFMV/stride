// Package cmd provides the CLI commands for the stride command.
//
// This package contains the implementation of the `stride` command, which is a
// high-performance file walking utility that extends the standard `filepath.Walk`
// functionality with concurrency, filtering, and monitoring capabilities.
//
// The `stride` command supports various options for filtering files based on name,
// path, size, modification time, and more. It also provides functionality to execute
// commands for each matched file or format the output using templates.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	stride "github.com/TFMV/stride/internal/walk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "0.1.0"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "stride [options] <path>",
	Short: "A high-performance file walking utility",
	Long: `stride is a command line utility for high-performance filesystem traversal.
It supports concurrent processing, filtering, and real-time progress monitoring.

Example:
  stride /path/to/directory                    # Basic usage
  stride --pattern="*.go" --workers=8 /src     # Find Go files using 8 workers
  stride --follow-symlinks --progress /data    # Follow symlinks with progress`,
	Version: version,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing required argument: path\n\nUsage: stride <path>\nExample: stride /path/to/directory")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments: expected 1, got %d\n\nUsage: stride <path>", len(args))
		}
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set default excluded directories if none specified
		if cmd.Flags().Lookup("exclude-dir").Value.String() == "" {
			// Common system directories that often have permission issues
			defaultExcludes := []string{
				".Trash",
				".Trashes",
				".fseventsd",
				".Spotlight-V100",
				"System Volume Information",
				"$RECYCLE.BIN",
				"lost+found",
			}
			cmd.Flags().Set("exclude-dir", strings.Join(defaultExcludes, ","))
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		return runFileWalker(path)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags
	rootCmd.Flags().StringP("workers", "w", "4", "Number of concurrent workers")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().Bool("silent", false, "Disable all output except errors")
	rootCmd.Flags().String("format", "text", "Output format (text|json)")
	rootCmd.Flags().String("min-size", "", "Minimum file size to process")
	rootCmd.Flags().String("max-size", "", "Maximum file size to process")
	rootCmd.Flags().String("pattern", "", "File pattern to match")
	rootCmd.Flags().String("exclude-dir", "", "Directories to exclude (comma-separated)")
	rootCmd.Flags().String("exclude-pattern", "", "Patterns to exclude files (comma-separated)")
	rootCmd.Flags().String("file-types", "", "File types to include (comma-separated: file,dir,symlink,pipe,socket,device,char)")
	rootCmd.Flags().Bool("follow-symlinks", false, "Follow symbolic links")
	rootCmd.Flags().Bool("progress", false, "Show progress updates")
	rootCmd.Flags().String("error-mode", "continue", "Error handling mode (continue|stop|skip)")
	rootCmd.Flags().String("min-permissions", "", "Minimum file permissions (octal, e.g. 0644)")
	rootCmd.Flags().String("max-permissions", "", "Maximum file permissions (octal, e.g. 0755)")
	rootCmd.Flags().String("exact-permissions", "", "Exact file permissions to match (octal, e.g. 0644)")
	rootCmd.Flags().String("owner", "", "Filter by owner username")
	rootCmd.Flags().String("group", "", "Filter by group name")
	rootCmd.Flags().Int("owner-uid", 0, "Filter by owner UID")
	rootCmd.Flags().Int("owner-gid", 0, "Filter by group GID")
	rootCmd.Flags().Int("min-depth", 0, "Minimum directory depth to process")
	rootCmd.Flags().Int("max-depth", 0, "Maximum directory depth to process")
	rootCmd.Flags().Bool("empty-files", false, "Include only empty files")
	rootCmd.Flags().Bool("empty-dirs", false, "Include only empty directories")
	rootCmd.Flags().String("modified-after", "", "Include files modified after (format: YYYY-MM-DD)")
	rootCmd.Flags().String("modified-before", "", "Include files modified before (format: YYYY-MM-DD)")
	rootCmd.Flags().String("accessed-after", "", "Include files accessed after (format: YYYY-MM-DD)")
	rootCmd.Flags().String("accessed-before", "", "Include files accessed before (format: YYYY-MM-DD)")
	rootCmd.Flags().String("created-after", "", "Include files created after (format: YYYY-MM-DD)")
	rootCmd.Flags().String("created-before", "", "Include files created before (format: YYYY-MM-DD)")

	// Bind flags to viper
	viper.BindPFlag("workers", rootCmd.Flags().Lookup("workers"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("silent", rootCmd.Flags().Lookup("silent"))
	viper.BindPFlag("format", rootCmd.Flags().Lookup("format"))
	viper.BindPFlag("min-size", rootCmd.Flags().Lookup("min-size"))
	viper.BindPFlag("max-size", rootCmd.Flags().Lookup("max-size"))
	viper.BindPFlag("pattern", rootCmd.Flags().Lookup("pattern"))
	viper.BindPFlag("exclude-dir", rootCmd.Flags().Lookup("exclude-dir"))
	viper.BindPFlag("exclude-pattern", rootCmd.Flags().Lookup("exclude-pattern"))
	viper.BindPFlag("file-types", rootCmd.Flags().Lookup("file-types"))
	viper.BindPFlag("follow-symlinks", rootCmd.Flags().Lookup("follow-symlinks"))
	viper.BindPFlag("progress", rootCmd.Flags().Lookup("progress"))
	viper.BindPFlag("error-mode", rootCmd.Flags().Lookup("error-mode"))
	viper.BindPFlag("min-permissions", rootCmd.Flags().Lookup("min-permissions"))
	viper.BindPFlag("max-permissions", rootCmd.Flags().Lookup("max-permissions"))
	viper.BindPFlag("exact-permissions", rootCmd.Flags().Lookup("exact-permissions"))
	viper.BindPFlag("owner", rootCmd.Flags().Lookup("owner"))
	viper.BindPFlag("group", rootCmd.Flags().Lookup("group"))
	viper.BindPFlag("owner-uid", rootCmd.Flags().Lookup("owner-uid"))
	viper.BindPFlag("owner-gid", rootCmd.Flags().Lookup("owner-gid"))
	viper.BindPFlag("min-depth", rootCmd.Flags().Lookup("min-depth"))
	viper.BindPFlag("max-depth", rootCmd.Flags().Lookup("max-depth"))
	viper.BindPFlag("empty-files", rootCmd.Flags().Lookup("empty-files"))
	viper.BindPFlag("empty-dirs", rootCmd.Flags().Lookup("empty-dirs"))
	viper.BindPFlag("modified-after", rootCmd.Flags().Lookup("modified-after"))
	viper.BindPFlag("modified-before", rootCmd.Flags().Lookup("modified-before"))
	viper.BindPFlag("accessed-after", rootCmd.Flags().Lookup("accessed-after"))
	viper.BindPFlag("accessed-before", rootCmd.Flags().Lookup("accessed-before"))
	viper.BindPFlag("created-after", rootCmd.Flags().Lookup("created-after"))
	viper.BindPFlag("created-before", rootCmd.Flags().Lookup("created-before"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".stride" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".stride")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func runFileWalker(root string) error {
	// Parse workers
	workersStr := viper.GetString("workers")
	workers, err := strconv.Atoi(workersStr)
	if err != nil {
		return fmt.Errorf("invalid workers value: %s", workersStr)
	}

	// Create filter options
	filter := stride.FilterOptions{
		ExcludeDir: []string{},
	}

	// Parse min-size
	if minSizeStr := viper.GetString("min-size"); minSizeStr != "" {
		minSize, err := strconv.ParseInt(minSizeStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid min-size value: %s", minSizeStr)
		}
		filter.MinSize = minSize
	}

	// Parse max-size
	if maxSizeStr := viper.GetString("max-size"); maxSizeStr != "" {
		maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max-size value: %s", maxSizeStr)
		}
		filter.MaxSize = maxSize
	}

	// Set pattern
	if pattern := viper.GetString("pattern"); pattern != "" {
		filter.Pattern = pattern
	}

	// Set exclude directories
	if excludeDirs := viper.GetString("exclude-dir"); excludeDirs != "" {
		filter.ExcludeDir = strings.Split(excludeDirs, ",")
	}

	// Set exclude patterns
	if excludePatterns := viper.GetString("exclude-pattern"); excludePatterns != "" {
		filter.ExcludePattern = strings.Split(excludePatterns, ",")
	}

	// Set file types
	if fileTypes := viper.GetString("file-types"); fileTypes != "" {
		filter.FileTypes = strings.Split(fileTypes, ",")
	}

	// Parse permission filters
	if minPermStr := viper.GetString("min-permissions"); minPermStr != "" {
		// Parse octal string to int64
		minPerm, err := strconv.ParseInt(minPermStr, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid min-permissions value: %s (should be octal, e.g. 0644)", minPermStr)
		}
		filter.MinPermissions = os.FileMode(minPerm)
	}

	if maxPermStr := viper.GetString("max-permissions"); maxPermStr != "" {
		// Parse octal string to int64
		maxPerm, err := strconv.ParseInt(maxPermStr, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid max-permissions value: %s (should be octal, e.g. 0755)", maxPermStr)
		}
		filter.MaxPermissions = os.FileMode(maxPerm)
	}

	if exactPermStr := viper.GetString("exact-permissions"); exactPermStr != "" {
		// Parse octal string to int64
		exactPerm, err := strconv.ParseInt(exactPermStr, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid exact-permissions value: %s (should be octal, e.g. 0644)", exactPermStr)
		}
		filter.ExactPermissions = os.FileMode(exactPerm)
		filter.UseExactPermissions = true
	}

	// Parse owner filter
	if owner := viper.GetString("owner"); owner != "" {
		filter.OwnerName = owner
	}

	// Parse group filter
	if group := viper.GetString("group"); group != "" {
		filter.GroupName = group
	}

	// Parse owner UID filter
	if ownerUID := viper.GetInt("owner-uid"); ownerUID != 0 {
		filter.OwnerUID = ownerUID
	}

	// Parse owner GID filter
	if ownerGID := viper.GetInt("owner-gid"); ownerGID != 0 {
		filter.OwnerGID = ownerGID
	}

	// Parse directory depth filters
	if minDepth := viper.GetInt("min-depth"); minDepth != 0 {
		filter.MinDepth = minDepth
	}

	if maxDepth := viper.GetInt("max-depth"); maxDepth != 0 {
		filter.MaxDepth = maxDepth
	}

	// Parse empty files filter
	if viper.GetBool("empty-files") {
		filter.IncludeEmptyFiles = true
	}

	// Parse empty directories filter
	if viper.GetBool("empty-dirs") {
		filter.IncludeEmptyDirs = true
	}

	// Parse modified time filters
	if modifiedAfter := viper.GetString("modified-after"); modifiedAfter != "" {
		modifiedAfterTime, err := time.Parse("2006-01-02", modifiedAfter)
		if err != nil {
			return fmt.Errorf("invalid modified-after format: %s", modifiedAfter)
		}
		filter.ModifiedAfter = modifiedAfterTime
	}

	if modifiedBefore := viper.GetString("modified-before"); modifiedBefore != "" {
		modifiedBeforeTime, err := time.Parse("2006-01-02", modifiedBefore)
		if err != nil {
			return fmt.Errorf("invalid modified-before format: %s", modifiedBefore)
		}
		filter.ModifiedBefore = modifiedBeforeTime
	}

	// Parse accessed time filters
	if accessedAfter := viper.GetString("accessed-after"); accessedAfter != "" {
		accessedAfterTime, err := time.Parse("2006-01-02", accessedAfter)
		if err != nil {
			return fmt.Errorf("invalid accessed-after format: %s", accessedAfter)
		}
		filter.AccessedAfter = accessedAfterTime
	}

	if accessedBefore := viper.GetString("accessed-before"); accessedBefore != "" {
		accessedBeforeTime, err := time.Parse("2006-01-02", accessedBefore)
		if err != nil {
			return fmt.Errorf("invalid accessed-before format: %s", accessedBefore)
		}
		filter.AccessedBefore = accessedBeforeTime
	}

	// Parse created time filters
	if createdAfter := viper.GetString("created-after"); createdAfter != "" {
		createdAfterTime, err := time.Parse("2006-01-02", createdAfter)
		if err != nil {
			return fmt.Errorf("invalid created-after format: %s", createdAfter)
		}
		filter.CreatedAfter = createdAfterTime
	}

	if createdBefore := viper.GetString("created-before"); createdBefore != "" {
		createdBeforeTime, err := time.Parse("2006-01-02", createdBefore)
		if err != nil {
			return fmt.Errorf("invalid created-before format: %s", createdBefore)
		}
		filter.CreatedBefore = createdBeforeTime
	}

	// Create walk options
	opts := stride.WalkOptions{
		Filter: filter,
	}

	// Set error handling mode
	errorMode := viper.GetString("error-mode")
	switch errorMode {
	case "continue":
		opts.ErrorHandling = stride.ErrorHandlingContinue
	case "stop":
		opts.ErrorHandling = stride.ErrorHandlingStop
	case "skip":
		opts.ErrorHandling = stride.ErrorHandlingSkip
	default:
		return fmt.Errorf("invalid error-mode: %s", errorMode)
	}

	// Set symlink handling
	if viper.GetBool("follow-symlinks") {
		opts.SymlinkHandling = stride.SymlinkFollow
	} else {
		opts.SymlinkHandling = stride.SymlinkIgnore
	}

	// Set log level and logger
	if viper.GetBool("verbose") {
		opts.LogLevel = stride.LogLevelDebug
	} else if viper.GetBool("silent") {
		opts.LogLevel = stride.LogLevelError
	} else {
		opts.LogLevel = stride.LogLevelInfo
	}

	// Ensure logger is initialized
	if opts.Logger == nil {
		// We can't directly call createLogger as it's not exported
		// Let the stride package handle logger creation
		// The logger will be created in WalkLimitWithOptions if it's nil
	}

	// Set progress function if requested
	if viper.GetBool("progress") {
		// Print a final newline when done
		defer fmt.Println()

		opts.Progress = func(stats stride.Stats) {
			if viper.GetString("format") == "json" {
				jsonStats, _ := json.Marshal(stats)
				fmt.Println(string(jsonStats))
			} else {
				fmt.Printf("\rProcessed: %d files, %d dirs, %.2f MB (%.2f MB/s)    ",
					stats.FilesProcessed,
					stats.DirsProcessed,
					float64(stats.BytesProcessed)/(1024*1024),
					stats.SpeedMBPerSec)
			}
		}
	}

	// Create a context
	ctx := context.Background()

	// Set buffer size based on workers
	opts.BufferSize = workers

	// Process files
	return stride.WalkLimitWithOptions(ctx, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if info is nil to avoid nil pointer dereference
		if info == nil {
			return nil
		}

		// Skip directories as they are handled by the walker
		if info.IsDir() {
			return nil
		}

		// Output file information based on format
		if viper.GetString("format") == "json" {
			fileInfo := map[string]interface{}{
				"path":          path,
				"size":          info.Size(),
				"mode":          info.Mode().String(),
				"last_modified": info.ModTime().Format(time.RFC3339),
			}
			jsonInfo, _ := json.Marshal(fileInfo)
			fmt.Println(string(jsonInfo))
		} else if !viper.GetBool("silent") && !viper.GetBool("progress") {
			relPath, _ := filepath.Rel(root, path)
			fmt.Printf("%s (%d bytes)\n", relPath, info.Size())
		}

		return nil
	}, opts)
}
