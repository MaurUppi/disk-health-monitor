package output

import (
	"reflect"
	"testing"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
)

func TestBaseFormatter_Options(t *testing.T) {
	bf := NewBaseFormatter()

	// 测试默认设置
	if bf.options == nil {
		t.Error("Expected options to be initialized")
	}

	// 测试设置和获取选项
	testCases := []struct {
		name         string
		optionName   string
		optionValue  interface{}
		defaultValue interface{}
		expected     interface{}
	}{
		{"string option", "test_string", "value", "default", "value"},
		{"bool option", "test_bool", true, false, true},
		{"int option", "test_int", 42, 0, 42},
		{"missing option", "missing", nil, "default", "default"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.optionValue != nil {
				bf.SetOption(tc.optionName, tc.optionValue)
			}
			result := bf.GetOption(tc.optionName, tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}

	// 测试类型特定的获取方法
	bf.SetOption("string_opt", "test")
	if bf.GetStringOption("string_opt", "default") != "test" {
		t.Error("GetStringOption failed")
	}

	bf.SetOption("bool_opt", true)
	if !bf.GetBoolOption("bool_opt", false) {
		t.Error("GetBoolOption failed")
	}

	bf.SetOption("int_opt", 123)
	if bf.GetIntOption("int_opt", 0) != 123 {
		t.Error("GetIntOption failed")
	}

	// 测试类型转换
	bf.SetOption("string_int", "456")
	if bf.GetIntOption("string_int", 0) != 456 {
		t.Error("GetIntOption string conversion failed")
	}

	bf.SetOption("float_int", 78.9)
	if bf.GetIntOption("float_int", 0) != 78 {
		t.Error("GetIntOption float conversion failed")
	}
}

func TestFormatDiskStatus(t *testing.T) {
	testCases := []struct {
		status   model.DiskStatus
		expected string
	}{
		{model.DiskStatusOK, "正常"},
		{model.DiskStatusWarning, "警告"},
		{model.DiskStatusError, "错误"},
		{model.DiskStatusUnknown, "未知"},
		{"SomeOtherStatus", "未知"},
	}

	for _, tc := range testCases {
		result := FormatDiskStatus(tc.status)
		if result != tc.expected {
			t.Errorf("FormatDiskStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}
}

func TestFormatSMARTStatus(t *testing.T) {
	testCases := []struct {
		status   string
		expected string
	}{
		{"PASSED", "正常"},
		{"OK", "正常"},
		{"WARNING", "警告"},
		{"警告", "警告"},
		{"FAILED", "错误"},
		{"错误", "错误"},
		{"N/A", "N/A"},
		{"SomeOtherStatus", "SomeOtherStatus"},
	}

	for _, tc := range testCases {
		result := FormatSMARTStatus(tc.status)
		if result != tc.expected {
			t.Errorf("FormatSMARTStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}
}

func TestGetStatusClass(t *testing.T) {
	testCases := []struct {
		status   string
		expected string
	}{
		{"PASSED", "status-ok"},
		{"OK", "status-ok"},
		{"正常", "status-ok"},
		{"WARNING", "status-warning"},
		{"警告", "status-warning"},
		{"FAILED", "status-error"},
		{"错误", "status-error"},
		{"SomeOtherStatus", "status-unknown"},
	}

	for _, tc := range testCases {
		result := GetStatusClass(tc.status)
		if result != tc.expected {
			t.Errorf("GetStatusClass(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}
}

func TestFormatPowerOnHours(t *testing.T) {
	testCases := []struct {
		hours    string
		expected string
	}{
		{"", "N/A"},
		{"N/A", "N/A"},
		{"12", "12h"},
		{"24", "1d"},
		{"48", "2d"},
		{"720", "1m"},
		{"1440", "2m"},
		{"8760", "1y"},
		{"17520", "2y"},
		{"9000", "1y 1m"},
		{"8784", "1y 1d"},
		{"9024", "1y 1m 1d"},
		{"9025", "1y 1m 1d 1h"},
		{"invalid", "invalid"},
	}

	for _, tc := range testCases {
		result := FormatPowerOnHours(tc.hours)
		if result != tc.expected {
			t.Errorf("FormatPowerOnHours(%s): expected %s, got %s", tc.hours, tc.expected, result)
		}
	}
}

func TestBaseFormatter_GetSummaryInfo(t *testing.T) {
	// 创建模拟的磁盘数据
	diskData := model.NewDiskData()
	diskData.CollectedTime = time.Date(2025, 3, 10, 12, 34, 56, 0, time.UTC)

	// 添加一些磁盘
	disk1 := model.NewDisk("sda", "SSD", "Samsung SSD", "1 TB")
	disk1.Status = model.DiskStatusOK
	disk1.Type = model.DiskTypeSASSSD

	disk2 := model.NewDisk("sdb", "SSD", "Samsung SSD", "1 TB")
	disk2.Status = model.DiskStatusWarning
	disk2.Type = model.DiskTypeSASSSD

	disk3 := model.NewDisk("sdc", "HDD", "WD HDD", "4 TB")
	disk3.Status = model.DiskStatusError
	disk3.Type = model.DiskTypeSASHDD

	disk4 := model.NewDisk("nvme0n1", "SSD", "NVMe SSD", "1 TB")
	disk4.Status = model.DiskStatusOK
	disk4.Type = model.DiskTypeNVMESSD

	diskData.AddDisk(disk1)
	diskData.AddDisk(disk2)
	diskData.AddDisk(disk3)
	diskData.AddDisk(disk4)

	// 创建模拟的控制器数据
	controllerData := model.NewControllerData()
	lsiController := model.NewLSIController("LSI_Controller_0")
	nvmeController := model.NewNVMeController("NVMe_Controller_0")
	controllerData.LSIControllers["LSI_Controller_0"] = lsiController
	controllerData.NVMeControllers["NVMe_Controller_0"] = nvmeController

	// 创建格式化器并设置数据
	bf := NewBaseFormatter()
	bf.SetData(diskData, controllerData)

	// 获取摘要信息
	summary := bf.GetSummaryInfo()

	// 验证摘要信息
	expectedSummary := map[string]string{
		"TotalDisks":      "4",
		"SSDCount":        "3", // 2 SAS SSD + 1 NVMe SSD
		"HDDCount":        "1",
		"WarningCount":    "1",
		"ErrorCount":      "1",
		"ControllerCount": "2",
		"CollectionTime":  "2025-03-10 12:34:56",
	}

	if !reflect.DeepEqual(summary, expectedSummary) {
		t.Errorf("GetSummaryInfo: expected %v, got %v", expectedSummary, summary)
	}

	// 测试没有数据的情况
	bf = NewBaseFormatter()
	if len(bf.GetSummaryInfo()) != 0 {
		t.Error("GetSummaryInfo should return empty map when diskData is nil")
	}
}

func TestNewFormatter(t *testing.T) {
	// 存储原始工厂函数
	originalPDFFormatter := NewPDFFormatter
	originalTextFormatter := NewTextFormatter

	// 恢复原始工厂函数
	defer func() {
		NewPDFFormatter = originalPDFFormatter
		NewTextFormatter = originalTextFormatter
	}()

	// 设置模拟工厂函数
	mockPDFFormatter := &MockFormatter{}
	mockTextFormatter := &MockFormatter{}

	NewPDFFormatter = func(options map[string]interface{}) OutputFormatter {
		return mockPDFFormatter
	}

	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		return mockTextFormatter
	}

	// 测试PDF格式
	formatter, err := NewFormatter("pdf", nil)
	if err != nil {
		t.Errorf("NewFormatter(\"pdf\") returned error: %v", err)
	}
	if formatter != mockPDFFormatter {
		t.Error("NewFormatter(\"pdf\") did not return PDF formatter")
	}

	// 测试带大写字母的PDF格式
	formatter, err = NewFormatter("PDF", nil)
	if err != nil {
		t.Errorf("NewFormatter(\"PDF\") returned error: %v", err)
	}
	if formatter != mockPDFFormatter {
		t.Error("NewFormatter(\"PDF\") did not return PDF formatter")
	}

	// 测试文本格式
	formatter, err = NewFormatter("text", nil)
	if err != nil {
		t.Errorf("NewFormatter(\"text\") returned error: %v", err)
	}
	if formatter != mockTextFormatter {
		t.Error("NewFormatter(\"text\") did not return text formatter")
	}

	// 测试文本格式别名
	formatter, err = NewFormatter("txt", nil)
	if err != nil {
		t.Errorf("NewFormatter(\"txt\") returned error: %v", err)
	}
	if formatter != mockTextFormatter {
		t.Error("NewFormatter(\"txt\") did not return text formatter")
	}

	// 测试无效格式
	formatter, err = NewFormatter("invalid", nil)
	if err == nil {
		t.Error("NewFormatter(\"invalid\") did not return error")
	}
	if formatter != nil {
		t.Error("NewFormatter(\"invalid\") returned a formatter")
	}
}

// MockFormatter 用于测试的模拟格式化器
type MockFormatter struct {
	diskData       *model.DiskData
	controllerData *model.ControllerData
	options        map[string]interface{}
}

func (m *MockFormatter) FormatDiskInfo(diskData *model.DiskData) error {
	m.diskData = diskData
	return nil
}

func (m *MockFormatter) FormatControllerInfo(controllerData *model.ControllerData) error {
	m.controllerData = controllerData
	return nil
}

func (m *MockFormatter) SaveToFile(filename string) error {
	return nil
}

func (m *MockFormatter) SetOption(name string, value interface{}) error {
	if m.options == nil {
		m.options = make(map[string]interface{})
	}
	m.options[name] = value
	return nil
}

func (m *MockFormatter) GetSupportedOptions() map[string]string {
	return map[string]string{
		"test_option": "Test option description",
	}
}

func (m *MockFormatter) SetData(diskData *model.DiskData, controllerData *model.ControllerData) {
	m.diskData = diskData
	m.controllerData = controllerData
}
