package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/output"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// 测试 checkRequiredTools 函数
func TestIntegrationCheckRequiredTools(t *testing.T) {
	// 检查 smartctl 是否存在
	_, err := exec.LookPath("smartctl")
	if err != nil {
		t.Skip("smartctl is not available, skipping this test")
	}

	mockLogger := system.NewMockLogger()
	mockCmdRunner := system.NewMockCommandRunner()

	// 模拟所有工具都存在
	for _, tool := range []string{
		"command -v smartctl >/dev/null 2>&1",
		"command -v lspci >/dev/null 2>&1",
		"command -v storcli64 >/dev/null 2>&1 || command -v storcli >/dev/null 2>&1",
		"command -v zpool >/dev/null 2>&1",
	} {
		mockCmdRunner.SetMockOutput(tool, "")
	}

	err = checkRequiredTools(mockLogger, mockCmdRunner)
	if err != nil {
		t.Errorf("checkRequiredTools failed unexpectedly: %v", err)
	}

	// 模拟缺少必需工具 smartctl
	mockCmdRunner.SetMockError("command -v smartctl >/dev/null 2>&1", fmt.Errorf("tool not found"))
	err = checkRequiredTools(mockLogger, mockCmdRunner)
	if err == nil {
		t.Errorf("checkRequiredTools should have failed due to missing smartctl")
	}
}

// 测试 runCheck 函数
func TestRunCheck(t *testing.T) {
	mockCmdRunner := system.NewMockCommandRunner()

	// 模拟命令成功
	mockCmdRunner.SetMockOutput("test -f /dev/null", "")
	exitCode := runCheck(mockCmdRunner, "test -f /dev/null")
	if exitCode != 0 {
		t.Errorf("runCheck should have returned 0 for a successful command, got %d", exitCode)
	}

	// 模拟命令失败
	mockCmdRunner.SetMockError("test -f /nonexistent", fmt.Errorf("file not found"))
	exitCode = runCheck(mockCmdRunner, "test -f /nonexistent")
	if exitCode != 1 {
		t.Errorf("runCheck should have returned 1 for a failed command, got %d", exitCode)
	}
}

// 测试 createDummyOutput 函数
func TestIntegrationCreateDummyOutput(t *testing.T) {
	config := model.NewDefaultConfig()
	reason := "Test failure reason"

	// 测试不指定输出文件
	config.OutputFile = ""
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := createDummyOutput(config, reason)
	if err != nil {
		t.Errorf("createDummyOutput failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		t.Errorf("Failed to read from pipe: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, reason) {
		t.Errorf("Output does not contain failure reason: %s", output)
	}

	// 测试指定输出文件
	tempDir, err := os.MkdirTemp("", "test-output")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.OutputFile = filepath.Join(tempDir, "test-output.txt")
	err = createDummyOutput(config, reason)
	if err != nil {
		t.Errorf("createDummyOutput failed when writing to file: %v", err)
	}

	data, err := os.ReadFile(config.OutputFile)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	}

	output = string(data)
	if !strings.Contains(output, reason) {
		t.Errorf("Output file does not contain failure reason: %s", output)
	}
}

// 测试 formatFormatterOptions 函数
func TestIntegrationFormatFormatterOptions(t *testing.T) {
	app := &Application{
		Config: &model.Config{
			NoGroup:      false,
			OutputFormat: model.OutputFormatText,
		},
		Quiet:       false,
		CompactMode: false,
	}

	options := formatFormatterOptions(app)

	if options[output.OptionGroupByType] != true {
		t.Errorf("OptionGroupByType should be true, got %v", options[output.OptionGroupByType])
	}
	if options[output.OptionIncludeSummary] != true {
		t.Errorf("OptionIncludeSummary should be true, got %v", options[output.OptionIncludeSummary])
	}
	if options[output.OptionIncludeTimestamp] != true {
		t.Errorf("OptionIncludeTimestamp should be true, got %v", options[output.OptionIncludeTimestamp])
	}
	if options[output.OptionColorOutput] != true {
		t.Errorf("OptionColorOutput should be true, got %v", options[output.OptionColorOutput])
	}
	if options[output.OptionCompactMode] != false {
		t.Errorf("OptionCompactMode should be false, got %v", options[output.OptionCompactMode])
	}
	if options[output.OptionBorderStyle] != output.BorderStyleClassic {
		t.Errorf("OptionBorderStyle should be BorderStyleClassic, got %v", options[output.OptionBorderStyle])
	}
	if options[output.OptionMaxWidth] != 120 {
		t.Errorf("OptionMaxWidth should be 120, got %v", options[output.OptionMaxWidth])
	}

	// 测试 PDF 格式
	app.Config.OutputFormat = model.OutputFormatPDF
	options = formatFormatterOptions(app)

	if options[output.OptionPaperSize] != output.PaperSizeA4 {
		t.Errorf("OptionPaperSize should be PaperSizeA4, got %v", options[output.OptionPaperSize])
	}
	if options[output.OptionOrientation] != output.OrientationPortrait {
		t.Errorf("OptionOrientation should be OrientationPortrait, got %v", options[output.OptionOrientation])
	}
	if options[output.OptionIncludeCover] != true {
		t.Errorf("OptionIncludeCover should be true, got %v", options[output.OptionIncludeCover])
	}
}
