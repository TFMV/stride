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
	Use:   "filewalker [options] <path>",
	Short: "A file walking utility using stride",
	Long: `filewalker is a command line utility that walks through directories
and processes files based on specified criteria using the stride library.`,
	Version: version,
	Args:    cobra.ExactArgs(1),
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
	rootCmd.Flags().Bool("follow-symlinks", false, "Follow symbolic links")
	rootCmd.Flags().Bool("progress", false, "Show progress updates")
	rootCmd.Flags().String("error-mode", "continue", "Error handling mode (continue|stop|skip)")
	rootCmd.Flags().String("min-permissions", "", "Minimum file permissions (octal, e.g. 0644)")
	rootCmd.Flags().String("max-permissions", "", "Maximum file permissions (octal, e.g. 0755)")
	rootCmd.Flags().String("exact-permissions", "", "Exact file permissions to match (octal, e.g. 0644)")

	// Bind flags to viper
	viper.BindPFlag("workers", rootCmd.Flags().Lookup("workers"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("silent", rootCmd.Flags().Lookup("silent"))
	viper.BindPFlag("format", rootCmd.Flags().Lookup("format"))
	viper.BindPFlag("min-size", rootCmd.Flags().Lookup("min-size"))
	viper.BindPFlag("max-size", rootCmd.Flags().Lookup("max-size"))
	viper.BindPFlag("pattern", rootCmd.Flags().Lookup("pattern"))
	viper.BindPFlag("exclude-dir", rootCmd.Flags().Lookup("exclude-dir"))
	viper.BindPFlag("follow-symlinks", rootCmd.Flags().Lookup("follow-symlinks"))
	viper.BindPFlag("progress", rootCmd.Flags().Lookup("progress"))
	viper.BindPFlag("error-mode", rootCmd.Flags().Lookup("error-mode"))
	viper.BindPFlag("min-permissions", rootCmd.Flags().Lookup("min-permissions"))
	viper.BindPFlag("max-permissions", rootCmd.Flags().Lookup("max-permissions"))
	viper.BindPFlag("exact-permissions", rootCmd.Flags().Lookup("exact-permissions"))
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

		// Search config in home directory with name ".filewalker" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".filewalker")
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
		opts.Progress = func(stats stride.Stats) {
			if viper.GetString("format") == "json" {
				jsonStats, _ := json.Marshal(stats)
				fmt.Println(string(jsonStats))
			} else {
				fmt.Printf("\rProcessed: %d files, %d dirs, %d bytes, %.2f MB/s",
					stats.FilesProcessed, stats.DirsProcessed, stats.BytesProcessed, stats.SpeedMBPerSec)
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
