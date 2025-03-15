// output/html.go
package output

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
)

// Default option values for HTML formatter
const (
	DefaultShowTemperatureBar  = true
	DefaultEnableInteractivity = true
	DefaultHtmlTitle           = "TrueNAS磁盘健康监控"
)

// HTMLFormatter implements the OutputFormatter interface for HTML output
type HTMLFormatter struct {
	BaseFormatter
	htmlBuffer strings.Builder
}

// createHTMLFormatter creates a new instance of HTMLFormatter (internal use only)
func createHTMLFormatter(options map[string]interface{}) *HTMLFormatter {
	hf := &HTMLFormatter{
		BaseFormatter: NewBaseFormatter(),
	}

	// Set default options
	hf.SetOption(OptionTemperatureBar, DefaultShowTemperatureBar)
	hf.SetOption(OptionEnableInteractivity, DefaultEnableInteractivity)
	hf.SetOption(OptionHtmlTitle, DefaultHtmlTitle)
	hf.SetOption(OptionGroupByType, true)
	hf.SetOption(OptionIncludeSummary, true)
	hf.SetOption(OptionIncludeTimestamp, true)

	// Override with provided options
	for name, value := range options {
		hf.SetOption(name, value)
	}

	return hf
}

// GetSupportedOptions returns a map of supported options and their descriptions
func (hf *HTMLFormatter) GetSupportedOptions() map[string]string {
	return map[string]string{
		OptionTemperatureBar:      "Show visual temperature indicators",
		OptionEnableInteractivity: "Enable interactive features (sorting, filtering)",
		OptionHtmlTitle:           "HTML page title",
		OptionGroupByType:         "Group disks by type",
		OptionIncludeSummary:      "Include summary information",
		OptionIncludeTimestamp:    "Include timestamp",
	}
}

// FormatDiskInfo formats disk information into HTML
func (hf *HTMLFormatter) FormatDiskInfo(diskData *model.DiskData) error {
	if diskData == nil {
		return fmt.Errorf("no disk data to format")
	}

	// Save disk data
	hf.diskData = diskData

	// Generate HTML
	return hf.generateHTML()
}

// FormatControllerInfo formats controller information into HTML
func (hf *HTMLFormatter) FormatControllerInfo(controllerData *model.ControllerData) error {
	if controllerData == nil {
		return fmt.Errorf("no controller data to format")
	}

	// Save controller data
	hf.controllerData = controllerData

	// Generate HTML (if disk data is available)
	if hf.diskData != nil {
		return hf.generateHTML()
	}

	// Create controller-only HTML if disk data is not available
	hf.htmlBuffer.Reset()
	controllerOnly := true
	return hf.generateControllerOnlyHTML(controllerOnly)
}

