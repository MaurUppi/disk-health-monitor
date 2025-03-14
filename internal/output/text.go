// output/text.go
package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/olekukonko/tablewriter"
)

// Default option values
const (
	DefaultBorderStyle = BorderStyleClassic
	DefaultMaxWidth    = 120
	DefaultCompactMode = false
	DefaultColorOutput = true
)

// TextFormatter implements the OutputFormatter interface for text output
type TextFormatter struct {
	BaseFormatter
	buffer      *strings.Builder
	tableBuffer bytes.Buffer
}

// createTextFormatter creates a new instance of TextFormatter (internal use only)
func createTextFormatter(options map[string]interface{}) *TextFormatter {
	tf := &TextFormatter{
		BaseFormatter: NewBaseFormatter(),
		buffer:        &strings.Builder{},
	}

	// Set default options
	tf.SetOption(OptionBorderStyle, DefaultBorderStyle)
	tf.SetOption(OptionMaxWidth, DefaultMaxWidth)
	tf.SetOption(OptionCompactMode, DefaultCompactMode)
	tf.SetOption(OptionColorOutput, DefaultColorOutput)
	tf.SetOption(OptionGroupByType, true)
	tf.SetOption(OptionIncludeSummary, true)
	tf.SetOption(OptionIncludeTimestamp, true)

	// Override with provided options
	for name, value := range options {
		tf.SetOption(name, value)
	}

	return tf
}

// GetSupportedOptions returns a map of supported options and their descriptions
func (tf *TextFormatter) GetSupportedOptions() map[string]string {
	return map[string]string{
		OptionBorderStyle:    "Table border style (classic, simple, none)",
		OptionMaxWidth:       "Maximum width for tables (0 for no limit)",
		OptionCompactMode:    "Use compact mode with fewer columns",
		OptionColorOutput:    "Use ANSI color codes for terminal output",
		OptionGroupByType:    "Group disks by type",
		OptionIncludeSummary: "Include summary information",
		OptionIncludeTimestamp: "Include timestamp",
	}
}

// FormatDiskInfo formats disk information into text
func (tf *TextFormatter) FormatDiskInfo(diskData *model.DiskData) error {
	if diskData == nil {
		return fmt.Errorf("no disk data to format")
	}

	// Save disk data
	tf.diskData = diskData

	// Reset buffer
	tf.buffer.Reset()

	// Add title
	tf.writeTitle("TrueNAS磁盘健康监控")

	// Add timestamp if enabled
	if tf.GetBoolOption(OptionIncludeTimestamp, true) {
		tf.buffer.WriteString(fmt.Sprintf("生成时间: %s\n\n", tf.FormatTimestamp()))
	}

	// Add summary if enabled
	if tf.GetBoolOption(OptionIncludeSummary, true) {
		tf.writeSummary()
	}

	// Check if disks should be grouped
	if tf.GetBoolOption(OptionGroupByType, true) {
		// Write each disk type section
		if count := diskData.GetDiskCountByType(model.DiskTypeSASSSD); count > 0 {
			tf.writeDiskGroup(model.DiskTypeSASSSD)
		}
		if count := diskData.GetDiskCountByType(model.DiskTypeSASHDD); count > 0 {
			tf.writeDiskGroup(model.DiskTypeSASHDD)
		}
		if count := diskData.GetDiskCountByType(model.DiskTypeNVMESSD); count > 0 {
			tf.writeDiskGroup(model.DiskTypeNVMESSD)
		}
		if count := diskData.GetDiskCountByType(model.DiskTypeVirtual); count > 0 {
			tf.writeDiskGroup(model.DiskTypeVirtual)
		}
	} else {
		// Write all disks together
		tf.writeAllDisks()
	}

	// Add read/write increment information if available
	if diskData.HasPreviousData() {
		tf.writeIncrementTable()
	}

	return nil
}

