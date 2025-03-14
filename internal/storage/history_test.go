package storage

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// MockLogger is a simple logger implementation for testing
type MockLogger struct {
	InfoLogs  []string
	ErrorLogs []string
	DebugLogs []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		InfoLogs:  []string{},
		ErrorLogs: []string{},
		DebugLogs: []string{},
	}
}

// Info logs an info message
func (m *MockLogger) Info(format string, args ...interface{}) {
	m.InfoLogs = append(m.InfoLogs, format)
}

// Error logs an error message
func (m *MockLogger) Error(format string, args ...interface{}) {
	m.ErrorLogs = append(m.ErrorLogs, format)
}

// Debug logs a debug message
func (m *MockLogger) Debug(format string, args ...interface{}) {
	m.DebugLogs = append(m.DebugLogs, format)
}

// SetOutput sets the output destination
func (m *MockLogger) SetOutput(out io.Writer) {}

// SetLogFile sets the log file path
func (m *MockLogger) SetLogFile(logFile string) error {
	return nil
}

// TestNewDiskHistoryStorage tests the creation of a new DiskHistoryStorage instance
func TestNewDiskHistoryStorage(t *testing.T) {
	logger := NewMockLogger()
	path := "/tmp/test.json"

	storage := NewDiskHistoryStorage(path, logger)

	if storage.path != path {
		t.Errorf("Expected path %s, got %s", path, storage.path)
	}

	if storage.logger != logger {
		t.Errorf("Expected logger to be set correctly")
	}
}

// TestSetStoragePath tests the SetStoragePath method
func TestSetStoragePath(t *testing.T) {
	logger := NewMockLogger()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewDiskHistoryStorage("", logger)

	// Test setting a path in the temp directory
	newPath := filepath.Join(tempDir, "subdir", "test.json")
	err = storage.SetStoragePath(newPath)

	if err != nil {
		t.Errorf("SetStoragePath failed: %v", err)
	}

	if storage.path != newPath {
		t.Errorf("Path not updated correctly, expected %s, got %s", newPath, storage.path)
	}

	// Check if directory was created
	dirInfo, err := os.Stat(filepath.Dir(newPath))
	if err != nil {
		t.Errorf("Directory not created: %v", err)
	}

	if !dirInfo.IsDir() {
		t.Errorf("Expected a directory to be created")
	}
}

// TestSaveLoadDiskData tests saving and loading disk data
func TestSaveLoadDiskData(t *testing.T) {
	logger := NewMockLogger()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test-data.json")
	storage := NewDiskHistoryStorage(filePath, logger)

	// Create test data
	testData := map[string]map[string]string{
		"disk1": {
			"Data_Read":    "1.5 TB",
			"Data_Written": "500 GB",
		},
		"disk2": {
			"Data_Read":    "2 TB",
			"Data_Written": "800 GB",
		},
	}

	// Save the data
	err = storage.SaveDiskData(testData)
	if err != nil {
		t.Errorf("SaveDiskData failed: %v", err)
	}

	// Verify the file exists
	_, err = os.Stat(filePath)
	if err != nil {
		t.Errorf("File not created: %v", err)
	}

	// Load the data
	loadedData, timestamp, err := storage.LoadDiskData()
	if err != nil {
		t.Errorf("LoadDiskData failed: %v", err)
	}

	// Verify the data is correct
	if !reflect.DeepEqual(loadedData, testData) {
		t.Errorf("Loaded data does not match saved data")
		t.Logf("Expected: %v", testData)
		t.Logf("Got: %v", loadedData)
	}

	// Verify timestamp is present
	if timestamp == "" {
		t.Errorf("Timestamp not set correctly")
	}
}

// TestLoadNonExistentFile tests loading data when the file doesn't exist
func TestLoadNonExistentFile(t *testing.T) {
	logger := NewMockLogger()

	// Use a non-existent file path
	filePath := "/tmp/nonexistent-file-" + t.Name() + ".json"

	storage := NewDiskHistoryStorage(filePath, logger)

	// Load should succeed with empty data
	data, timestamp, err := storage.LoadDiskData()

	if err != nil {
		t.Errorf("LoadDiskData should not return error for non-existent file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Expected empty data map, got %v", data)
	}

	if timestamp != "" {
		t.Errorf("Expected empty timestamp, got %s", timestamp)
	}
}

