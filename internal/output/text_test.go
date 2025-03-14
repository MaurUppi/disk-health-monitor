package output

import (
	"strings"
	"testing"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
)

func createTestDiskData() *model.DiskData {
	// Create mock disk data
	diskData := model.NewDiskData()
	diskData.CollectedTime = time.Date(2025, 3, 10, 12, 34, 56, 0, time.UTC)

	// Add SAS SSD disks
	disk1 := model.NewDisk("sda", "SSD", "Samsung SSD 870 EVO", "1 TB")
	disk1.Status = model.DiskStatusOK
	disk1.Type = model.DiskTypeSASSSD
	disk1.Pool = "tank"
	disk1.SMARTData = model.SMARTData{
		"Temperature":      "32",
		"Trip_Temperature": "70",
		"Power_On_Hours":   "9025", // 1y 1m 1d 1h
		"Power_Cycles":     "120",
		"Percentage_Used":  "12",
		"Smart_Status":     "PASSED",
		"Data_Read":        "12.5 TB",
		"Data_Written":     "8.2 TB",
		"Non_Medium_Errors": "0",
		"Uncorrected_Errors": "0",
	}
	disk1.ReadIncrement = "125.8 GB"
	disk1.WriteIncrement = "84.3 GB"

	disk2 := model.NewDisk("sdb", "SSD", "Samsung SSD 870 EVO", "1 TB")
	disk2.Status = model.DiskStatusWarning
	disk2.Type = model.DiskTypeSASSSD
	disk2.Pool = "tank"
	disk2.SMARTData = model.SMARTData{
		"Temperature":      "35",
		"Trip_Temperature": "70",
		"Power_On_Hours":   "9048", // 1y 1m 2d
		"Power_Cycles":     "125",
		"Percentage_Used":  "15",
		"Smart_Status":     "WARNING",
		"Data_Read":        "13.1 TB",
		"Data_Written":     "9.7 TB",
		"Non_Medium_Errors": "2",
		"Uncorrected_Errors": "0",
	}
	disk2.ReadIncrement = "132.4 GB"
	disk2.WriteIncrement = "89.5 GB"

	// Add SAS HDD disks
	disk3 := model.NewDisk("sdc", "HDD", "WDC WD40EFRX-68N", "4 TB")
	disk3.Status = model.DiskStatusOK
	disk3.Type = model.DiskTypeSASHDD
	disk3.Pool = "data"
	disk3.SMARTData = model.SMARTData{
		"Temperature":      "34",
		"Trip_Temperature": "68",
		"Power_On_Hours":   "19512", // 2y 5m 12d
		"Power_Cycles":     "98",
		"Smart_Status":     "PASSED",
		"Data_Read":        "45.2 TB",
		"Data_Written":     "22.8 TB",
		"Non_Medium_Errors": "0",
		"Uncorrected_Errors": "0",
	}
	disk3.ReadIncrement = "12.3 GB"
	disk3.WriteIncrement = "8.7 GB"

	disk4 := model.NewDisk("sdd", "HDD", "WDC WD40EFRX-68N", "4 TB")
	disk4.Status = model.DiskStatusError
	disk4.Type = model.DiskTypeSASHDD
	disk4.Pool = "data"
	disk4.SMARTData = model.SMARTData{
		"Temperature":      "43",
		"Trip_Temperature": "68",
		"Power_On_Hours":   "19632", // 2y 5m 15d
		"Power_Cycles":     "105",
		"Smart_Status":     "FAILED",
		"Data_Read":        "48.7 TB",
		"Data_Written":     "25.1 TB",
		"Non_Medium_Errors": "0",
		"Uncorrected_Errors": "2",
	}
	disk4.ReadIncrement = "14.5 GB"
	disk4.WriteIncrement = "9.2 GB"

	// Add NVMe SSD disk
	disk5 := model.NewDisk("nvme0n1", "SSD", "Samsung SSD 980 PRO", "1 TB")
	disk5.Status = model.DiskStatusOK
	disk5.Type = model.DiskTypeNVMESSD
	disk5.Pool = "cache"
	disk5.SMARTData = model.SMARTData{
		"Temperature":        "38",
		"Warning_Temperature": "70",
		"Critical_Temperature": "80",
		"Power_On_Hours":     "6135", // 8m 15d
		"Power_Cycles":       "45",
		"Percentage_Used":    "5",
		"Available_Spare":    "100",
		"Smart_Status":       "PASSED",
		"Data_Read":          "8.5 TB",
		"Data_Written":       "12.3 TB",
	}
	disk5.ReadIncrement = "345.8 GB"
	disk5.WriteIncrement = "532.7 GB"

	// Add disks to data
	diskData.AddDisk(disk1)
	diskData.AddDisk(disk2)
	diskData.AddDisk(disk3)
	diskData.AddDisk(disk4)
	diskData.AddDisk(disk5)

	// Sort disks
	diskData.SortDisks()

	// Set previous data
	diskData.SetPreviousData(map[string]map[string]string{
		"sda": {
			"Data_Read":    "12.37 TB",
			"Data_Written": "8.12 TB",
		},
		"sdb": {
			"Data_Read":    "12.97 TB",
			"Data_Written": "9.61 TB",
		},
		"sdc": {
			"Data_Read":    "45.19 TB",
			"Data_Written": "22.79 TB",
		},
		"sdd": {
			"Data_Read":    "48.69 TB",
			"Data_Written": "25.09 TB",
		},
		"nvme0n1": {
			"Data_Read":    "8.16 TB",
			"Data_Written": "11.77 TB",
		},
	}, "2025-03-09 12:34:56")

	return diskData
}

