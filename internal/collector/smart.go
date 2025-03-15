package collector

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// SMARTCollector 实现SMART数据收集
type SMARTCollector struct {
	config        *model.Config
	logger        system.Logger
	commandRunner system.CommandRunner
}

// NewSMARTCollector 创建一个新的SMART数据收集器
func NewSMARTCollector(config *model.Config, logger system.Logger, runner system.CommandRunner) *SMARTCollector {
	return &SMARTCollector{
		config:        config,
		logger:        logger,
		commandRunner: runner,
	}
}

// GetSMARTData 获取磁盘的SMART数据
func (s *SMARTCollector) GetSMARTData(ctx context.Context, diskName, diskType string, diskModel string) (map[string]string, error) {
	// 根据磁盘类型选择不同的处理方法
	diskClassification := model.ClassifyDiskType(diskName, diskType, diskModel)

	switch diskClassification {
	case model.DiskTypeNVMESSD:
		return s.getNVMeSmartData(ctx, diskName)
	case model.DiskTypeVirtual:
		// 虚拟设备没有真正的SMART数据
		return map[string]string{
			"Type":         "虚拟设备",
			"Smart_Status": "虚拟设备",
		}, nil
	default:
		// SAS/SATA磁盘处理
		return s.getSATASmartData(ctx, diskName, string(diskClassification))
	}
}