// SaveToFile saves the formatted output to a file
func (hf *HTMLFormatter) SaveToFile(filename string) error {
	// Ensure directory exists
	if err := hf.EnsureDirectoryExists(filename); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if buffer has content
	if hf.htmlBuffer.Len() == 0 {
		return fmt.Errorf("no content to save to file")
	}

	// Write to file
	content := hf.htmlBuffer.String()
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// generateHTML generates the complete HTML document
func (hf *HTMLFormatter) generateHTML() error {
	hf.htmlBuffer.Reset()

	// 转换 GroupedDisks 的键为字符串
	groupedDisksStr := make(map[string][]*model.Disk)
	if hf.diskData != nil {
		for diskType, disks := range hf.diskData.GroupedDisks {
			groupedDisksStr[string(diskType)] = disks
		}
	}

	// Define the data to pass to the template
	data := map[string]interface{}{
		"Title":               hf.GetStringOption(OptionHtmlTitle, DefaultHtmlTitle),
		"Timestamp":           hf.FormatTimestamp(),
		"DiskData":            hf.diskData,
		"ControllerData":      hf.controllerData,
		"ShowTemperatureBar":  hf.GetBoolOption(OptionTemperatureBar, DefaultShowTemperatureBar),
		"EnableInteractivity": hf.GetBoolOption(OptionEnableInteractivity, DefaultEnableInteractivity),
		"HasIncrement":        hf.diskData != nil && hf.diskData.HasPreviousData(),
		"PreviousTime": func() string {
			if hf.diskData != nil {
				return hf.diskData.PreviousTime
			} else {
				return ""
			}
		}(),
		"SummaryInfo":     hf.GetSummaryInfo(),
		"GroupedDisksStr": groupedDisksStr, // 新增传入转换后的 groupedDisks
	}

	// Create a new template and parse the HTML template string
	t, err := template.New("diskHealthReport").Funcs(template.FuncMap{
		"getStatusClass":       GetStatusClass,
		"formatTemperatureBar": hf.formatTemperatureBar,
		"formatPowerOnHours":   FormatPowerOnHours,
		"formatSize":           FormatSciNotation,
		"string": func(v interface{}) string {
			return fmt.Sprintf("%v", v)
		},
	}).Parse(htmlTemplate)

	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Execute the template
	if err := t.Execute(&hf.htmlBuffer, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// generateControllerOnlyHTML generates an HTML document with only controller information
func (hf *HTMLFormatter) generateControllerOnlyHTML(controllerOnly bool) error {
	hf.htmlBuffer.Reset()

	// Define the data to pass to the template
	data := map[string]interface{}{
		"Title":               hf.GetStringOption(OptionHtmlTitle, DefaultHtmlTitle) + " - 控制器信息",
		"Timestamp":           hf.FormatTimestamp(),
		"ControllerData":      hf.controllerData,
		"ControllerOnly":      controllerOnly,
		"ShowTemperatureBar":  hf.GetBoolOption(OptionTemperatureBar, DefaultShowTemperatureBar),
		"EnableInteractivity": hf.GetBoolOption(OptionEnableInteractivity, DefaultEnableInteractivity),
	}

	// Create a new template and parse the controller-only HTML template string
	t, err := template.New("controllerReport").Funcs(template.FuncMap{
		"getStatusClass": GetStatusClass,
		"string": func(v interface{}) string {
			return fmt.Sprintf("%v", v)
		},
	}).Parse(controllerOnlyTemplate)

	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Execute the template
	if err := t.Execute(&hf.htmlBuffer, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// formatTemperatureBar formats a visual temperature bar
func (hf *HTMLFormatter) formatTemperatureBar(temp string) string {
	// Parse the temperature
	var tempValue int
	_, err := fmt.Sscanf(temp, "%d", &tempValue)
	if err != nil {
		return "" // If we can't parse the temperature, don't generate a bar
	}

	// Calculate position (percentage) based on temperature
	// Assuming normal range is 20-60C, with 20C at 0% and 60C at 100%
	position := 0
	if tempValue >= 20 && tempValue <= 60 {
		position = (tempValue - 20) * 100 / 40 // Convert to percentage within our range
	} else if tempValue < 20 {
		position = 0
	} else {
		position = 100
	}

	// Return HTML for temperature bar
	return fmt.Sprintf(`<div class="temperature">
            <div class="temperature-marker" style="left: %d%%;"></div>
        </div>`, position)
}

// HTML templates

// The main HTML template
const htmlTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            margin: 0;
            padding: 20px;
            color: #333;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .panel {
            background-color: white;
            border-radius: 5px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24);
            margin-bottom: 20px;
            overflow: hidden;
        }
        .panel-header {
            padding: 15px 20px;
            background-color: #0747a6;
            color: white;
            font-weight: 500;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .panel-body {
            padding: 0;
            overflow: auto;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th {
            background-color: #f4f5f7;
            text-align: left;
            padding: 10px;
            border-bottom: 1px solid #ddd;
            position: sticky;
            top: 0;
            cursor: pointer;
        }
        td {
            padding: 10px;
            border-bottom: 1px solid #eee;
            white-space: nowrap;
        }
        tr:hover {
            background-color: #f9f9f9;
        }
        .status-ok {
            color: #00875a;
            font-weight: bold;
        }
        .status-warning {
            color: #ff8b00;
            font-weight: bold;
        }
        .status-error {
            color: #de350b;
            font-weight: bold;
        }
        .temperature {
            position: relative;
            display: inline-block;
            width: 50px;
            height: 15px;
            background: linear-gradient(to right, #00b8d9, #ffab00, #ff5630);
            border-radius: 2px;
            margin-right: 10px;
        }
        .temperature-marker {
            position: absolute;
            top: -5px;
            width: 3px;
            height: 25px;
            background-color: #333;
        }
        .summary-tiles {
            display: flex;
            flex-wrap: wrap;
            margin: 0 -10px 20px -10px;
        }
        .summary-tile {
            flex: 1;
            min-width: 200px;
            margin: 10px;
            padding: 15px;
            background-color: white;
            border-radius: 5px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24);
        }
        .summary-tile h3 {
            margin: 0 0 10px 0;
            font-size: 14px;
            color: #5e6c84;
        }
        .summary-tile .value {
            font-size: 24px;
            font-weight: bold;
        }
        .search-box {
            padding: 10px 20px;
            background-color: white;
            border-bottom: 1px solid #eee;
        }
        .search-box input {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }
        .tab-container {
            margin-bottom: 20px;
        }
        .tabs {
            display: flex;
            list-style: none;
            padding: 0;
            margin: 0;
            background-color: white;
            border-radius: 5px 5px 0 0;
            overflow: hidden;
        }
        .tab {
            padding: 12px 24px;
            cursor: pointer;
            transition: background-color 0.3s;
        }
        .tab.active {
            background-color: #0747a6;
            color: white;
            font-weight: 500;
        }
        .tab:hover:not(.active) {
            background-color: #f4f5f7;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: block;
        }
        .last-update {
            font-size: 12px;
            color: #5e6c84;
            text-align: right;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        
        <div class="last-update">最后更新时间: {{.Timestamp}}</div>
        
        {{if .SummaryInfo}}
        <div class="summary-tiles">
            <div class="summary-tile">
                <h3>总磁盘数</h3>
                <div class="value">{{.SummaryInfo.TotalDisks}}</div>
            </div>
            <div class="summary-tile">
                <h3>SSD</h3>
                <div class="value">{{.SummaryInfo.SSDCount}}</div>
            </div>
            <div class="summary-tile">
                <h3>HDD</h3>
                <div class="value">{{.SummaryInfo.HDDCount}}</div>
            </div>
            <div class="summary-tile">
                <h3>警告数</h3>
                <div class="value {{if ne .SummaryInfo.WarningCount "0"}}status-warning{{end}}">{{.SummaryInfo.WarningCount}}</div>
            </div>
            <div class="summary-tile">
                <h3>错误数</h3>
                <div class="value {{if ne .SummaryInfo.ErrorCount "0"}}status-error{{end}}">{{.SummaryInfo.ErrorCount}}</div>
            </div>
        </div>
        {{end}}
        
        <div class="tab-container">
            <ul class="tabs">
                <li class="tab active" onclick="openTab(event, 'disk-tab')">磁盘</li>
                <li class="tab" onclick="openTab(event, 'controller-tab')">控制器</li>
                {{if .HasIncrement}}
                <li class="tab" onclick="openTab(event, 'history-tab')">历史数据</li>
                {{end}}
            </ul>
            
            <div id="disk-tab" class="tab-content active">
                <!-- SSD Section -->
                {{if index .GroupedDisksStr "SAS_SSD"}}
                <div class="panel">
                    <div class="panel-header">
                        <span>SAS/SATA 固态硬盘</span>
                    </div>
                    <div class="search-box">
                        <input type="text" placeholder="搜索磁盘..." oninput="filterTable('ssd-table', this.value)">
                    </div>
                    <div class="panel-body">
                        <table id="ssd-table">
                            <thead>
                                <tr>
                                    <th onclick="sortTable('ssd-table', 0)">磁盘名称</th>
                                    <th onclick="sortTable('ssd-table', 1)">型号</th>
                                    <th onclick="sortTable('ssd-table', 2)">容量</th>
                                    <th onclick="sortTable('ssd-table', 3)">存储池</th>
                                    <th onclick="sortTable('ssd-table', 4)">温度</th>
                                    <th onclick="sortTable('ssd-table', 5)">通电时间</th>
                                    <th onclick="sortTable('ssd-table', 6)">已用寿命</th>
                                    <th onclick="sortTable('ssd-table', 7)">SMART状态</th>
                                    <th onclick="sortTable('ssd-table', 8)">已读数据</th>
                                    <th onclick="sortTable('ssd-table', 9)">已写数据</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range index .GroupedDisksStr "SAS_SSD"}}
                                <tr>
                                    <td>{{.Name}}</td>
                                    <td>{{.Model}}</td>
                                    <td>{{formatSize .Size}}</td>
                                    <td>{{.Pool}}</td>
                                    <td>
                                        {{if $.ShowTemperatureBar}}
                                        {{formatTemperatureBar .GetDisplayTemperature}}
                                        {{end}}
                                        {{.GetDisplayTemperature}}
                                    </td>
                                    <td>{{formatPowerOnHours (.GetAttribute "Power_On_Hours")}}</td>
                                    <td>{{.GetAttribute "Percentage_Used"}}</td>
                                    <td class="{{getStatusClass (.GetAttribute "Smart_Status")}}">{{.GetAttribute "Smart_Status"}}</td>
                                    <td>{{.GetAttribute "Data_Read"}}</td>
                                    <td>{{.GetAttribute "Data_Written"}}</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
                
                <!-- HDD Section -->
                {{if index .GroupedDisksStr "SAS_HDD"}}
                <div class="panel">
                    <div class="panel-header">
                        <span>SAS/SATA 机械硬盘</span>
                    </div>
                    <div class="search-box">
                        <input type="text" placeholder="搜索磁盘..." oninput="filterTable('hdd-table', this.value)">
                    </div>
                    <div class="panel-body">
                        <table id="hdd-table">
                            <thead>
                                <tr>
                                    <th onclick="sortTable('hdd-table', 0)">磁盘名称</th>
                                    <th onclick="sortTable('hdd-table', 1)">型号</th>
                                    <th onclick="sortTable('hdd-table', 2)">容量</th>
                                    <th onclick="sortTable('hdd-table', 3)">存储池</th>
                                    <th onclick="sortTable('hdd-table', 4)">温度</th>
                                    <th onclick="sortTable('hdd-table', 5)">通电时间</th>
                                    <th onclick="sortTable('hdd-table', 6)">SMART状态</th>
                                    <th onclick="sortTable('hdd-table', 7)">已读数据</th>
                                    <th onclick="sortTable('hdd-table', 8)">已写数据</th>
                                    <th onclick="sortTable('hdd-table', 9)">未修正错误</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range index .GroupedDisksStr "SAS_HDD"}}
                                <tr>
                                    <td>{{.Name}}</td>
                                    <td>{{.Model}}</td>
                                    <td>{{formatSize .Size}}</td>
                                    <td>{{.Pool}}</td>
                                    <td>
                                        {{if $.ShowTemperatureBar}}
                                        {{formatTemperatureBar .GetDisplayTemperature}}
                                        {{end}}
                                        {{.GetDisplayTemperature}}
                                    </td>
                                    <td>{{formatPowerOnHours (.GetAttribute "Power_On_Hours")}}</td>
                                    <td class="{{getStatusClass (.GetAttribute "Smart_Status")}}">{{.GetAttribute "Smart_Status"}}</td>
                                    <td>{{.GetAttribute "Data_Read"}}</td>
                                    <td>{{.GetAttribute "Data_Written"}}</td>
                                    <td>{{.GetAttribute "Uncorrected_Errors"}}</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
                
                <!-- NVMe Section -->
                {{if index .GroupedDisksStr "NVME_SSD"}}
                <div class="panel">
                    <div class="panel-header">
                        <span>NVMe 固态硬盘</span>
                    </div>
                    <div class="search-box">
                        <input type="text" placeholder="搜索磁盘..." oninput="filterTable('nvme-table', this.value)">
                    </div>
                    <div class="panel-body">
                        <table id="nvme-table">
                            <thead>
                                <tr>
                                    <th onclick="sortTable('nvme-table', 0)">磁盘名称</th>
                                    <th onclick="sortTable('nvme-table', 1)">型号</th>
                                    <th onclick="sortTable('nvme-table', 2)">容量</th>
                                    <th onclick="sortTable('nvme-table', 3)">存储池</th>
                                    <th onclick="sortTable('nvme-table', 4)">温度</th>
                                    <th onclick="sortTable('nvme-table', 5)">通电时间</th>
                                    <th onclick="sortTable('nvme-table', 6)">已用寿命</th>
                                    <th onclick="sortTable('nvme-table', 7)">可用备件</th>
                                    <th onclick="sortTable('nvme-table', 8)">SMART状态</th>
                                    <th onclick="sortTable('nvme-table', 9)">已读数据</th>
                                    <th onclick="sortTable('nvme-table', 10)">已写数据</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range index .GroupedDisksStr "NVME_SSD"}}
                                <tr>
                                    <td>{{.Name}}</td>
                                    <td>{{.Model}}</td>
                                    <td>{{formatSize .Size}}</td>
                                    <td>{{.Pool}}</td>
                                    <td>
                                        {{if $.ShowTemperatureBar}}
                                        {{formatTemperatureBar .GetDisplayTemperature}}
                                        {{end}}
                                        {{.GetDisplayTemperature}}
                                    </td>
                                    <td>{{formatPowerOnHours (.GetAttribute "Power_On_Hours")}}</td>
                                    <td>{{.GetAttribute "Percentage_Used"}}</td>
                                    <td>{{.GetAttribute "Available_Spare"}}</td>
                                    <td class="{{getStatusClass (.GetAttribute "Smart_Status")}}">{{.GetAttribute "Smart_Status"}}</td>
                                    <td>{{.GetAttribute "Data_Read"}}</td>
                                    <td>{{.GetAttribute "Data_Written"}}</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
                
                <!-- Virtual Devices Section -->
                {{if index .GroupedDisksStr "VIRTUAL"}}
                <div class="panel">
                    <div class="panel-header">
                        <span>虚拟设备</span>
                    </div>
                    <div class="search-box">
                        <input type="text" placeholder="搜索磁盘..." oninput="filterTable('virtual-table', this.value)">
                    </div>
                    <div class="panel-body">
                        <table id="virtual-table">
                            <thead>
                                <tr>
                                    <th onclick="sortTable('virtual-table', 0)">磁盘名称</th>
                                    <th onclick="sortTable('virtual-table', 1)">型号</th>
                                    <th onclick="sortTable('virtual-table', 2)">容量</th>
                                    <th onclick="sortTable('virtual-table', 3)">存储池</th>
                                    <th onclick="sortTable('virtual-table', 4)">类型</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range index .GroupedDisksStr "VIRTUAL"}}
                                <tr>
                                    <td>{{.Name}}</td>
                                    <td>{{.Model}}</td>
                                    <td>{{formatSize .Size}}</td>
                                    <td>{{.Pool}}</td>
                                    <td>{{.GetAttribute "Type"}}</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
            </div>
            
            <div id="controller-tab" class="tab-content">
                <!-- LSI Controllers Section -->
                {{if and .ControllerData .ControllerData.LSIControllers}}
                <div class="panel">
                    <div class="panel-header">
                        <span>LSI SAS HBA控制器</span>
                    </div>
                    <div class="panel-body">
                        <table>
                            <thead>
                                <tr>
                                    <th>控制器名称</th>
                                    <th>型号</th>
                                    <th>固件版本</th>
                                    <th>驱动版本</th>
                                    <th>温度</th>
                                    <th>设备数</th>
                                    <th>状态</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range $id, $controller := .ControllerData.LSIControllers}}
                                <tr>
                                    <td>{{$id}}</td>
                                    <td>{{$controller.Model}}</td>
                                    <td>{{$controller.FirmwareVersion}}</td>
                                    <td>{{$controller.DriverVersion}}</td>
                                    <td>{{$controller.GetDisplayTemperature}}</td>
                                    <td>{{$controller.DeviceCount}}</td>
                                    <td class="{{getStatusClass (string $controller.Status)}}">{{$controller.Status}}</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
                
                <!-- NVMe Controllers Section -->
                {{if and .ControllerData .ControllerData.NVMeControllers}}
                <div class="panel">
                    <div class="panel-header">
                        <span>NVMe控制器</span>
                    </div>
                    <div class="panel-body">
                        <table>
                            <thead>
                                <tr>
                                    <th>总线ID</th>
                                    <th>控制器描述</th>
                                    <th>温度</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range $id, $controller := .ControllerData.NVMeControllers}}
                                <tr>
                                    <td>{{$controller.Bus}}</td>
                                    <td>{{$controller.Description}}</td>
                                    <td>{{$controller.GetDisplayTemperature}}</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
            </div>
            
            {{if .HasIncrement}}
            <div id="history-tab" class="tab-content">
                <div class="panel">
                    <div class="panel-header">
                        <span>磁盘读写增量信息 (自 {{.PreviousTime}})</span>
                    </div>
                    <div class="search-box">
                        <input type="text" placeholder="搜索磁盘..." oninput="filterTable('increment-table', this.value)">
                    </div>
                    <div class="panel-body">
                        <table id="increment-table">
                            <thead>
                                <tr>
                                    <th onclick="sortTable('increment-table', 0)">磁盘名称</th>
                                    <th onclick="sortTable('increment-table', 1)">类型</th>
                                    <th onclick="sortTable('increment-table', 2)">型号</th>
                                    <th onclick="sortTable('increment-table', 3)">存储池</th>
                                    <th onclick="sortTable('increment-table', 4)">当前读取总量</th>
                                    <th onclick="sortTable('increment-table', 5)">读取增量</th>
                                    <th onclick="sortTable('increment-table', 6)">当前写入总量</th>
                                    <th onclick="sortTable('increment-table', 7)">写入增量</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range .DiskData.Disks}}
                                {{if or .ReadIncrement .WriteIncrement}}
                                <tr>
                                    <td>{{.Name}}</td>
                                    <td>{{.Type}}</td>
                                    <td>{{.Model}}</td>
                                    <td>{{.Pool}}</td>
                                    <td>{{.GetAttribute "Data_Read"}}</td>
                                    <td>{{.ReadIncrement}}</td>
                                    <td>{{.GetAttribute "Data_Written"}}</td>
                                    <td>{{.WriteIncrement}}</td>
                                </tr>
                                {{end}}
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
            {{end}}
        </div>
    </div>
    
    {{if .EnableInteractivity}}
    <script>
        // Tab switching functionality
        function openTab(evt, tabName) {
            var i, tabcontent, tablinks;
            
            // Hide all tab content
            tabcontent = document.getElementsByClassName("tab-content");
            for (i = 0; i < tabcontent.length; i++) {
                tabcontent[i].className = tabcontent[i].className.replace(" active", "");
            }
            
            // Remove active class from all tabs
            tablinks = document.getElementsByClassName("tab");
            for (i = 0; i < tablinks.length; i++) {
                tablinks[i].className = tablinks[i].className.replace(" active", "");
            }
            
            // Show the current tab and add active class
            document.getElementById(tabName).className += " active";
            evt.currentTarget.className += " active";
        }
        
        // Table sorting functionality
        function sortTable(tableId, column) {
            var table, rows, switching, i, x, y, shouldSwitch, dir = "asc";
            table = document.getElementById(tableId);
            switching = true;
            
            // Set sorting direction to ascending
            var th = table.getElementsByTagName("th")[column];
            
            // Remove sorting indicators from all headers
            var headers = table.getElementsByTagName("th");
            for (i = 0; i < headers.length; i++) {
                headers[i].setAttribute("data-sort", "");
            }
            
            // Toggle sorting direction if clicking the same column again
            if (th.getAttribute("data-sort") === "asc") {
                dir = "desc";
                th.setAttribute("data-sort", "desc");
            } else {
                th.setAttribute("data-sort", "asc");
            }
            
            // Sorting loop
            while (switching) {
                switching = false;
                rows = table.rows;
                
                for (i = 1; i < (rows.length - 1); i++) {
                    shouldSwitch = false;
                    x = rows[i].getElementsByTagName("td")[column];
                    y = rows[i + 1].getElementsByTagName("td")[column];
                    
                    // Compare values (handle numbers and text differently)
                    if (dir === "asc") {
                        if (isNaN(x.innerHTML) || isNaN(y.innerHTML)) {
                            if (x.innerHTML.toLowerCase() > y.innerHTML.toLowerCase()) {
                                shouldSwitch = true;
                                break;
                            }
                        } else {
                            if (Number(x.innerHTML) > Number(y.innerHTML)) {
                                shouldSwitch = true;
                                break;
                            }
                        }
                    } else if (dir === "desc") {
                        if (isNaN(x.innerHTML) || isNaN(y.innerHTML)) {
                            if (x.innerHTML.toLowerCase() < y.innerHTML.toLowerCase()) {
                                shouldSwitch = true;
                                break;
                            }
                        } else {
                            if (Number(x.innerHTML) < Number(y.innerHTML)) {
                                shouldSwitch = true;
                                break;
                            }
                        }
                    }
                }
                
                if (shouldSwitch) {
                    rows[i].parentNode.insertBefore(rows[i + 1], rows[i]);
                    switching = true;
                }
            }
        }
        
        // Table filtering functionality
        function filterTable(tableId, query) {
            var table = document.getElementById(tableId);
            var rows = table.getElementsByTagName("tr");
            var filter = query.toLowerCase();
            
            // Loop through all rows, starting from row 1 (skipping header)
            for (var i = 1; i < rows.length; i++) {
                var shouldShow = false;
                var cells = rows[i].getElementsByTagName("td");
                
                // Check each cell in the row
                for (var j = 0; j < cells.length; j++) {
                    var cellText = cells[j].innerText || cells[j].textContent;
                    
                    // If the cell contains the filter text, show the row
                    if (cellText.toLowerCase().indexOf(filter) > -1) {
                        shouldShow = true;
                        break;
                    }
                }
                
                // Set display style based on filter match
                rows[i].style.display = shouldShow ? "" : "none";
            }
        }
    </script>
    {{end}}
