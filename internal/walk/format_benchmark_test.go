package stride

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// oldFormatCommand is the original implementation for comparison
func oldFormatCommand(template string, msg FindMessage) string {
	str := template

	// Replace basic placeholders
	str = strings.ReplaceAll(str, "{}", msg.Path)
	str = strings.ReplaceAll(str, "{base}", msg.Name)
	str = strings.ReplaceAll(str, "{dir}", msg.Dir)
	str = strings.ReplaceAll(str, "{size}", fmt.Sprintf("%d", msg.Size))
	str = strings.ReplaceAll(str, "{time}", msg.Time.Format(time.RFC3339))

	// Replace quoted versions
	str = strings.ReplaceAll(str, `{""}`, strconv.Quote(msg.Path))
	str = strings.ReplaceAll(str, `{"base"}`, strconv.Quote(msg.Name))
	str = strings.ReplaceAll(str, `{"dir"}`, strconv.Quote(msg.Dir))
	str = strings.ReplaceAll(str, `{"size"}`, strconv.Quote(fmt.Sprintf("%d", msg.Size)))
	str = strings.ReplaceAll(str, `{"time"}`, strconv.Quote(msg.Time.Format(time.RFC3339)))

	// Replace version if available
	if msg.VersionID != "" {
		str = strings.ReplaceAll(str, "{version}", msg.VersionID)
		str = strings.ReplaceAll(str, `{"version"}`, strconv.Quote(msg.VersionID))
	}

	return str
}

func BenchmarkFormatCommandOld(b *testing.B) {
	templates := []string{
		"Path: {}, Name: {base}, Dir: {dir}, Size: {size}, Time: {time}",
		`Path: {""}, Name: {"base"}, Dir: {"dir"}, Size: {"size"}, Time: {"time"}`,
		"Path: {}, Version: {version}, Quoted Version: {\"version\"}",
		"File: {base} ({size} bytes) in {dir}, modified at {time}, version: {version}",
		"This is a plain string with no placeholders",
	}

	msg := FindMessage{
		Path:      "/path/to/file.txt",
		Name:      "file.txt",
		Dir:       "/path/to",
		Size:      1024,
		Time:      time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		VersionID: "v1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, template := range templates {
			_ = oldFormatCommand(template, msg)
		}
	}
}

func BenchmarkFormatCommandNew(b *testing.B) {
	templates := []string{
		"Path: {}, Name: {base}, Dir: {dir}, Size: {size}, Time: {time}",
		`Path: {""}, Name: {"base"}, Dir: {"dir"}, Size: {"size"}, Time: {"time"}`,
		"Path: {}, Version: {version}, Quoted Version: {\"version\"}",
		"File: {base} ({size} bytes) in {dir}, modified at {time}, version: {version}",
		"This is a plain string with no placeholders",
	}

	msg := FindMessage{
		Path:      "/path/to/file.txt",
		Name:      "file.txt",
		Dir:       "/path/to",
		Size:      1024,
		Time:      time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		VersionID: "v1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, template := range templates {
			_ = formatCommand(template, msg)
		}
	}
}