// getNVMeSmartData 获取NVMe磁盘的SMART数据
func (s *SMARTCollector) getNVMeSmartData(ctx context.Context, diskName string) (map[string]string, error) {
	smartData := make(map[string]string)

	// 检查是否为VMware虚拟设备
	vendorCheck, _ := s.commandRunner.Run(ctx, fmt.Sprintf("smartctl -i /dev/%s | grep 'PCI Vendor'", diskName))
	if strings.Contains(vendorCheck, "0x15ad") {
		s.logger.Debug("%s是VMware虚拟设备，跳过详细SMART数据收集", diskName)
		smartData["Smart_Status"] = "虚拟设备"
		smartData["Type"] = "虚拟NVMe设备"
		return smartData, nil
	}

	// 获取健康状态
	healthOutput, _ := s.commandRunner.Run(ctx, fmt.Sprintf("smartctl -H /dev/%s", diskName))
	if healthOutput != "" {
		smartStatus := "未知"
		if strings.Contains(healthOutput, "PASSED") {
			smartStatus = "PASSED"
		} else if strings.Contains(healthOutput, "OK") {
			smartStatus = "OK"
		} else if strings.Contains(healthOutput, "FAILED") {
			smartStatus = "FAILED"
		}
		smartData["Smart_Status"] = smartStatus
	}

	// 获取SMART详情
	output, err := s.commandRunner.Run(ctx, fmt.Sprintf("smartctl -a /dev/%s", diskName))
	if err != nil {
		return smartData, fmt.Errorf("获取NVMe SMART数据失败: %w", err)
	}

	// 提取温度
	tempMatch := regexp.MustCompile(`Temperature:\s+(\d+)\s+Celsius`).FindStringSubmatch(output)
	if len(tempMatch) > 1 {
		temp, _ := strconv.Atoi(tempMatch[1])
		// 检查是否可能是开氏度(>200通常是开氏度)
		if temp > 200 {
			tempC := int(float64(temp) - 273.15)
			smartData["Temperature"] = strconv.Itoa(tempC)
		} else {
			smartData["Temperature"] = tempMatch[1]
		}
	}

	// 提取警告温度和临界温度
	warningTempMatch := regexp.MustCompile(`Warning\s+Comp\.\s+Temp\.\s+Threshold:\s+(\d+)\s+Celsius`).FindStringSubmatch(output)
	if len(warningTempMatch) > 1 {
		smartData["Warning_Temperature"] = warningTempMatch[1]
	}

	criticalTempMatch := regexp.MustCompile(`Critical\s+Comp\.\s+Temp\.\s+Threshold:\s+(\d+)\s+Celsius`).FindStringSubmatch(output)
	if len(criticalTempMatch) > 1 {
		smartData["Critical_Temperature"] = criticalTempMatch[1]
	}

	// 提取通电时间
	hoursPatterns := []string{
		`Power On Hours:\s+(\d+[,\d]*)`,
		`Power_On_Hours.*?(\d+)`,
		`Accumulated power on time.*?(\d+)[:\s]`,
		`number of hours powered up\s*=\s*(\d+\.?\d*)`,
	}

	for _, pattern := range hoursPatterns {
		hoursMatch := regexp.MustCompile(pattern).FindStringSubmatch(output)
		if len(hoursMatch) > 1 {
			smartData["Power_On_Hours"] = strings.ReplaceAll(hoursMatch[1], ",", "")
			break
		}
	}

	// 提取其他关键指标
	patterns := map[string]string{
		"Available_Spare": `Available Spare:\s+(\d+)%`,
		"Percentage_Used": `Percentage Used:\s+(\d+)%`,
		"Power_Cycles":    `Power Cycles:\s+(\d+[,\d]*)`,
	}

	for key, pattern := range patterns {
		match := regexp.MustCompile(pattern).FindStringSubmatch(output)
		if len(match) > 1 {
			value := strings.ReplaceAll(match[1], ",", "")
			smartData[key] = value
		}
	}

	// 提取数据读写量
	readMatch := regexp.MustCompile(`Data Units Read:\s+(\d+[,\d]*)\s+\[([^\]]+)\]`).FindStringSubmatch(output)
	if len(readMatch) > 2 {
		sizeRead := strings.TrimSpace(readMatch[2])
		smartData["Data_Read"] = s.normalizeSize(sizeRead)
	}

	writeMatch := regexp.MustCompile(`Data Units Written:\s+(\d+[,\d]*)\s+\[([^\]]+)\]`).FindStringSubmatch(output)
	if len(writeMatch) > 2 {
		sizeWritten := strings.TrimSpace(writeMatch[2])
		smartData["Data_Written"] = s.normalizeSize(sizeWritten)
	}

	// 提取 Uncorrected_Errors
	smartData["Uncorrected_Errors"] = "0" // 默认值
	uncorrectedErrorsMatch := regexp.MustCompile(`Media and Data Integrity Errors:\s+(\d+)`).FindStringSubmatch(output)
	if len(uncorrectedErrorsMatch) > 1 {
		smartData["Uncorrected_Errors"] = uncorrectedErrorsMatch[1]
	}

	return smartData, nil
}