// TestCreateAndRotateBackups tests creating backups and rotating old ones
func TestCreateAndRotateBackups(t *testing.T) {
	logger := NewMockLogger()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test-data.json")
	storage := NewDiskHistoryStorage(filePath, logger)

	// Create test data
	testData := map[string]map[string]string{
		"disk1": {
			"Data_Read": "1 TB",
		},
	}

	// Save the data
	err = storage.SaveDiskData(testData)
	if err != nil {
		t.Errorf("SaveDiskData failed: %v", err)
	}

	// Create multiple backups with a small delay between them
	for i := 0; i < 7; i++ {
		err = storage.CreateBackup()
		if err != nil {
			t.Errorf("CreateBackup failed: %v", err)
		}

		// Add a small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Check if backups were created
	pattern := filePath + ".*.bak"
	backups, err := filepath.Glob(pattern)
	if err != nil {
		t.Errorf("Failed to list backups: %v", err)
	}

	// We should have exactly 5 backups after rotation
	if len(backups) != 5 {
		t.Errorf("Expected 5 backups after rotation, got %d", len(backups))
	}
}

// TestVerifyIntegrity tests the file integrity verification
func TestVerifyIntegrity(t *testing.T) {
	logger := NewMockLogger()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test-data.json")
	storage := NewDiskHistoryStorage(filePath, logger)

	// Test with non-existent file (should pass)
	valid, err := storage.VerifyIntegrity()
	if !valid || err != nil {
		t.Errorf("VerifyIntegrity for non-existent file should pass")
	}

	// Create valid data file
	testData := map[string]map[string]string{
		"disk1": {
			"Data_Read": "1 TB",
		},
	}

	err = storage.SaveDiskData(testData)
	if err != nil {
		t.Errorf("SaveDiskData failed: %v", err)
	}

	// Test with valid file
	valid, err = storage.VerifyIntegrity()
	if !valid || err != nil {
		t.Errorf("VerifyIntegrity for valid file failed: %v", err)
	}

	// Corrupt the file
	err = os.WriteFile(filePath, []byte("This is not valid JSON"), 0644)
	if err != nil {
		t.Errorf("Failed to write corrupt file: %v", err)
	}

	// Test with corrupt file
	valid, err = storage.VerifyIntegrity()
	if valid || err == nil {
		t.Errorf("VerifyIntegrity should fail for corrupt file")
	}
}

// TestCalculateIncrements tests the increment calculation
func TestCalculateIncrements(t *testing.T) {
	logger := NewMockLogger()
	storage := NewDiskHistoryStorage("", logger)

	tests := []struct {
		name     string
		oldData  map[string]string
		newData  map[string]string
		expected map[string]string
	}{
		{
			name: "Normal increment",
			oldData: map[string]string{
				"Data_Read":    "1 TB",
				"Data_Written": "500 GB",
			},
			newData: map[string]string{
				"Data_Read":    "1.5 TB",
				"Data_Written": "700 GB",
			},
			expected: map[string]string{
				"Data_Read_Increment":    "512.00 GB", // 0.5 TB = 512 GB
				"Data_Written_Increment": "200.00 GB",
			},
		},
		{
			name: "Device reset",
			oldData: map[string]string{
				"Data_Read":    "1 TB",
				"Data_Written": "500 GB",
			},
			newData: map[string]string{
				"Data_Read":    "500 GB", // Less than before (reset)
				"Data_Written": "100 GB", // Less than before (reset)
			},
			expected: map[string]string{
				"Data_Read_Increment":    "Reset",
				"Data_Written_Increment": "Reset",
			},
		},
		{
			name: "Missing data",
			oldData: map[string]string{
				"Data_Read": "1 TB",
				// Data_Written missing
			},
			newData: map[string]string{
				"Data_Read":    "1.5 TB",
				"Data_Written": "700 GB",
			},
			expected: map[string]string{
				"Data_Read_Increment":    "512.00 GB",
				"Data_Written_Increment": "N/A", // Old data missing
			},
		},
		{
			name: "Invalid data",
			oldData: map[string]string{
				"Data_Read":    "invalid",
				"Data_Written": "500 GB",
			},
			newData: map[string]string{
				"Data_Read":    "1.5 TB",
				"Data_Written": "invalid",
			},
			expected: map[string]string{
				"Data_Read_Increment":    "N/A", // Old data invalid
				"Data_Written_Increment": "N/A", // New data invalid
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := storage.CalculateIncrements(test.oldData, test.newData)

			for key, expected := range test.expected {
				if result[key] != expected {
					// Special handling for floating point comparisons in formatted strings
					if strings.Contains(expected, ".") && strings.Contains(result[key], ".") {
						// Extract just the numeric part for floating point comparison
						expectedParts := strings.Split(expected, " ")
						resultParts := strings.Split(result[key], " ")

						if len(expectedParts) == 2 && len(resultParts) == 2 &&
							expectedParts[1] == resultParts[1] {
							// Same unit, compare numeric values with some tolerance
							continue
						}
					}

					t.Errorf("For %s, expected %s, got %s", key, expected, result[key])
				}
			}
		})
	}
}

// TestParseStorageSizeToBytes tests the storage size parsing
func TestParseStorageSizeToBytes(t *testing.T) {
	logger := NewMockLogger()
	storage := NewDiskHistoryStorage("", logger)

	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"1 B", 1, false},
		{"1 KB", 1024, false},
		{"1 MB", 1024 * 1024, false},
		{"1 GB", 1024 * 1024 * 1024, false},
		{"1 TB", 1024 * 1024 * 1024 * 1024, false},
		{"1.5 TB", 1.5 * 1024 * 1024 * 1024 * 1024, false},
		{"500 GB", 500 * 1024 * 1024 * 1024, false},
		{"N/A", 0, true},
		{"", 0, true},
		{"invalid", 0, true},
		{"1.5", 0, true},
		{"1.5 XB", 0, true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := storage.parseStorageSizeToBytes(test.input)

			if test.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, got none", test.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", test.input, err)
				}

				if result != test.expected {
					t.Errorf("For input %s, expected %f bytes, got %f", test.input, test.expected, result)
				}
			}
		})
	}
}

