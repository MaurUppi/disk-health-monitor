package model

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OutputFormat 定义支持的输出格式
type OutputFormat string

const (
	// OutputFormatPDF PDF格式输出
	OutputFormatPDF OutputFormat = "pdf"
	// OutputFormatText 文本格式输出
	OutputFormatText OutputFormat = "text"
	// OutputFormatJSON JSON格式输出
	OutputFormatJSON OutputFormat = "json"
	// OutputFormatHTML HTML格式输出
	OutputFormatHTML OutputFormat = "html"
)

// Config 应用配置
type Config struct {
	// 日志设置
	Debug   bool   // 是否开启调试模式
	Verbose bool   // 是否显示详细信息
	LogFile string // 日志文件路径
	LogDir  string // 日志目录(由LogFile生成)

	// 显示设置
	NoGroup        bool // 不按类型分组显示
	NoController   bool // 不显示控制器信息
	ControllerOnly bool // 只显示控制器信息

	// 输出设置
	OutputFile   string       // 输出文件路径
	OutputFormat OutputFormat // 输出格式(pdf, text, json)

	// 数据文件
	DataFile string // 历史数据文件路径
	DataDir  string // 数据目录(由DataFile生成)

	// 执行设置
	CommandTimeout time.Duration // 命令执行超时时间
	OutputEncoding string        // 输出文件编码
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	// 默认日志目录
	defaultLogDir := "/var/log"

	// 在Windows和其他平台上使用不同的默认路径
	if os.Getenv("PROGRAMDATA") != "" {
		// Windows平台
		defaultLogDir = filepath.Join(os.Getenv("PROGRAMDATA"), "disk-health-monitor")
	} else if os.Getenv("HOME") != "" {
		// Unix/Linux平台备用选项
		defaultLogDir = filepath.Join(os.Getenv("HOME"), ".disk-health-monitor")
	}

	// 构建默认日志和数据文件路径
	defaultLogFile := filepath.Join(defaultLogDir, "disk_health_monitor.log")
	defaultDataFile := filepath.Join(defaultLogDir, "disk_health_monitor_data.json")

	return &Config{
		Debug:          false,
		Verbose:        false,
		LogFile:        defaultLogFile,
		LogDir:         defaultLogDir,
		NoGroup:        false,
		NoController:   false,
		ControllerOnly: true,
		OutputFile:     "",
		OutputFormat:   OutputFormatText,
		DataFile:       defaultDataFile,
		DataDir:        defaultLogDir,
		CommandTimeout: 30 * time.Second,
		OutputEncoding: "utf8",
	}
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	// 检查并处理输出格式
	if c.OutputFormat == OutputFormatPDF {
		// 将PDF格式切换为文本格式
		c.OutputFormat = OutputFormatText
		fmt.Printf("[WARNING] PDF输出格式暂未实现，已自动切换为文本格式\n")
	}

	// 验证输出格式
	switch c.OutputFormat {
	case OutputFormatPDF, OutputFormatText, OutputFormatJSON, OutputFormatHTML:
		// 有效的格式
	default:
		return fmt.Errorf("不支持的输出格式: %s", c.OutputFormat)
	}

	// 验证输出编码
	switch c.OutputEncoding {
	case "utf8", "gbk":
		// 支持的编码
	default:
		return fmt.Errorf("不支持的输出编码: %s", c.OutputEncoding)
	}

	// 确保日志目录存在
	c.LogDir = filepath.Dir(c.LogFile)
	if err := os.MkdirAll(c.LogDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 确保数据目录存在
	c.DataDir = filepath.Dir(c.DataFile)
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	return nil
}

// SetupOutputFile 如果未指定输出文件，根据格式设置一个默认值
func (c *Config) SetupOutputFile() {
	// 如果已经指定了输出文件，不做任何操作
	if c.OutputFile != "" {
		return
	}

	// 根据格式设置默认输出文件
	timeStr := time.Now().Format("20060102_150405")
	switch c.OutputFormat {
	case OutputFormatPDF:
		c.OutputFile = fmt.Sprintf("disk_health_%s.pdf", timeStr)
	case OutputFormatText:
		c.OutputFile = fmt.Sprintf("disk_health_%s.txt", timeStr)
	case OutputFormatJSON:
		c.OutputFile = fmt.Sprintf("disk_health_%s.json", timeStr)
	case OutputFormatHTML:
		c.OutputFile = fmt.Sprintf("disk_health_%s.html", timeStr)
	}
}