func createTestControllerData() *model.ControllerData {
	// Create mock controller data
	controllerData := model.NewControllerData()

	// Add LSI controller
	lsiController := model.NewLSIController("LSI_Controller_0")
	lsiController.Model = "LSI SAS 9300-8i"
	lsiController.FirmwareVersion = "16.00.01.00"
	lsiController.DriverVersion = "7.705.18.00-rh8.1"
	lsiController.Temperature = "58"
	lsiController.ROCTemperature = "58"
	lsiController.DeviceCount = "8"
	lsiController.Status = model.ControllerStatusOK
	lsiController.SSDCount = "2"
	lsiController.HDDCount = "6"
	controllerData.LSIControllers["LSI_Controller_0"] = lsiController

	// Add NVMe controller
	nvmeController := model.NewNVMeController("NVMe_Controller_0")
	nvmeController.Bus = "0000:01:00.0"
	nvmeController.PCIAddress = "0000:01:00.0"
	nvmeController.Description = "Non-Volatile memory controller: Samsung Electronics Co Ltd NVMe SSD Controller 980 PRO"
	nvmeController.Model = "Samsung Electronics Co Ltd NVMe SSD Controller 980 PRO"
	nvmeController.Temperature = "42"
	nvmeController.Status = model.ControllerStatusOK
	controllerData.NVMeControllers["NVMe_Controller_0"] = nvmeController

	return controllerData
}

func TestTextFormatter_CreateAndFormat(t *testing.T) {
	// 设置变量的值，让测试能够继续
	originalNewTextFormatter := NewTextFormatter
	defer func() { NewTextFormatter = originalNewTextFormatter }()
	
	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		return createTextFormatter(options)
	}
	
	// Create formatter with default options
	formatter, err := NewFormatter("text", nil)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	// Test getting supported options
	options := formatter.GetSupportedOptions()
	if len(options) == 0 {
		t.Error("Expected non-empty options map")
	}

	// Create test data
	diskData := createTestDiskData()
	controllerData := createTestControllerData()

	// Format disk info
	err = formatter.FormatDiskInfo(diskData)
	if err != nil {
		t.Errorf("FormatDiskInfo failed: %v", err)
	}

	// Format controller info
	err = formatter.FormatControllerInfo(controllerData)
	if err != nil {
		t.Errorf("FormatControllerInfo failed: %v", err)
	}

	// Get output as string
	textFormatter, ok := formatter.(*TextFormatter)
	if !ok {
		t.Fatal("Failed to cast formatter to TextFormatter")
	}

	output := textFormatter.String()
	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Check for expected sections
	expectedSections := []string{
		"=== TrueNAS磁盘健康监控 ===",
		"系统摘要:",
		"--- SAS/SATA 固态硬盘 ---",
		"--- SAS/SATA 机械硬盘 ---",
		"--- NVMe 固态硬盘 ---",
		"--- LSI SAS HBA控制器 ---",
		"--- NVMe控制器 ---",
		"--- 磁盘读写增量信息",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Output missing expected section: %s", section)
		}
	}
}

