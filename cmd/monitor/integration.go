package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/output"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// checkRequiredTools verifies that necessary external tools are available
func checkRequiredTools(logger system.Logger, cmdRunner system.CommandRunner) error {
	requiredTools := []struct {
		name        string
		command     string
		optional    bool
		description string
	}{
		{"smartctl", "command -v smartctl >/dev/null 2>&1", false, "SMART monitoring utility"},
		{"lspci", "command -v lspci >/dev/null 2>&1", true, "PCI device information utility"},
		{"storcli", "command -v storcli64 >/dev/null 2>&1 || command -v storcli >/dev/null 2>&1", true, "LSI storage controller utility"},
		{"zpool", "command -v zpool >/dev/null 2>&1", true, "ZFS pool management utility"},
	}

	missing := []string{}
	
	for _, tool := range requiredTools {
		exitCode := runCheck(cmdRunner, tool.command)
		if exitCode != 0 {
			if tool.optional {
				logger.Info("Optional tool '%s' not found - %s", tool.name, tool.description)
			} else {
				logger.Error("Required tool '%s' not found - %s", tool.name, tool.description)
				missing = append(missing, tool.name)
			}
		} else {
			logger.Debug("Found required tool: %s", tool.name)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}
	
	return nil
}

// runCheck executes a command and returns the exit code
func runCheck(cmdRunner system.CommandRunner, command string) int {
	// Use platform-specific command invocation 
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = fmt.Sprintf("cmd /c \"%s && exit 0 || exit 1\"", command)
	} else {
		cmd = fmt.Sprintf("sh -c '%s && exit 0 || exit 1'", command)
	}
	
	// Run with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := cmdRunner.Run(ctx, cmd)
	if err != nil {
		return 1 // Command failed
	}
	return 0 // Command succeeded
}

// createDummyOutput generates a dummy output for testing or when real data collection failed
func createDummyOutput(config *model.Config, reason string) error {
	// Create a simple text message
	message := fmt.Sprintf(`
=========================================
磁盘健康监控工具 - 数据收集失败
=========================================

日期时间: %s
失败原因: %s

建议:
1. 检查系统权限（可能需要root权限）
2. 确认必要工具已安装（smartctl）
3. 请查看日志文件以获取更多信息: %s
4. 使用 --debug 参数启用调试模式获取更多信息

=========================================
`, time.Now().Format("2006-01-02 15:04:05"), reason, config.LogFile)

	// Only create file if output was specified
	if config.OutputFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(config.OutputFile)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory for error output: %w", err)
			}
		}
		
		// Write error message to file
		if err := os.WriteFile(config.OutputFile, []byte(message), 0644); err != nil {
			return fmt.Errorf("failed to write error output: %w", err)
		}
		
		fmt.Printf("Error information saved to %s\n", config.OutputFile)
	} else {
		// Print to console
		fmt.Println(message)
	}
	
	return nil
}

// formatFormatterOptions creates a map of options for the output formatter
func formatFormatterOptions(app *Application) map[string]interface{} {
	options := make(map[string]interface{})
	
	// Basic formatting options
	options[output.OptionGroupByType] = !app.Config.NoGroup
	options[output.OptionIncludeSummary] = true
	options[output.OptionIncludeTimestamp] = true
	options[output.OptionColorOutput] = !app.Quiet

	// Format-specific options
	options[output.OptionCompactMode] = app.CompactMode
	
	// PDF-specific options (if using PDF format)
	if app.Config.OutputFormat == model.OutputFormatPDF {
		options[output.OptionPaperSize] = output.PaperSizeA4       // Default to A4
		options[output.OptionOrientation] = output.OrientationPortrait // Default to portrait
		options[output.OptionIncludeCover] = true                  // Include a cover page
	}
	
	// Text-specific options (if using text format)
	if app.Config.OutputFormat == model.OutputFormatText {
		options[output.OptionBorderStyle] = output.BorderStyleClassic // Use classic borders
		options[output.OptionMaxWidth] = 120                         // Set max width to 120 chars
	}
	
	return options
}