// FormatControllerInfo formats controller information into text
func (tf *TextFormatter) FormatControllerInfo(controllerData *model.ControllerData) error {
	if controllerData == nil {
		return fmt.Errorf("no controller data to format")
	}

	// Save controller data
	tf.controllerData = controllerData

	// If this is called directly (without disk data), create a buffer
	if tf.buffer.Len() == 0 {
		// Add title
		tf.writeTitle("TrueNAS控制器信息")

		// Add timestamp if enabled
		if tf.GetBoolOption(OptionIncludeTimestamp, true) {
			tf.buffer.WriteString(fmt.Sprintf("生成时间: %s\n\n", tf.FormatTimestamp()))
		}
	}

	// Add LSI controller section if any
	if len(controllerData.LSIControllers) > 0 {
		tf.writeLSIControllers()
	}

	// Add NVMe controller section if any
	if len(controllerData.NVMeControllers) > 0 {
		tf.writeNVMeControllers()
	}

	return nil
}

// SaveToFile saves the formatted output to a file
func (tf *TextFormatter) SaveToFile(filename string) error {
	// Ensure directory exists
	if err := tf.EnsureDirectoryExists(filename); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	err := os.WriteFile(filename, []byte(tf.buffer.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// String returns the formatted output as a string
func (tf *TextFormatter) String() string {
	return tf.buffer.String()
}

// WriteToWriter writes the formatted output to a writer
func (tf *TextFormatter) WriteToWriter(w io.Writer) error {
	_, err := w.Write([]byte(tf.buffer.String()))
	return err
}

// writeTitle writes a title to the buffer
func (tf *TextFormatter) writeTitle(title string) {
	tf.buffer.WriteString("=== " + title + " ===\n\n")
}

// writeSectionTitle writes a section title to the buffer
func (tf *TextFormatter) writeSectionTitle(title string) {
	tf.buffer.WriteString("--- " + title + " ---\n\n")
}

// writeSummary writes the summary section
func (tf *TextFormatter) writeSummary() {
	summary := tf.GetSummaryInfo()
	
	tf.buffer.WriteString("系统摘要:\n")
	tf.buffer.WriteString(fmt.Sprintf("- 总磁盘数: %s", summary["TotalDisks"]))
	
	// Add SSD and HDD counts if available
	if ssdCount, ok := summary["SSDCount"]; ok {
		tf.buffer.WriteString(fmt.Sprintf(" (SSD: %s", ssdCount))
		if hddCount, ok := summary["HDDCount"]; ok {
			tf.buffer.WriteString(fmt.Sprintf(", HDD: %s", hddCount))
		}
		tf.buffer.WriteString(")")
	}
	tf.buffer.WriteString("\n")
	
	// Add warning and error counts
	warningCount := summary["WarningCount"]
	if warningCount != "0" {
		tf.buffer.WriteString(fmt.Sprintf("- 警告数: %s\n", colorizeText(warningCount, "yellow")))
	} else {
		tf.buffer.WriteString(fmt.Sprintf("- 警告数: %s\n", warningCount))
	}
	
	errorCount := summary["ErrorCount"]
	if errorCount != "0" {
		tf.buffer.WriteString(fmt.Sprintf("- 错误数: %s\n", colorizeText(errorCount, "red")))
	} else {
		tf.buffer.WriteString(fmt.Sprintf("- 错误数: %s\n", errorCount))
	}
	
	// Add controller count if available
	if controllerCount, ok := summary["ControllerCount"]; ok {
		tf.buffer.WriteString(fmt.Sprintf("- 控制器数: %s\n", controllerCount))
	}
	
	tf.buffer.WriteString("\n")
}

// writeDiskGroup writes a group of disks of the same type
func (tf *TextFormatter) writeDiskGroup(diskType model.DiskType) {
	// Get disks of this type
	disks, ok := tf.diskData.GroupedDisks[diskType]
	if !ok || len(disks) == 0 {
		return
	}

	// Determine group title
	var title string
	switch diskType {
	case model.DiskTypeSASSSD:
		title = "SAS/SATA 固态硬盘"
	case model.DiskTypeSASHDD:
		title = "SAS/SATA 机械硬盘"
	case model.DiskTypeNVMESSD:
		title = "NVMe 固态硬盘"
	case model.DiskTypeVirtual:
		title = "虚拟设备"
	default:
		title = "其他设备"
	}

	// Write section title
	tf.writeSectionTitle(title)

	// Create a table
	tf.writeTableForDiskType(diskType, disks)
}

// writeAllDisks writes all disks in a single table
func (tf *TextFormatter) writeAllDisks() {
	if len(tf.diskData.Disks) == 0 {
		return
	}

	tf.writeSectionTitle("所有磁盘")

	// Create a table with all necessary columns
	table := tf.createTable()
	
	// Set header
	if tf.GetBoolOption(OptionCompactMode, false) {
		table.SetHeader([]string{"名称", "类型", "容量", "存储池", "温度", "通电时间", "状态"})
	} else {
		table.SetHeader([]string{"名称", "型号", "类型", "容量", "存储池", "温度", "通电时间", "状态", "已读数据", "已写数据"})
	}

	// Add rows for all disks
	for _, disk := range tf.diskData.Disks {
		var row []string
		
		if tf.GetBoolOption(OptionCompactMode, false) {
			// Compact mode with fewer columns
			row = []string{
				disk.Name,
				string(disk.Type),
				disk.Size,
				disk.Pool,
				disk.GetDisplayTemperature(),
				FormatPowerOnHours(disk.GetAttribute("Power_On_Hours")),
				colorizeSMARTStatus(FormatSMARTStatus(disk.GetAttribute("Smart_Status")), tf.GetBoolOption(OptionColorOutput, true)),
			}
		} else {
			// Full mode with all columns
			row = []string{
				disk.Name,
				disk.Model,
				string(disk.Type),
				disk.Size,
				disk.Pool,
				disk.GetDisplayTemperature(),
				FormatPowerOnHours(disk.GetAttribute("Power_On_Hours")),
				colorizeSMARTStatus(FormatSMARTStatus(disk.GetAttribute("Smart_Status")), tf.GetBoolOption(OptionColorOutput, true)),
				disk.GetAttribute("Data_Read"),
				disk.GetAttribute("Data_Written"),
			}
		}
		
		table.Append(row)
	}

	// Render the table
	tf.renderTable(table)
}

// writeTableForDiskType writes a table for disks of a specific type
func (tf *TextFormatter) writeTableForDiskType(diskType model.DiskType, disks []*model.Disk) {
	if len(disks) == 0 {
		return
	}

	// Create a table
	table := tf.createTable()
	
	// Get attributes for this disk type
	attributes := tf.diskData.GetDiskAttributes(diskType)
	
	// Determine columns based on disk type and mode
	var headers []string
	
	// Base columns for all disk types
	if tf.GetBoolOption(OptionCompactMode, false) {
		// Compact mode
		headers = []string{"名称", "容量", "存储池"}
	} else {
		// Full mode
		headers = []string{"名称", "型号", "容量", "存储池"}
	}
	
	// Add attribute columns based on disk type
	for _, attr := range attributes {
		// Skip some columns in compact mode
		if tf.GetBoolOption(OptionCompactMode, false) {
			if attr.Name != "Temperature" && 
			   attr.Name != "Smart_Status" && 
			   attr.Name != "Power_On_Hours" {
				continue
			}
		}
		
		headers = append(headers, attr.DisplayName)
	}
	
	// Add increment columns if available
	if tf.diskData.HasPreviousData() && !tf.GetBoolOption(OptionCompactMode, false) {
		headers = append(headers, "读增量", "写增量")
	}
	
	table.SetHeader(headers)
	
	// Add rows for each disk
	for _, disk := range disks {
		var row []string
		
		// Add base columns
		if tf.GetBoolOption(OptionCompactMode, false) {
			row = []string{disk.Name, disk.Size, disk.Pool}
		} else {
			row = []string{disk.Name, disk.Model, disk.Size, disk.Pool}
		}
		
		// Add attribute values
		for _, attr := range attributes {
			// Skip some columns in compact mode
			if tf.GetBoolOption(OptionCompactMode, false) {
				if attr.Name != "Temperature" && 
				   attr.Name != "Smart_Status" && 
				   attr.Name != "Power_On_Hours" {
					continue
				}
			}
			
			value := disk.GetAttribute(attr.Name)
			
			// Format special values
			switch attr.Name {
			case "Temperature":
				value = disk.GetDisplayTemperature()
			case "Power_On_Hours":
				value = FormatPowerOnHours(value)
			case "Smart_Status":
				value = colorizeSMARTStatus(FormatSMARTStatus(value), tf.GetBoolOption(OptionColorOutput, true))
			}
			
			row = append(row, value)
		}
		
		// Add increment values if available
		if tf.diskData.HasPreviousData() && !tf.GetBoolOption(OptionCompactMode, false) {
			row = append(row, disk.ReadIncrement, disk.WriteIncrement)
		}
		
		table.Append(row)
	}
	
	// Render the table
	tf.renderTable(table)
}

// writeIncrementTable writes a table showing read/write increments
func (tf *TextFormatter) writeIncrementTable() {
	if !tf.diskData.HasPreviousData() {
		return
	}

	tf.writeSectionTitle(fmt.Sprintf("磁盘读写增量信息 (自 %s)", tf.diskData.PreviousTime))

	// Create a table
	table := tf.createTable()
	
	// Set header
	table.SetHeader([]string{"磁盘名称", "类型", "型号", "存储池", "当前读取总量", "读取增量", "当前写入总量", "写入增量"})
	
	// Add rows for disks with increment data
	for _, disk := range tf.diskData.Disks {
		// Skip disks without increment data
		if disk.ReadIncrement == "" && disk.WriteIncrement == "" {
			continue
		}
		
		// Create row
		row := []string{
			disk.Name,
			string(disk.Type),
			disk.Model,
			disk.Pool,
			disk.GetAttribute("Data_Read"),
			disk.ReadIncrement,
			disk.GetAttribute("Data_Written"),
			disk.WriteIncrement,
		}
		
		table.Append(row)
	}
	
	// Render the table
	tf.renderTable(table)
}

// writeLSIControllers writes LSI controller information
func (tf *TextFormatter) writeLSIControllers() {
	if len(tf.controllerData.LSIControllers) == 0 {
		return
	}

	tf.writeSectionTitle("LSI SAS HBA控制器")

	// Create a table
	table := tf.createTable()
	
	// Set header
	if tf.GetBoolOption(OptionCompactMode, false) {
		table.SetHeader([]string{"控制器名称", "型号", "温度", "设备数", "状态"})
	} else {
		table.SetHeader([]string{"控制器名称", "型号", "固件版本", "驱动版本", "温度", "设备数", "状态"})
	}
	
	// Add rows for each controller
	for id, controller := range tf.controllerData.LSIControllers {
		var row []string
		
		if tf.GetBoolOption(OptionCompactMode, false) {
			row = []string{
				id,
				controller.Model,
				controller.GetDisplayTemperature(),
				controller.DeviceCount,
				colorizeControllerStatus(string(controller.Status), tf.GetBoolOption(OptionColorOutput, true)),
			}
		} else {
			row = []string{
				id,
				controller.Model,
				controller.FirmwareVersion,
				controller.DriverVersion,
				controller.GetDisplayTemperature(),
				controller.DeviceCount,
				colorizeControllerStatus(string(controller.Status), tf.GetBoolOption(OptionColorOutput, true)),
			}
		}
		
		table.Append(row)
	}
	
	// Render the table
	tf.renderTable(table)
}

// writeNVMeControllers writes NVMe controller information
func (tf *TextFormatter) writeNVMeControllers() {
	if len(tf.controllerData.NVMeControllers) == 0 {
		return
	}

	tf.writeSectionTitle("NVMe控制器")

	// Create a table
	table := tf.createTable()
	
	// Set header
	table.SetHeader([]string{"总线ID", "控制器描述", "温度"})
	
	// Add rows for each controller
	for _, controller := range tf.controllerData.NVMeControllers {
		row := []string{
			controller.Bus,
			controller.Description,
			controller.GetDisplayTemperature(),
		}
		
		table.Append(row)
	}
	
	// Render the table
	tf.renderTable(table)
}

// createTable creates a table with common settings
func (tf *TextFormatter) createTable() *tablewriter.Table {
	// Reset table buffer
	tf.tableBuffer.Reset()
	
	// Create table writer
	table := tablewriter.NewWriter(&tf.tableBuffer)
	
	// Set border style
	switch tf.GetStringOption(OptionBorderStyle, DefaultBorderStyle) {
	case BorderStyleClassic:
		// Default style
	case BorderStyleSimple:
		table.SetBorders(tablewriter.Border{Left: false, Top: true, Right: false, Bottom: true})
		table.SetCenterSeparator("|")
		table.SetColumnSeparator("|")
		table.SetRowSeparator("-")
	case BorderStyleNone:
		table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
		table.SetCenterSeparator("")
		table.SetColumnSeparator("  ")
		table.SetRowSeparator("")
	}
	
	// Common settings
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	
	// Set max width if specified
	// The tablewriter library doesn't directly support max width,
	// so we'd need to calculate column widths manually if needed
	// For now, this is handled through the table settings already applied
	
	return table
}

// renderTable renders a table to the main buffer
func (tf *TextFormatter) renderTable(table *tablewriter.Table) {
	table.Render()
	tf.buffer.WriteString(tf.tableBuffer.String())
	tf.buffer.WriteString("\n")
}

// colorizeText applies ANSI color to text if color output is enabled
func colorizeText(text string, color string) string {
	// ANSI color codes
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorGreen  = "\033[32m"
		colorYellow = "\033[33m"
	)
	
	// Return plain text if empty
	if text == "" {
		return ""
	}
	
	// Apply color based on requested color
	switch color {
	case "red":
		return colorRed + text + colorReset
	case "green":
		return colorGreen + text + colorReset
	case "yellow":
		return colorYellow + text + colorReset
	default:
		return text
	}
}

// colorizeSMARTStatus applies appropriate color to SMART status
func colorizeSMARTStatus(status string, useColor bool) string {
	if !useColor {
		return status
	}
	
	switch status {
	case "正常":
		return colorizeText(status, "green")
	case "警告":
		return colorizeText(status, "yellow")
	case "错误":
		return colorizeText(status, "red")
	default:
		return status
	}
}

// colorizeControllerStatus applies appropriate color to controller status
func colorizeControllerStatus(status string, useColor bool) string {
	if !useColor {
		return status
	}
	
	switch status {
	case "正常":
		return colorizeText(status, "green")
	case "警告":
		return colorizeText(status, "yellow")
	case "错误":
		return colorizeText(status, "red")
	default:
		return status
	}
}

// init registers the text formatter factory
func init() {
	// 将工厂函数赋值给formatter.go中声明的变量
	NewTextFormatter = func(options map[string]interface{}) OutputFormatter {
		formatter := &TextFormatter{
			BaseFormatter: NewBaseFormatter(),
			buffer:        &strings.Builder{},
		}
		
		// Set default options
		formatter.SetOption(OptionBorderStyle, DefaultBorderStyle)
		formatter.SetOption(OptionMaxWidth, DefaultMaxWidth)
		formatter.SetOption(OptionCompactMode, DefaultCompactMode)
		formatter.SetOption(OptionColorOutput, DefaultColorOutput)
		formatter.SetOption(OptionGroupByType, true)
		formatter.SetOption(OptionIncludeSummary, true)
		formatter.SetOption(OptionIncludeTimestamp, true)
		
		// Override with provided options
		if options != nil {
			for name, value := range options {
				formatter.SetOption(name, value)
			}
		}
		
		return formatter
	}
}
