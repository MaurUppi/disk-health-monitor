package system

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultLogger_Levels(t *testing.T) {
	// 创建一个缓冲区来捕获输出
	var buf bytes.Buffer
	
	// 创建设置为Debug级别的Logger
	logger := &DefaultLogger{
		level:     LogLevelDebug,
		verbose:   true,
		out:       &buf,
	}
	
	// 测试Debug级别
	logger.Debug("Debug message")
	if !strings.Contains(buf.String(), "Debug message") {
		t.Errorf("Expected debug message in log, got: %s", buf.String())
	}
	buf.Reset()
	
	// 测试Info级别
	logger.Info("Info message")
	if !strings.Contains(buf.String(), "Info message") {
		t.Errorf("Expected info message in log, got: %s", buf.String())
	}
	buf.Reset()
	
	// 测试Error级别
	logger.Error("Error message")
	// 由于Error直接写入stderr，这里不检查buf
	
	// 测试Info级别日志记录器
	logger = &DefaultLogger{
		level:     LogLevelInfo,
		verbose:   true,
		out:       &buf,
	}
	
	// Debug消息不应该被记录
	logger.Debug("Debug message")
	if buf.String() != "" {
		t.Errorf("Unexpected debug message in log: %s", buf.String())
	}
	
	// Info消息应该被记录
	logger.Info("Info message")
	if !strings.Contains(buf.String(), "Info message") {
		t.Errorf("Expected info message in log, got: %s", buf.String())
	}
	buf.Reset()
	
	// 测试Error级别日志记录器
	logger = &DefaultLogger{
		level:     LogLevelError,
		verbose:   false,
		out:       &buf,
	}
	
	// Debug和Info消息不应该被记录
	logger.Debug("Debug message")
	logger.Info("Info message")
	if buf.String() != "" {
		t.Errorf("Unexpected message in log: %s", buf.String())
	}
}

func TestDefaultLogger_LogFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// 设置日志文件路径
	logFile := filepath.Join(tempDir, "test.log")
	
	// 创建一个logger并设置日志文件
	logger := NewLogger(logFile, LogLevelDebug, false)
	defer logger.Close()
	
	// 写入一些日志
	logger.Debug("Debug log message")
	logger.Info("Info log message")
	logger.Error("Error log message")
	
	// 关闭日志以确保写入
	logger.Close()
	
	// 读取日志文件内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	// 验证日志内容
	logContent := string(content)
	
	if !strings.Contains(logContent, "Debug log message") {
		t.Error("Log file does not contain debug message")
	}
	
	if !strings.Contains(logContent, "Info log message") {
		t.Error("Log file does not contain info message")
	}
	
	if !strings.Contains(logContent, "Error log message") {
		t.Error("Log file does not contain error message")
	}
}

func TestMockLogger(t *testing.T) {
	// 创建mock logger
	mock := NewMockLogger()
	
	// 测试日志记录
	mock.Debug("Debug %d", 1)
	mock.Info("Info %d", 2)
	mock.Error("Error %d", 3)
	
	// 验证日志被正确记录
	if len(mock.DebugLogs) != 1 || mock.DebugLogs[0] != "Debug 1" {
		t.Errorf("Expected 'Debug 1' in debug logs, got: %v", mock.DebugLogs)
	}
	
	if len(mock.InfoLogs) != 1 || mock.InfoLogs[0] != "Info 2" {
		t.Errorf("Expected 'Info 2' in info logs, got: %v", mock.InfoLogs)
	}
	
	if len(mock.ErrorLogs) != 1 || mock.ErrorLogs[0] != "Error 3" {
		t.Errorf("Expected 'Error 3' in error logs, got: %v", mock.ErrorLogs)
	}
	
	// 测试清除功能
	mock.Clear()
	
	if len(mock.DebugLogs) != 0 || len(mock.InfoLogs) != 0 || len(mock.ErrorLogs) != 0 {
		t.Error("Clear() did not remove all logs")
	}
}