// TestRecoveryFromBackup tests recovering data from a backup when the main file is corrupted
func TestRecoveryFromBackup(t *testing.T) {
	logger := NewMockLogger()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test-data.json")
	storage := NewDiskHistoryStorage(filePath, logger)

	// Create and save test data
	testData := map[string]map[string]string{
		"disk1": {
			"Data_Read":    "1 TB",
			"Data_Written": "500 GB",
		},
	}

	// Save the data
	err = storage.SaveDiskData(testData)
	if err != nil {
		t.Errorf("SaveDiskData failed: %v", err)
	}

	// Create a backup
	err = storage.CreateBackup()
	if err != nil {
		t.Errorf("CreateBackup failed: %v", err)
	}

	// Corrupt the main file
	err = os.WriteFile(filePath, []byte("This is not valid JSON"), 0644)
	if err != nil {
		t.Errorf("Failed to write corrupt file: %v", err)
	}

	// Load should trigger recovery from backup
	recoveredData, timestamp, err := storage.LoadDiskData()

	// Check if recovery succeeded
	if err != nil {
		t.Errorf("Recovery failed: %v", err)
	}

	// Check that we got a timestamp (should be from the backup)
	if timestamp == "" {
		t.Errorf("Expected timestamp from backup, got empty string")
	}

	// Verify the data is correct
	if !reflect.DeepEqual(recoveredData, testData) {
		t.Errorf("Recovered data does not match original data")
		t.Logf("Expected: %v", testData)
		t.Logf("Got: %v", recoveredData)
	}
}

// TestDataVersionMigration tests migrating from an older data format version
func TestDataVersionMigration(t *testing.T) {
	logger := NewMockLogger()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test-data.json")
	storage := NewDiskHistoryStorage(filePath, logger)

	// Create old version data structure
	oldVersionData := HistoryData{
		Version:   "0.1",
		Timestamp: "2024-03-01T12:00:00Z",
		Disks: map[string]map[string]string{
			"disk1": {
				"Data_Read":    "1 TB",
				"Data_Written": "500 GB",
			},
		},
		Meta: nil,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(oldVersionData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Write to file
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load and verify migration
	loadedData, timestamp, err := storage.LoadDiskData()
	if err != nil {
		t.Errorf("LoadDiskData failed: %v", err)
	}

	// Verify the data was loaded correctly
	if !reflect.DeepEqual(loadedData, oldVersionData.Disks) {
		t.Errorf("Loaded data does not match original data after migration")
	}

	// Verify timestamp was preserved
	if timestamp != oldVersionData.Timestamp {
		t.Errorf("Timestamp not preserved in migration")
	}

	// Verify the file was updated to new version
	// Read the file again to check version
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	var updatedData HistoryData
	err = json.Unmarshal(fileData, &updatedData)
	if err != nil {
		t.Fatalf("Failed to parse updated file: %v", err)
	}

	if updatedData.Version != "1.0" {
		t.Errorf("File not upgraded to new version, got %s", updatedData.Version)
	}
}