// getSATASmartData 获取SATA/SAS磁盘的SMART数据
func (s *SMARTCollector) getSATASmartData(ctx context.Context, diskName, diskType string) (map[string]string, error) {
	smartData := make(map[string]string)

	// 确定是SSD还是HDD
	isSSD := diskType == string(model.DiskTypeSASSSD)

	// 获取健康状态
	healthOutput, _ := s.commandRunner.Run(ctx, fmt.Sprintf("smartctl -H /dev/%s", diskName))
	if healthOutput != "" {
		smartStatus := "未知"
		if strings.Contains(healthOutput, "PASSED") {
			smartStatus = "PASSED"
		} else if strings.Contains(healthOutput, "OK") {
			smartStatus = "OK"
		} else if strings.Contains(healthOutput, "FAILED") {
			smartStatus = "FAILED"
		}
		smartData["Smart_Status"] = smartStatus
	}

	// 检查是否存在"Percentage used endurance indicator"（仅适用于SSD）
	if isSSD && strings.Contains(healthOutput, "Percentage used endurance indicator:") {
		match := regexp.MustCompile(`Percentage used endurance indicator:\s+(\d+)%`).FindStringSubmatch(healthOutput)
		if len(match) > 1 {
			smartData["Percentage_Used"] = match[1]
		}
	}

	// 获取SMART详情
	output, err := s.commandRunner.Run(ctx, fmt.Sprintf("smartctl -a /dev/%s", diskName))
	if err != nil {
		return smartData, fmt.Errorf("获取SATA/SAS SMART数据失败: %w", err)
	}

	// 提取温度 - 尝试多种模式
	tempPatterns := []string{
		`Current Drive Temperature:\s+(\d+)\s+C`,
		`Temperature:\s+(\d+)\s+Celsius`,
		`Temperature_Celsius.*?(\d+)`,
		`Temperature.*?(\d+)`,
	}

	for _, pattern := range tempPatterns {
		match := regexp.MustCompile(pattern).FindStringSubmatch(output)
		if len(match) > 1 {
			smartData["Temperature"] = match[1]
			break
		}
	}

	// 提取警告温度
	tripTempPatterns := []string{
		`Drive Trip Temperature:\s+(\d+)\s+C`,
		`Warning\s+Comp\.\s+Temp\.\s+Threshold:\s+(\d+)`,
	}

	for _, pattern := range tripTempPatterns {
		match := regexp.MustCompile(pattern).FindStringSubmatch(output)
		if len(match) > 1 {
			smartData["Trip_Temperature"] = match[1]
			break
		}
	}

	// 提取通电时间
	hoursPatterns := []string{
		`number of hours powered up\s*[=:]?\s*(\d+\.\d+)`,        // SAS HDD
		`Accumulated power on time, hours:minutes\s+(\d+):[\d]+`, // SAS SSD
		`Power On Hours:\s+(\d+[,\d]*)`,
		`Power_On_Hours.*?(\d+)`,
		`Accumulated power on time.*?(\d+)[:\s]`,
		`power on time.*?(\d+)\s+hours`,
		`number of hours powered up\s*=\s*(\d+\.?\d*)`,
		`Accumulated power on time, hours:minutes (\d+):`,
	}

	for _, pattern := range hoursPatterns {
		match := regexp.MustCompile(pattern).FindStringSubmatch(output)
		if len(match) > 1 {
			smartData["Power_On_Hours"] = strings.ReplaceAll(match[1], ",", "")
			break
		}
	}

	// 提取通电周期
	cyclesPatterns := []string{
		`Accumulated start-stop cycles:\s+(\d+)`,
		`Power Cycles:\s+(\d+[,\d]*)`,
		`Power_Cycle_Count.*?(\d+)`,
		`start-stop cycles:\s+(\d+)`,
		`Power Cycle Count:\s+(\d+)`,
		`Specified cycle count over device lifetime:\s+(\d+)`,
	}

	for _, pattern := range cyclesPatterns {
		match := regexp.MustCompile(pattern).FindStringSubmatch(output)
		if len(match) > 1 {
			smartData["Power_Cycles"] = strings.ReplaceAll(match[1], ",", "")
			break
		}
	}

	// 提取非介质错误数量
	nonMediumMatch := regexp.MustCompile(`Non-medium error count:\s+(\d+)`).FindStringSubmatch(output)
	if len(nonMediumMatch) > 1 {
		smartData["Non_Medium_Errors"] = nonMediumMatch[1]
	}

	// 提取 Data_Read 和 Data_Written
	errorLogPattern := regexp.MustCompile(`(?s)Error counter log:.*?(read:.*?write:.*?)(\n\n|\z)`)
	errorLogSection := errorLogPattern.FindStringSubmatch(output)
	if len(errorLogSection) > 1 {
		errorLogText := errorLogSection[1]

		// 提取 read 的 Gigabytes processed
		readMatch := regexp.MustCompile(`read:.*?(\d+\.\d+)\s+`).FindStringSubmatch(errorLogText)
		if len(readMatch) > 1 {
			value, _ := strconv.ParseFloat(readMatch[1], 64)
			sizeStr := fmt.Sprintf("%.2f GB", value)
			smartData["Data_Read"] = s.normalizeSize(sizeStr)
		}

		// 提取 write 的 Gigabytes processed
		writeMatch := regexp.MustCompile(`write:.*?(\d+\.\d+)\s+`).FindStringSubmatch(errorLogText)
		if len(writeMatch) > 1 {
			value, _ := strconv.ParseFloat(writeMatch[1], 64)
			sizeStr := fmt.Sprintf("%.2f GB", value)
			smartData["Data_Written"] = s.normalizeSize(sizeStr)
		}
	}

	// 提取 Uncorrected_Errors
	smartData["Uncorrected_Errors"] = "0" // 默认值
	uncorrectedErrorsMatch := regexp.MustCompile(`Total uncorrected errors:\s+(\d+)`).FindStringSubmatch(output)
	if len(uncorrectedErrorsMatch) > 1 {
		smartData["Uncorrected_Errors"] = uncorrectedErrorsMatch[1]
	}

	return smartData, nil
}