</body>
</html>`

// Template for controller-only view
const controllerOnlyTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            margin: 0;
            padding: 20px;
            color: #333;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .panel {
            background-color: white;
            border-radius: 5px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24);
            margin-bottom: 20px;
            overflow: hidden;
        }
        .panel-header {
            padding: 15px 20px;
            background-color: #0747a6;
            color: white;
            font-weight: 500;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .panel-body {
            padding: 0;
            overflow: auto;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th {
            background-color: #f4f5f7;
            text-align: left;
            padding: 10px;
            border-bottom: 1px solid #ddd;
            position: sticky;
            top: 0;
            cursor: pointer;
        }
        td {
            padding: 10px;
            border-bottom: 1px solid #eee;
            white-space: nowrap;
        }
        tr:hover {
            background-color: #f9f9f9;
        }
        .status-ok {
            color: #00875a;
            font-weight: bold;
        }
        .status-warning {
            color: #ff8b00;
            font-weight: bold;
        }
        .status-error {
            color: #de350b;
            font-weight: bold;
        }
        .last-update {
            font-size: 12px;
            color: #5e6c84;
            text-align: right;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        
        <div class="last-update">最后更新时间: {{.Timestamp}}</div>
        
        <!-- LSI Controllers Section -->
        {{if and .ControllerData .ControllerData.LSIControllers}}
        <div class="panel">
            <div class="panel-header">
                <span>LSI SAS HBA控制器</span>
            </div>
            <div class="panel-body">
                <table>
                    <thead>
                        <tr>
                            <th>控制器名称</th>
                            <th>型号</th>
                            <th>固件版本</th>
                            <th>驱动版本</th>
                            <th>温度</th>
                            <th>设备数</th>
                            <th>状态</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range $id, $controller := .ControllerData.LSIControllers}}
                        <tr>
                            <td>{{$id}}</td>
                            <td>{{$controller.Model}}</td>
                            <td>{{$controller.FirmwareVersion}}</td>
                            <td>{{$controller.DriverVersion}}</td>
                            <td>{{$controller.GetDisplayTemperature}}</td>
                            <td>{{$controller.DeviceCount}}</td>
                            <td class="{{getStatusClass (string $controller.Status)}}">{{$controller.Status}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
        {{end}}
        
        <!-- NVMe Controllers Section -->
        {{if and .ControllerData .ControllerData.NVMeControllers}}
        <div class="panel">
            <div class="panel-header">
                <span>NVMe控制器</span>
            </div>
            <div class="panel-body">
                <table>
                    <thead>
                        <tr>
                            <th>总线ID</th>
                            <th>控制器描述</th>
                            <th>温度</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range $id, $controller := .ControllerData.NVMeControllers}}
                        <tr>
                            <td>{{$controller.Bus}}</td>
                            <td>{{$controller.Description}}</td>
                            <td>{{$controller.GetDisplayTemperature}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
        {{end}}
    </div>
</body>
</html>`

// init registers the HTML formatter factory
func init() {
	// Register the HTML formatter factory
	NewHTMLFormatter = func(options map[string]interface{}) OutputFormatter {
		return createHTMLFormatter(options)
	}
}
