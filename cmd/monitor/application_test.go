package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/collector"
	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/output"
	"github.com/MaurUppi/disk-health-monitor/internal/storage"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// TestNewApplication 测试 NewApplication 函数
func TestNewApplication(t *testing.T) {
	config := model.NewDefaultConfig()
	options := map[string]interface{}{
		"exit_on_warning": false,
		"only_warnings":   false,
		"quiet":           false,
		"compact":         false,
	}

	app, err := NewApplication(config, options)
	if err != nil {
		t.Errorf("NewApplication() error = %v", err)
		return
	}
	if app == nil {
		t.Errorf("NewApplication() returned nil app")
	}
}

// TestGetBoolOption 测试 getBoolOption 函数
func TestGetBoolOption(t *testing.T) {
	options := map[string]interface{}{
		"key": true,
	}
	result := getBoolOption(options, "key", false)
	if !result {
		t.Errorf("getBoolOption() = %v, want true", result)
	}

	result = getBoolOption(nil, "key", false)
	if result {
		t.Errorf("getBoolOption() = %v, want false", result)
	}
}

// TestApplicationRun 测试 Application.Run 方法
func TestApplicationRun(t *testing.T) {
	config := model.NewDefaultConfig()
	config.CommandTimeout = 10 * time.Second
	//options := map[string]interface{}{
	//	"exit_on_warning": false,
	//	"only_warnings":   false,
	//	"quiet":           false,
	//	"compact":         false,
	//}
	logger := system.NewMockLogger()
	cmdRunner := system.NewMockCommandRunner()
	diskCollector := collector.NewDiskCollector(config, logger, cmdRunner)
	ctrlCollector := collector.NewControllerCollector(cmdRunner, logger)
	historyStorage := storage.NewDiskHistoryStorage(config.DataFile, logger)

	app := &Application{
		Config:         config,
		Logger:         logger,
		CommandRunner:  cmdRunner,
		DiskCollector:  diskCollector,
		CtrlCollector:  ctrlCollector,
		HistoryStorage: historyStorage,
		ExitOnWarning:  false,
		OnlyWarnings:   false,
		Quiet:          false,
		CompactMode:    false,
	}

	exitCode := app.Run()
	if exitCode != 0 {
		t.Errorf("Application.Run() = %d, want 0", exitCode)
	}
}

// TestApplicationGenerateOutput 测试 Application.generateOutput 方法
func TestApplicationGenerateOutput(t *testing.T) {
	config := model.NewDefaultConfig()
	config.OutputFormat = model.OutputFormatText
	logger := system.NewMockLogger()
	cmdRunner := system.NewMockCommandRunner()
	diskCollector := collector.NewDiskCollector(config, logger, cmdRunner)
	ctrlCollector := collector.NewControllerCollector(cmdRunner, logger)
	historyStorage := storage.NewDiskHistoryStorage(config.DataFile, logger)

	app := &Application{
		Config:         config,
		Logger:         logger,
		CommandRunner:  cmdRunner,
		DiskCollector:  diskCollector,
		CtrlCollector:  ctrlCollector,
		HistoryStorage: historyStorage,
		ExitOnWarning:  false,
		OnlyWarnings:   false,
		Quiet:          false,
		CompactMode:    false,
	}

	diskData, _ := diskCollector.Collect(context.Background())
	ctrlData, _ := ctrlCollector.Collect(context.Background())

	formatter, err := output.NewFormatter(string(config.OutputFormat), nil)
	if err != nil {
		t.Errorf("Failed to create formatter: %v", err)
		return
	}

	err = app.generateOutputWithFormatter(diskData, ctrlData, formatter)
	if err != nil {
		t.Errorf("Application.generateOutputWithFormatter() error = %v", err)
	}
}

// 在 application.go 文件中添加以下方法
func (app *Application) generateOutputWithFormatter(diskData *model.DiskData, ctrlData *model.ControllerData, formatter output.OutputFormatter) error {
	// 实现具体的输出逻辑
	formatter.SetData(diskData, ctrlData)
	// 假设保存到文件，文件名可根据实际情况修改
	err := formatter.SaveToFile("output.txt")
	if err != nil {
		return err
	}
	return nil
}

// TestStorageSaveAndLoad 测试存储模块的保存和加载功能
func TestStorageSaveAndLoad(t *testing.T) {
	config := model.NewDefaultConfig()
	logger := system.NewMockLogger()
	historyStorage := storage.NewDiskHistoryStorage(config.DataFile, logger)

	// 模拟磁盘数据
	diskData := map[string]map[string]string{
		"sda": {
			"Data_Read":    "1.0 TB",
			"Data_Written": "2.0 TB",
		},
	}

	// 保存数据
	err := historyStorage.SaveDiskData(diskData)
	if err != nil {
		t.Errorf("Failed to save disk data: %v", err)
	}

	// 加载数据
	loadedData, _, err := historyStorage.LoadDiskData()
	if err != nil {
		t.Errorf("Failed to load disk data: %v", err)
	}

	// 验证数据是否一致
	if len(loadedData) != len(diskData) {
		t.Errorf("Loaded data length %d does not match saved data length %d", len(loadedData), len(diskData))
	}
	for diskName, data := range diskData {
		loadedDiskData, ok := loadedData[diskName]
		if !ok {
			t.Errorf("Disk %s not found in loaded data", diskName)
		}
		for key, value := range data {
			if loadedDiskData[key] != value {
				t.Errorf("Value for %s in disk %s does not match: expected %s, got %s", key, diskName, value, loadedDiskData[key])
			}
		}
	}
}

// TestLogger 测试日志记录器
func TestLogger(t *testing.T) {
	logger := system.NewMockLogger()

	// 记录不同级别的日志
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Error("Error message")

	// 验证日志记录
	if len(logger.DebugLogs) != 1 {
		t.Errorf("Expected 1 debug log, got %d", len(logger.DebugLogs))
	}
	if len(logger.InfoLogs) != 1 {
		t.Errorf("Expected 1 info log, got %d", len(logger.InfoLogs))
	}
	if len(logger.ErrorLogs) != 1 {
		t.Errorf("Expected 1 error log, got %d", len(logger.ErrorLogs))
	}
}

// TestCommandRunner 测试命令执行器
func TestCommandRunner(t *testing.T) {
	cmdRunner := system.NewMockCommandRunner()
	command := "test_command"
	output := "test_output"
	cmdRunner.SetMockOutput(command, output)

	result, err := cmdRunner.Run(context.Background(), command)
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}
	if result != output {
		t.Errorf("Expected output %s, got %s", output, result)
	}
}

// TestMain 运行所有测试
func TestMain(m *testing.M) {
	// 运行所有测试
	exitCode := m.Run()
	// 退出测试
	os.Exit(exitCode)
}