// normalizeSize 将大小字符串标准化为合适的单位
func (s *SMARTCollector) normalizeSize(sizeStr string) string {
	// 添加调试日志记录
	s.logger.Debug("开始处理大小字符串: %s", sizeStr)

	if sizeStr == "" {
		s.logger.Debug("输入字符串为空")
		return ""
	}

	// 先处理科学计数法的容量
	if match, _ := regexp.MatchString(`^\d+\.\d+e[+-]\d+$`, sizeStr); match {
		value, err := strconv.ParseFloat(sizeStr, 64)
		if err == nil {
			// 直接转换为 TB
			tbValue := value / (1024 * 1024 * 1024 * 1024)
			result := fmt.Sprintf("%.2f TB", tbValue)
			s.logger.Debug("科学计数法直接转换: %s", result)
			return result
		}
	}

	match := regexp.MustCompile(`(\d+\.?\d*)\s*([KMGTP]?B)`).FindStringSubmatch(sizeStr)

	// 记录正则匹配结果
	s.logger.Debug("正则匹配结果: %v", match)

	if len(match) > 2 {
		value, _ := strconv.ParseFloat(match[1], 64)
		unit := strings.ToUpper(match[2])

		// 记录解析的数值和单位
		s.logger.Debug("解析的数值: %f, 单位: %s", value, unit)

		// 转换为字节数
		var bytes float64
		switch unit {
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
			s.logger.Debug("未知单位: %s", unit)
			return sizeStr
		}

		// 记录转换后的字节数
		s.logger.Debug("转换后的字节数: %f", bytes)

		result := s.formatSize(bytes)

		// 记录最终结果
		s.logger.Debug("最终转换结果: %s", result)

		return result
	}

	// 如果正则匹配失败，尝试处理科学计数法
	if value, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		s.logger.Debug("尝试处理科学计数法: %f", value)
		result := s.formatSize(value)
		s.logger.Debug("科学计数法转换结果: %s", result)
		return result
	}

	s.logger.Debug("无法处理的大小字符串: %s", sizeStr)
	return sizeStr
}

// formatSize 格式化容量大小
func (s *SMARTCollector) formatSize(sizeBytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	unitIndex := 0
	size := sizeBytes

	// 处理负数情况
	sign := ""
	if size < 0 {
		sign = "-"
		size = math.Abs(size)
	}

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	// 根据单位和大小选择最佳显示方式
	switch units[unitIndex] {
	case "B", "KB":
		return fmt.Sprintf("%s%.2f %s", sign, size, units[unitIndex])
	case "MB", "GB":
		return fmt.Sprintf("%s%.2f %s", sign, size, units[unitIndex])
	case "TB", "PB":
		return fmt.Sprintf("%s%.2f %s", sign, size, units[unitIndex])
	default:
		return fmt.Sprintf("%s%.2f B", sign, size)
	}
}
