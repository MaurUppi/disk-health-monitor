package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
)

// Version information (can be set during build)
var (
	Version   = "0.1.0"
	BuildDate = "unknown"
)

func main() {
	// Parse command line flags
	config, options, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize the application
	app, err := NewApplication(config, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(2)
	}

	// Run the application and get exit code
	exitCode := app.Run()

	// Exit with the appropriate code
	os.Exit(exitCode)
}

// parseFlags parses command-line flags into a config object
func parseFlags() (*model.Config, map[string]interface{}, error) {
	// Reset flag parsing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	
	// Create default config
	config := model.NewDefaultConfig()

	// Define flags
	// Basic flags
	help := flag.Bool("help", false, "显示帮助信息")
	flagH := flag.Bool("h", false, "显示帮助信息 (简写)")
	version := flag.Bool("version", false, "显示版本信息")
	flagV := flag.Bool("v", false, "显示版本信息 (简写)")
	debug := flag.Bool("debug", false, "启用调试模式")
	flagD := flag.Bool("d", false, "启用调试模式 (简写)")
	verbose := flag.Bool("verbose", false, "显示详细信息")

	// Output flags
	output := flag.String("output", "", "输出到指定文件")
	flagO := flag.String("o", "", "输出到指定文件 (简写)")
	format := flag.String("format", "", "指定输出格式 (text, pdf)")
	flagF := flag.String("f", "", "指定输出格式 (简写)")
	quiet := flag.Bool("quiet", false, "静默模式，减少屏幕输出")

	// Display flags
	noGroup := flag.Bool("no-group", false, "不按类型分组显示")
	noController := flag.Bool("no-controller", false, "不显示控制器信息")
	controllerOnly := flag.Bool("controller-only", false, "只显示控制器信息")
	onlyWarnings := flag.Bool("only-warnings", false, "只显示有警告或错误的磁盘")
	compact := flag.Bool("compact", false, "使用紧凑输出模式")

	// Advanced flags
	dataFile := flag.String("data-file", "", "指定历史数据文件")
	logFile := flag.String("log-file", "", "指定日志文件")
	timeout := flag.Int("timeout", 30, "设置命令执行超时时间（秒）")
	exitOnWarning := flag.Bool("exit-on-warning", false, "发现警告时以非零状态退出")

	// Parse flags
	flag.Parse()

	// Check for help flag
	if *help || *flagH {
		printHelp()
		os.Exit(0)
	}

	// Check for version flag
	if *version || *flagV {
		fmt.Printf("磁盘健康监控工具 v%s (构建时间: %s)\n", Version, BuildDate)
		os.Exit(0)
	}

	// Check for parameter conflicts
	if *controllerOnly && *noController {
		return nil, nil, fmt.Errorf("参数冲突: --controller-only 和 --no-controller 不能同时使用")
	}

	// Apply flags to config
	config.Debug = *debug || *flagD
	config.Verbose = *verbose
	
	if *output != "" {
		config.OutputFile = *output
	} else if *flagO != "" {
		config.OutputFile = *flagO
	}
	
	if *format != "" {
		switch *format {
		case "text", "txt":
			config.OutputFormat = model.OutputFormatText
		case "pdf":
			config.OutputFormat = model.OutputFormatPDF
		default:
			return nil, nil, fmt.Errorf("不支持的输出格式: %s", *format)
		}
	} else if *flagF != "" {
		switch *flagF {
		case "text", "txt":
			config.OutputFormat = model.OutputFormatText
		case "pdf":
			config.OutputFormat = model.OutputFormatPDF
		default:
			return nil, nil, fmt.Errorf("不支持的输出格式: %s", *flagF)
		}
	}
	
	if *dataFile != "" {
		config.DataFile = *dataFile
	}
	
	if *logFile != "" {
		config.LogFile = *logFile
	}
	
	config.NoGroup = *noGroup
	config.NoController = *noController
	config.ControllerOnly = *controllerOnly
	config.CommandTimeout = time.Duration(*timeout) * time.Second

	// Store additional options that aren't in the core Config struct
	additionalOptions := make(map[string]interface{})
	additionalOptions["only_warnings"] = *onlyWarnings
	additionalOptions["exit_on_warning"] = *exitOnWarning
	additionalOptions["quiet"] = *quiet
	additionalOptions["compact"] = *compact

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, nil, err
	}

	// Auto-generate output file name if not specified but format is
	if config.OutputFile == "" && format != nil && *format != "" {
		config.SetupOutputFile()
	}

	return config, additionalOptions, nil
}

// printHelp prints detailed help information
func printHelp() {
	helpText := `用法: disk-health-monitor [选项...]

选项:
  基本选项:
    -h, --help             显示此帮助信息并退出
    -v, --version          显示版本信息并退出
    -d, --debug            启用调试模式
    --verbose              显示详细输出信息

  输出选项:
    -o, --output FILE      输出到指定文件
    -f, --format FORMAT    指定输出格式 (text, pdf)
    --quiet                静默模式，减少屏幕输出

  显示选项:
    --no-group             不按类型分组显示
    --no-controller        不显示控制器信息
    --controller-only      只显示控制器信息
    --only-warnings        只显示有警告或错误的磁盘
    --compact              使用紧凑输出模式

  高级选项:
    --data-file FILE       指定历史数据文件
    --log-file FILE        指定日志文件
    --timeout SECONDS      设置命令执行超时时间
    --exit-on-warning      发现警告时以非零状态退出

例子:
  disk-health-monitor                    # 显示所有磁盘和控制器信息
  disk-health-monitor -o report.txt      # 将输出保存到文件
  disk-health-monitor --only-warnings    # 只显示有问题的磁盘
  disk-health-monitor --controller-only  # 只显示控制器信息
`
	fmt.Print(helpText)
}
