package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
)

func TestHTMLFormatter_GetSupportedOptions(t *testing.T) {
	formatter := createHTMLFormatter(nil)
	options := formatter.GetSupportedOptions()

	// Check if all expected options are present
	expectedOptions := []string{
		OptionTemperatureBar,
		OptionEnableInteractivity,
		OptionHtmlTitle,
		OptionGroupByType,
		OptionIncludeSummary,
		OptionIncludeTimestamp,
	}

	for _, opt := range expectedOptions {
		if _, exists := options[opt]; !exists {
			t.Errorf("Expected option %s not found in supported options", opt)
		}
	}
}

func TestHTMLFormatter_FormatDiskInfo(t *testing.T) {
	// Create a mock disk data
	diskData := &model.DiskData{
		Disks: []*model.Disk{
			{
				Name:  "sda",
				Type:  model.DiskTypeSASSSD,
				Model: "Samsung SSD 870 EVO",
				Size:  "1 TB",
				Pool:  "tank",
				SMARTData: model.SMARTData{
					"Temperature":     "35",
					"Power_On_Hours":  "8760",
					"Percentage_Used": "5",
					"Smart_Status":    "PASSED",
					"Data_Read":       "10.5 TB",
					"Data_Written":    "5.2 TB",
				},
			},
		},
		GroupedDisks: map[model.DiskType][]*model.Disk{
			model.DiskTypeSASSSD: {
				{
					Name:  "sda",
					Type:  model.DiskTypeSASSSD,
					Model: "Samsung SSD 870 EVO",
					Size:  "1 TB",
					Pool:  "tank",
					SMARTData: model.SMARTData{
						"Temperature":     "35",
						"Power_On_Hours":  "8760",
						"Percentage_Used": "5",
						"Smart_Status":    "PASSED",
						"Data_Read":       "10.5 TB",
						"Data_Written":    "5.2 TB",
					},
				},
			},
		},
	}

	formatter := createHTMLFormatter(nil)
	err := formatter.FormatDiskInfo(diskData)

	if err != nil {
		t.Errorf("Expected no error when formatting disk info, got: %v", err)
	}

	// Check if HTML content was generated
	htmlContent := formatter.htmlBuffer.String()
	if len(htmlContent) == 0 {
		t.Error("Expected HTML content to be generated, got empty string")
	}

	// Check for key HTML elements
	expectedElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"SAS/SATA 固态硬盘",
		"Samsung SSD 870 EVO",
		"tank",
	}

	for _, element := range expectedElements {
		if !strings.Contains(htmlContent, element) {
			t.Errorf("Expected HTML to contain '%s'", element)
		}
	}
}

func TestHTMLFormatter_FormatControllerInfo(t *testing.T) {
	// Create a mock controller data
	controllerData := &model.ControllerData{
		LSIControllers: map[string]*model.LSIController{
			"LSI_Controller_0": {
				Controller: model.Controller{
					Type:            model.ControllerTypeLSI,
					Model:           "LSI 9400-16i",
					FirmwareVersion: "24.00.00.00",
					DriverVersion:   "43.100.00.00",
					Temperature:     "58",
					DeviceCount:     "14",
					Status:          model.ControllerStatusOK,
				},
			},
		},
		NVMeControllers: map[string]*model.NVMeController{
			"NVMe_Controller_01:00.0": {
				Controller: model.Controller{
					Type:        model.ControllerTypeNVMe,
					Description: "Samsung NVMe SSD Controller",
					Bus:         "01:00.0",
					Temperature: "42",
					Status:      model.ControllerStatusOK,
				},
			},
		},
	}

	formatter := createHTMLFormatter(nil)
	err := formatter.FormatControllerInfo(controllerData)

	if err != nil {
		t.Errorf("Expected no error when formatting controller info, got: %v", err)
	}

	// Check if HTML content was generated
	htmlContent := formatter.htmlBuffer.String()
	if len(htmlContent) == 0 {
		t.Error("Expected HTML content to be generated, got empty string")
	}

	// Check for key HTML elements
	expectedElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"LSI SAS HBA控制器",
		"LSI 9400-16i",
		"NVMe控制器",
		"Samsung NVMe SSD Controller",
	}

	for _, element := range expectedElements {
		if !strings.Contains(htmlContent, element) {
			t.Errorf("Expected HTML to contain '%s'", element)
		}
	}
}

func TestHTMLFormatter_SaveToFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "html_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup a formatter with some content
	formatter := createHTMLFormatter(nil)
	formatter.htmlBuffer.WriteString("<!DOCTYPE html><html><body>Test Content</body></html>")

	// Save to file
	tempFile := filepath.Join(tempDir, "test_output.html")
	err = formatter.SaveToFile(tempFile)
	if err != nil {
		t.Errorf("Failed to save HTML to file: %v", err)
	}

	// Verify file exists and contains content
	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}

	if !strings.Contains(string(fileContent), "Test Content") {
		t.Error("File content does not match expected content")
	}
}

func TestHTMLFormatter_formatTemperatureBar(t *testing.T) {
	formatter := createHTMLFormatter(nil)

	// Test with normal temperature
	tempBar := formatter.formatTemperatureBar("40")
	if !strings.Contains(tempBar, "temperature-marker") {
		t.Error("Expected temperature bar to contain marker")
	}
	// A 40C temperature should be at 50% position (40-20)/40*100
	if !strings.Contains(tempBar, "left: 50%") {
		t.Errorf("Expected 40C to be at 50%% position, got: %s", tempBar)
	}

	// Test with low temperature
	tempBar = formatter.formatTemperatureBar("10")
	if !strings.Contains(tempBar, "left: 0%") {
		t.Errorf("Expected 10C (below min) to be at 0%% position, got: %s", tempBar)
	}

	// Test with high temperature
	tempBar = formatter.formatTemperatureBar("70")
	if !strings.Contains(tempBar, "left: 100%") {
		t.Errorf("Expected 70C (above max) to be at 100%% position, got: %s", tempBar)
	}

	// Test with invalid temperature
	tempBar = formatter.formatTemperatureBar("invalid")
	if tempBar != "" {
		t.Errorf("Expected empty string for invalid temperature, got: %s", tempBar)
	}
}
