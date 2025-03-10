package cmd

import (
	"fmt"
	"os"

	walk "github.com/TFMV/stride/internal/walk"
	"github.com/spf13/cobra"
)

var (
	// Analyze command options
	analyzeOutputFormat   string
	analyzeOutputFile     string
	analyzeDuplicates     bool
	analyzeCodeStats      bool
	analyzeStorageReport  bool
	analyzeSecurityScan   bool
	analyzeContentPattern bool
	analyzeLanguages      []string
	analyzeMaxDepth       int
	analyzeMinSize        string
	analyzeMaxSize        string
	analyzeIncludeHidden  bool
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze filesystem structure and content",
	Long: `Analyze filesystem structure, content patterns, and usage statistics.

Examples:
  stride analyze /path/to/directory
  stride analyze --duplicates /path/to/directory
  stride analyze --storage-report --output=html --output-file=report.html /path/to/directory
  stride analyze --code-stats --languages=go,js,py /path/to/repos
  stride analyze --security-scan /path/to/directory
  stride analyze --content-pattern --max-depth=3 /path/to/directory`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the directory to analyze
		var analyzeDir string
		if len(args) > 0 {
			analyzeDir = args[0]
		} else {
			var err error
			analyzeDir, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Create an analyzer based on the flags
		analyzer := walk.NewAnalyzer()

		// Configure the analyzer
		analyzer.SetOutputFormat(analyzeOutputFormat)
		analyzer.SetOutputFile(analyzeOutputFile)

		if analyzeDuplicates {
			analyzer.EnableDuplicateDetection()
		}

		if analyzeCodeStats {
			analyzer.EnableCodeStats()
			analyzer.SetLanguages(analyzeLanguages)
		}

		if analyzeStorageReport {
			analyzer.EnableStorageReport()
		}

		if analyzeSecurityScan {
			analyzer.EnableSecurityScan()
		}

		if analyzeContentPattern {
			analyzer.EnableContentPatternAnalysis()
		}

		// Set common options
		analyzer.SetMaxDepth(analyzeMaxDepth)
		analyzer.SetSizeRange(analyzeMinSize, analyzeMaxSize)
		analyzer.SetIncludeHidden(analyzeIncludeHidden)

		// Run the analysis
		result, err := analyzer.Analyze(analyzeDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing directory: %v\n", err)
			os.Exit(1)
		}

		// Output the results
		if analyzeOutputFile != "" {
			err = result.SaveToFile(analyzeOutputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving results to file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Analysis results saved to %s\n", analyzeOutputFile)
		} else {
			// Print to stdout
			fmt.Println(result.String())
		}
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	// Define flags for the analyze command
	analyzeCmd.Flags().StringVar(&analyzeOutputFormat, "output", "text", "Output format (text, json, csv, html)")
	analyzeCmd.Flags().StringVar(&analyzeOutputFile, "output-file", "", "File to write output to")
	analyzeCmd.Flags().BoolVar(&analyzeDuplicates, "duplicates", false, "Find duplicate files")
	analyzeCmd.Flags().BoolVar(&analyzeCodeStats, "code-stats", false, "Analyze code statistics")
	analyzeCmd.Flags().BoolVar(&analyzeStorageReport, "storage-report", false, "Generate storage usage report")
	analyzeCmd.Flags().BoolVar(&analyzeSecurityScan, "security-scan", false, "Perform security scan of permissions and ownership")
	analyzeCmd.Flags().BoolVar(&analyzeContentPattern, "content-pattern", false, "Analyze content patterns")
	analyzeCmd.Flags().StringSliceVar(&analyzeLanguages, "languages", []string{}, "Languages to analyze for code stats (comma-separated)")
	analyzeCmd.Flags().IntVar(&analyzeMaxDepth, "max-depth", 0, "Maximum directory depth to analyze (0 for unlimited)")
	analyzeCmd.Flags().StringVar(&analyzeMinSize, "min-size", "", "Minimum file size to analyze")
	analyzeCmd.Flags().StringVar(&analyzeMaxSize, "max-size", "", "Maximum file size to analyze")
	analyzeCmd.Flags().BoolVar(&analyzeIncludeHidden, "include-hidden", false, "Include hidden files and directories")
}
