package storage

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// HistoryData represents the structure of the stored historical data
type HistoryData struct {
	Version   string                       `json:"version"`   // Data format version
	Timestamp string                       `json:"timestamp"` // Data collection timestamp
	Disks     map[string]map[string]string `json:"disks"`     // Disk data mapping
	Meta      map[string]interface{}       `json:"meta"`      // Metadata (extensible fields)
}

// HistoryStorage defines the interface for historical data storage operations
type HistoryStorage interface {
	// SaveDiskData persists disk data to a file
	SaveDiskData(data map[string]map[string]string) error

	// LoadDiskData loads disk data from a file, returns data and timestamp
	LoadDiskData() (data map[string]map[string]string, timestamp string, err error)

	// CalculateIncrements calculates increments between old and new data
	CalculateIncrements(oldData, newData map[string]string) map[string]string

	// SetStoragePath sets the storage file path
	SetStoragePath(path string) error

	// CreateBackup creates a backup of the current data file
	CreateBackup() error

	// VerifyIntegrity checks the integrity of the storage file
	VerifyIntegrity() (bool, error)
}

// DiskHistoryStorage implements the HistoryStorage interface
type DiskHistoryStorage struct {
	path   string        // Path to the data file
	logger system.Logger // Logger for recording operations
}

// NewDiskHistoryStorage creates a new instance of DiskHistoryStorage
func NewDiskHistoryStorage(path string, logger system.Logger) *DiskHistoryStorage {
	return &DiskHistoryStorage{
		path:   path,
		logger: logger,
	}
}

// SetStoragePath sets the storage file path
func (s *DiskHistoryStorage) SetStoragePath(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	s.path = path
	return nil
}