func TestTextFormatter_BorderStyles(t *testing.T) {
	// 设置变量的值，让测试能够继续
	originalNewTextFormatter := NewTextFormatter
	defer func() { NewTextFormatter = originalNewTextFormatter }()
	
	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		return createTextFormatter(options)
	}
	
	// Test different border styles
	borderStyles := []string{
		BorderStyleClassic,
		BorderStyleSimple,
		BorderStyleNone,
	}

	diskData := createTestDiskData()
	controllerData := createTestControllerData()

	for _, style := range borderStyles {
		t.Run(style, func(t *testing.T) {
			options := map[string]interface{}{
				OptionBorderStyle: style,
			}

			formatter, err := NewFormatter("text", options)
			if err != nil {
				t.Fatalf("Failed to create formatter: %v", err)
			}

			// Format data
			formatter.FormatDiskInfo(diskData)
			formatter.FormatControllerInfo(controllerData)

			// Get output
			textFormatter, ok := formatter.(*TextFormatter)
			if !ok {
				t.Fatal("Failed to cast formatter to TextFormatter")
			}

			output := textFormatter.String()
			if output == "" {
				t.Error("Expected non-empty output")
			}

			// Specific checks based on border style
			switch style {
			case BorderStyleNone:
				if strings.Contains(output, "+") || strings.Contains(output, "-+") {
					t.Error("BorderStyleNone should not contain border characters")
				}
			case BorderStyleSimple:
				if !strings.Contains(output, "|") {
					t.Error("BorderStyleSimple should contain vertical border characters")
				}
			case BorderStyleClassic:
				if !strings.Contains(output, "+") || !strings.Contains(output, "-+") {
					t.Error("BorderStyleClassic should contain classic border characters")
				}
			}
		})
	}
}

func TestTextFormatter_CompactMode(t *testing.T) {
	// 设置变量的值，让测试能够继续
	originalNewTextFormatter := NewTextFormatter
	defer func() { NewTextFormatter = originalNewTextFormatter }()
	
	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		return createTextFormatter(options)
	}
	
	// Test compact mode
	options := map[string]interface{}{
		OptionCompactMode: true,
	}

	formatter, err := NewFormatter("text", options)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	diskData := createTestDiskData()
	controllerData := createTestControllerData()

	// Format data
	formatter.FormatDiskInfo(diskData)
	formatter.FormatControllerInfo(controllerData)

	// Get output
	textFormatter, ok := formatter.(*TextFormatter)
	if !ok {
		t.Fatal("Failed to cast formatter to TextFormatter")
	}

	output := textFormatter.String()
	
	// In compact mode, we should have fewer columns
	// Check that some columns are missing
	if strings.Contains(output, "Percentage_Used") {
		t.Error("Compact mode should not contain Percentage_Used column")
	}
	
	// Check that essential columns are still present
	essentialColumns := []string{"名称", "温度", "通电时间", "SMART状态"}
	for _, col := range essentialColumns {
		if !strings.Contains(output, col) {
			t.Errorf("Compact mode missing essential column: %s", col)
		}
	}
}

func TestTextFormatter_NoGroup(t *testing.T) {
	// 设置变量的值，让测试能够继续
	originalNewTextFormatter := NewTextFormatter
	defer func() { NewTextFormatter = originalNewTextFormatter }()
	
	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		return createTextFormatter(options)
	}
	
	// Test without grouping
	options := map[string]interface{}{
		OptionGroupByType: false,
	}

	formatter, err := NewFormatter("text", options)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	diskData := createTestDiskData()

	// Format disk info
	formatter.FormatDiskInfo(diskData)

	// Get output
	textFormatter, ok := formatter.(*TextFormatter)
	if !ok {
		t.Fatal("Failed to cast formatter to TextFormatter")
	}

	output := textFormatter.String()
	
	// Should not have disk type headings
	if strings.Contains(output, "--- SAS/SATA 固态硬盘 ---") {
		t.Error("No group mode should not contain disk type headings")
	}
	
	// Should have a combined heading
	if !strings.Contains(output, "--- 所有磁盘 ---") {
		t.Error("No group mode should contain '所有磁盘' heading")
	}
}

func TestTextFormatter_EdgeCases(t *testing.T) {
	// 设置变量的值，让测试能够继续
	originalNewTextFormatter := NewTextFormatter
	defer func() { NewTextFormatter = originalNewTextFormatter }()
	
	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		return createTextFormatter(options)
	}
	
	formatter, err := NewFormatter("text", nil)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	// Test with nil disk data
	err = formatter.FormatDiskInfo(nil)
	if err == nil {
		t.Error("FormatDiskInfo should return error with nil disk data")
	}

	// Test with nil controller data
	err = formatter.FormatControllerInfo(nil)
	if err == nil {
		t.Error("FormatControllerInfo should return error with nil controller data")
	}

	// Test with empty disk data
	emptyDiskData := model.NewDiskData()
	err = formatter.FormatDiskInfo(emptyDiskData)
	if err != nil {
		t.Errorf("FormatDiskInfo should not return error with empty disk data: %v", err)
	}

	// Test with empty controller data
	emptyControllerData := model.NewControllerData()
	err = formatter.FormatControllerInfo(emptyControllerData)
	if err != nil {
		t.Errorf("FormatControllerInfo should not return error with empty controller data: %v", err)
	}
}
