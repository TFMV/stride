package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	stride "github.com/TFMV/stride/walk"
	"github.com/spf13/cobra"
)

var (
	// Watch command options
	watchEvents        []string
	watchRecursive     bool
	watchExec          string
	watchFormat        string
	watchPattern       string
	watchIgnore        string
	watchTimeout       time.Duration
	watchIncludeHidden bool
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch [path]",
	Short: "Watch for filesystem changes",
	Long: `Watch for filesystem changes and perform actions when files are created, modified, or deleted.

Examples:
  stride watch /path/to/watch
  stride watch --events=create,modify --exec="echo Changed: {}" /path/to/watch
  stride watch --pattern="*.go" --format="{base} was {event} at {time}" /path/to/watch
  stride watch --recursive /path/to/watch`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the directory to watch
		var watchDir string
		if len(args) > 0 {
			watchDir = args[0]
		} else {
			var err error
			watchDir, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Create a context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Convert string events to WatchEvent types
		var events []stride.WatchEvent
		for _, e := range watchEvents {
			switch strings.ToLower(e) {
			case "create":
				events = append(events, stride.EventCreate)
			case "write", "modify":
				events = append(events, stride.EventModify)
			case "remove", "delete":
				events = append(events, stride.EventDelete)
			case "rename":
				events = append(events, stride.EventRename)
			case "chmod":
				events = append(events, stride.EventChmod)
			default:
				fmt.Fprintf(os.Stderr, "Unknown event type: %s\n", e)
			}
		}

		// Create watch options
		opts := stride.WatchOptions{
			Context:       ctx,
			Events:        events,
			Recursive:     watchRecursive,
			Pattern:       watchPattern,
			IgnorePattern: watchIgnore,
			IncludeHidden: watchIncludeHidden,
			Timeout:       watchTimeout,
		}

		// Start watching
		fmt.Printf("Watching %s for changes...\n", watchDir)
		fmt.Println("Press Ctrl+C to exit.")

		var err error
		if watchExec != "" {
			// Execute command for each event
			err = stride.WatchWithExec(ctx, watchDir, opts, watchExec)
		} else if watchFormat != "" {
			// Format output for each event
			err = stride.WatchWithFormat(ctx, watchDir, opts, watchFormat)
		} else {
			// Use default handler
			err = stride.Watch(ctx, watchDir, opts, nil)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error watching directory: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Define flags for the watch command
	watchCmd.Flags().StringSliceVar(&watchEvents, "events", []string{}, "Events to watch for (create, modify, delete, rename, chmod)")
	watchCmd.Flags().BoolVar(&watchRecursive, "recursive", false, "Watch subdirectories recursively")
	watchCmd.Flags().StringVar(&watchExec, "exec", "", "Command to execute when an event occurs")
	watchCmd.Flags().StringVar(&watchFormat, "format", "", "Format string for output")
	watchCmd.Flags().StringVar(&watchPattern, "pattern", "", "File pattern to match (e.g., *.go)")
	watchCmd.Flags().StringVar(&watchIgnore, "ignore", "", "File pattern to ignore")
	watchCmd.Flags().DurationVar(&watchTimeout, "timeout", 0, "Duration to watch before exiting (e.g., 1h, 30m)")
	watchCmd.Flags().BoolVar(&watchIncludeHidden, "include-hidden", false, "Include hidden files and directories")
}