// SaveDiskData saves disk data to the storage file
func (s *DiskHistoryStorage) SaveDiskData(data map[string]map[string]string) error {
	// Create backup before saving new data
	_ = s.CreateBackup() // Ignore backup errors to prioritize saving current data

	// Build data structure
	historyData := HistoryData{
		Version:   "1.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Disks:     data,
		Meta:      map[string]interface{}{"generator": "disk-health-monitor"},
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(historyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// Use temporary file for safe writing
	tempFile := s.path + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename to ensure data integrity
	if err := os.Rename(tempFile, s.path); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	s.logger.Info("Successfully saved disk data to %s", s.path)
	return nil
}

// LoadDiskData loads disk data from the storage file
func (s *DiskHistoryStorage) LoadDiskData() (map[string]map[string]string, string, error) {
	// Check if file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		s.logger.Info("History file %s does not exist, returning empty data", s.path)
		return make(map[string]map[string]string), "", nil
	}

	// Read file
	fileData, err := os.ReadFile(s.path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read history file: %w", err)
	}

	// Try to parse JSON
	var historyData HistoryData
	if err := json.Unmarshal(fileData, &historyData); err != nil {
		// Parse failed, try recovery
		s.logger.Error("Failed to parse history file: %v", err)
		return s.attemptRecovery(err)
	}

	// Version compatibility check
	if historyData.Version != "1.0" {
		s.logger.Info("Data format version %s detected, migrating to current version", historyData.Version)
		historyData = s.migrateDataFormat(historyData)

		// Save migrated data back to file
		jsonData, err := json.MarshalIndent(historyData, "", "  ")
		if err != nil {
			s.logger.Error("Failed to serialize migrated data: %v", err)
		} else {
			// Write migrated data back to file
			if err := os.WriteFile(s.path, jsonData, 0644); err != nil {
				s.logger.Error("Failed to save migrated data: %v", err)
			} else {
				s.logger.Info("Successfully migrated data format from %s to 1.0", historyData.Version)
			}
		}
	}

	s.logger.Info("Successfully loaded disk data from %s, timestamp: %s", s.path, historyData.Timestamp)
	return historyData.Disks, historyData.Timestamp, nil
}

// attemptRecovery tries to recover data from a backup if the main file is corrupted
func (s *DiskHistoryStorage) attemptRecovery(originalErr error) (map[string]map[string]string, string, error) {
	s.logger.Error("Data file corrupt, attempting recovery from backup: %v", originalErr)

	// Find backup files
	pattern := s.path + ".*.bak"
	backups, err := filepath.Glob(pattern)
	if err != nil || len(backups) == 0 {
		return make(map[string]map[string]string), "", fmt.Errorf("no backups found for recovery: %w", originalErr)
	}

	// Sort by name (timestamp from newest to oldest)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	// Try to load from newest backup
	for _, backup := range backups {
		data, timestamp, err := s.loadFromFile(backup)
		if err == nil {
			s.logger.Info("Successfully recovered data from backup: %s", backup)

			// Recovery successful, update main file
			if data, err := os.ReadFile(backup); err == nil {
				_ = os.WriteFile(s.path, data, 0644)
			}

			return data, timestamp, nil
		}
		s.logger.Debug("Failed to recover from backup %s: %v", backup, err)
	}

	// All backups failed
	return make(map[string]map[string]string), "", fmt.Errorf("all backup recovery attempts failed: %w", originalErr)
}

// loadFromFile loads data from a specific file
func (s *DiskHistoryStorage) loadFromFile(filePath string) (map[string]map[string]string, string, error) {
	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse JSON
	var historyData HistoryData
	if err := json.Unmarshal(fileData, &historyData); err != nil {
		return nil, "", fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	return historyData.Disks, historyData.Timestamp, nil
}

// migrateDataFormat migrates data from older versions to the current format
func (s *DiskHistoryStorage) migrateDataFormat(oldData HistoryData) HistoryData {
	// Implement migration strategy based on version
	switch oldData.Version {
	case "0.1":
		return s.migrateFrom01To10(oldData)
	case "0.2":
		return s.migrateFrom02To10(oldData)
	default:
		// Unknown version, try compatible handling
		s.logger.Error("Unknown data version: %s, attempting compatible handling", oldData.Version)
		// Set the version to current
		oldData.Version = "1.0"
		return oldData
	}
}

// migrateFrom01To10 migrates data from version 0.1 to 1.0
func (s *DiskHistoryStorage) migrateFrom01To10(oldData HistoryData) HistoryData {
	// Version 0.1 to 1.0 migration logic
	newData := HistoryData{
		Version:   "1.0",
		Timestamp: oldData.Timestamp,
		Disks:     oldData.Disks,
		Meta:      map[string]interface{}{"migrated_from": "0.1"},
	}

	// If there were specific changes between 0.1 and 1.0, handle them here

	return newData
}

// migrateFrom02To10 migrates data from version 0.2 to 1.0
func (s *DiskHistoryStorage) migrateFrom02To10(oldData HistoryData) HistoryData {
	// Version 0.2 to 1.0 migration logic
	newData := HistoryData{
		Version:   "1.0",
		Timestamp: oldData.Timestamp,
		Disks:     oldData.Disks,
		Meta:      map[string]interface{}{"migrated_from": "0.2"},
	}

	// If there were specific changes between 0.2 and 1.0, handle them here

	return newData
}

// CreateBackup creates a backup of the current data file
func (s *DiskHistoryStorage) CreateBackup() error {
	// Check if source file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		s.logger.Debug("No file to backup at %s", s.path)
		return nil // No need to backup
	}

	// Create timestamp for backup filename with microseconds to ensure uniqueness
	timestamp := time.Now().Format("20060102150405.000000")
	backupPath := fmt.Sprintf("%s.%s.bak", s.path, timestamp)

	// Copy file content
	input, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to backup file
	if err := os.WriteFile(backupPath, input, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	s.logger.Info("Created backup at %s", backupPath)

	// Rotate old backups (keep last 5)
	return s.rotateBackups(5)
}

// rotateBackups removes old backups, keeping only the most recent n
func (s *DiskHistoryStorage) rotateBackups(keep int) error {
	// Find all backups
	pattern := s.path + ".*.bak"
	backups, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// If we have fewer backups than the limit, do nothing
	if len(backups) <= keep {
		return nil
	}

	// Sort backups by modification time (newest to oldest)
	sort.Slice(backups, func(i, j int) bool {
		fi, _ := os.Stat(backups[i])
		fj, _ := os.Stat(backups[j])
		return fi.ModTime().After(fj.ModTime())
	})

	// Remove oldest backups beyond the keep limit
	for i := 0; i < len(backups)-keep; i++ {
		if err := os.Remove(backups[i]); err != nil {
			s.logger.Error("Failed to remove old backup %s: %v", backups[i], err)
			// Continue with other backups despite this error
		} else {
			s.logger.Debug("Removed old backup %s", backups[i])
		}
	}

	return nil
}

// VerifyIntegrity checks if the data file is valid and readable
func (s *DiskHistoryStorage) VerifyIntegrity() (bool, error) {
	// Check if file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return true, nil // No file means no corruption
	}

	// Try to read and parse file
	fileData, err := os.ReadFile(s.path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	var historyData HistoryData
	if err := json.Unmarshal(fileData, &historyData); err != nil {
		return false, fmt.Errorf("file is corrupted: %w", err)
	}

	// Additional checks could be added here (schema validation, etc.)

	return true, nil
}

// CalculateIncrements calculates increments between old and new data values
func (s *DiskHistoryStorage) CalculateIncrements(oldData, newData map[string]string) map[string]string {
	increments := make(map[string]string)

	// 处理读写数据增量
	for _, key := range []string{"Data_Read", "Data_Written"} {
		newValue, newExists := newData[key]
		oldValue, oldExists := oldData[key]

		// 检查两个数据点是否都存在
		if !newExists || !oldExists {
			increments[key+"_Increment"] = "N/A"
			continue
		}

		// 解析为字节数
		newBytes, newErr := s.parseStorageSizeToBytes(newValue)
		oldBytes, oldErr := s.parseStorageSizeToBytes(oldValue)

		// 检查解析错误
		if newErr != nil || oldErr != nil {
			s.logger.Debug("Parse error calculating increment for %s: old=%s, new=%s", key, oldValue, newValue)
			increments[key+"_Increment"] = "N/A"
			continue
		}

		// 计算差值
		if newBytes >= oldBytes {
			// 正常情况 - 新值大于等于旧值
			diffBytes := newBytes - oldBytes
			increments[key+"_Increment"] = s.formatBytes(diffBytes)
		} else {
			// 可能是设备重置或更换
			increments[key+"_Increment"] = "重置"
		}
	}

	return increments
}

// 解析存储大小为字节数
func (s *DiskHistoryStorage) parseStorageSizeToBytes(sizeStr string) (float64, error) {
	if sizeStr == "" || sizeStr == "N/A" {
		return 0, fmt.Errorf("invalid size string: %s", sizeStr)
	}

	// 使用正则表达式提取数字和单位
	re := regexp.MustCompile(`(\d+\.?\d*)\s*([KMGTP]?B)`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("unable to parse size: %s", sizeStr)
	}

	valueStr := matches[1]
	unit := matches[2]

	// 解析数字值
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", valueStr)
	}

	// 转换为字节数
	var bytes float64
	switch strings.ToUpper(unit) {
	case "B":
		bytes = value
	case "KB":
		bytes = value * 1024
	case "MB":
		bytes = value * 1024 * 1024
	case "GB":
		bytes = value * 1024 * 1024 * 1024
	case "TB":
		bytes = value * 1024 * 1024 * 1024 * 1024
	case "PB":
		bytes = value * 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return bytes, nil
}

// 格式化字节为可读字符串
func (s *DiskHistoryStorage) formatBytes(bytes float64) string {
	const unit = 1024.0
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB"}

	if bytes < unit {
		return fmt.Sprintf("%.2f B", bytes)
	}

	exp := 0
	for n := bytes / unit; n >= unit; n /= unit {
		exp++
	}

	return fmt.Sprintf("%.2f %s", bytes/math.Pow(unit, float64(exp)), sizes[exp])
}
