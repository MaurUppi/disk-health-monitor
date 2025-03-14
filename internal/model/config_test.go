package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	// 获取默认配置
	config := NewDefaultConfig()

	// 验证默认值
	if config.Debug {
		t.Error("Default Debug should be false")
	}

	if config.Verbose {
		t.Error("Default Verbose should be false")
	}

	if config.OutputFormat != OutputFormatPDF {
		t.Errorf("Default OutputFormat should be PDF, got %s", config.OutputFormat)
	}

	if config.CommandTimeout != 30*time.Second {
		t.Errorf("Default CommandTimeout should be 30s, got %v", config.CommandTimeout)
	}

	// 验证日志路径
	if !strings.Contains(config.LogFile, "disk_health_monitor.log") {
		t.Errorf("Default LogFile should contain 'disk_health_monitor.log', got %s", config.LogFile)
	}

	// 验证数据文件路径
	if !strings.Contains(config.DataFile, "disk_health_monitor_data.json") {
		t.Errorf("Default DataFile should contain 'disk_health_monitor_data.json', got %s", config.DataFile)
	}
}

func TestConfig_Validate(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建有效配置
	validConfig := &Config{
		LogFile:        filepath.Join(tempDir, "log.txt"),
		DataFile:       filepath.Join(tempDir, "data.json"),
		OutputFormat:   OutputFormatPDF,
		OutputEncoding: "utf8",
	}

	// 测试有效配置
	if err := validConfig.Validate(); err != nil {
		t.Errorf("Validation failed for valid config: %v", err)
	}

	// 验证目录是否已创建
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}

	// 测试无效的输出格式
	invalidFormat := &Config{
		LogFile:        filepath.Join(tempDir, "log.txt"),
		DataFile:       filepath.Join(tempDir, "data.json"),
		OutputFormat:   "invalid",
		OutputEncoding: "utf8",
	}

	if err := invalidFormat.Validate(); err == nil {
		t.Error("Expected error for invalid output format, got nil")
	} else if !strings.Contains(err.Error(), "不支持的输出格式") {
		t.Errorf("Unexpected error message: %v", err)
	}

	// 测试无效的输出编码
	invalidEncoding := &Config{
		LogFile:        filepath.Join(tempDir, "log.txt"),
		DataFile:       filepath.Join(tempDir, "data.json"),
		OutputFormat:   OutputFormatPDF,
		OutputEncoding: "invalid",
	}

	if err := invalidEncoding.Validate(); err == nil {
		t.Error("Expected error for invalid output encoding, got nil")
	} else if !strings.Contains(err.Error(), "不支持的输出编码") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestConfig_SetupOutputFile(t *testing.T) {
	// 创建配置对象
	config := &Config{
		OutputFormat: OutputFormatPDF,
	}

	// 测试未指定输出文件的情况
	config.SetupOutputFile()

	if config.OutputFile == "" {
		t.Error("Expected non-empty OutputFile after SetupOutputFile")
	}

	if !strings.HasSuffix(config.OutputFile, ".pdf") {
		t.Errorf("Expected PDF file extension, got %s", config.OutputFile)
	}

	// 测试已指定输出文件的情况
	predefinedFile := "test_output.pdf"
	config.OutputFile = predefinedFile
	config.SetupOutputFile()

	if config.OutputFile != predefinedFile {
		t.Errorf("Expected OutputFile to remain %s, got %s", predefinedFile, config.OutputFile)
	}

	// 测试不同格式
	config = &Config{
		OutputFormat: OutputFormatText,
		OutputFile:   "",
	}
	config.SetupOutputFile()

	if !strings.HasSuffix(config.OutputFile, ".txt") {
		t.Errorf("Expected TXT file extension, got %s", config.OutputFile)
	}

	config = &Config{
		OutputFormat: OutputFormatJSON,
		OutputFile:   "",
	}
	config.SetupOutputFile()

	if !strings.HasSuffix(config.OutputFile, ".json") {
		t.Errorf("Expected JSON file extension, got %s", config.OutputFile)
	}
}
