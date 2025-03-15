# TrueNAS Disk Health Monitor

A Go-based tool for monitoring disk health status on TrueNAS systems, showing SMART health information in a well-organized tabular format.

![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)

## Features

- Collects and displays SMART health status for all disk types (SAS/SATA SSD, SAS/SATA HDD, NVMe SSD)
- Gathers storage controller information (LSI/SAS and NVMe)
- Shows disk-to-pool mapping from ZFS
- Displays key health metrics: temperature, power-on hours, read/write volume, error counts, etc.
- Organizes disks by type with customizable output formats
- Calculates read/write increment data between runs
- Supports both console output and file output

## Requirements

- Go 1.18 or later (for building from source)
- TrueNAS/FreeBSD/Linux environment for full functionality
- Required tools: `smartctl` (from smartmontools), `midclt` (on TrueNAS), `lspci`

## Installation

### Pre-built Binaries

Download the latest release package from the [Releases](https://github.com/MaurUppi/disk-health-monitor/releases) page.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/MaurUppi/disk-health-monitor.git
cd disk-health-monitor

# Build
go build -o disk-health-monitor

# Install (optional)
sudo mv disk-health-monitor /usr/local/bin/
```

## Usage

### Basic Usage

```bash
# Display all disk and controller information
./disk-health-monitor

# Save output to a file
./disk-health-monitor -o report.txt

# Display only controller information
./disk-health-monitor --controller-only

# Use compact mode (fewer columns)
./disk-health-monitor --compact
```

### Command-Line Options

```
Usage: disk-health-monitor [options...]

Options:
  Basic options:
    -h, --help             Show help information
    -v, --version          Show version information
    -d, --debug            Enable debug mode
    --verbose              Show verbose information

  Output options:
    -o, --output FILE      Save output to specified file
    -f, --format FORMAT    Report format (text, html)
    --compact              Use compact mode (fewer columns)
    --quiet                Quiet mode, reduce screen output

  Display options:
    --no-group             Don't group disks by type
    --no-controller        Don't show controller information
    --controller-only      Only show controller information

  Advanced options:
    --data-file FILE       Specify history data file
    --log-file FILE        Specify log file
```

## Building on Windows

This tool is primarily designed for TrueNAS/FreeBSD/Linux systems, but it can be cross-compiled on Windows for deployment. Use the included `BuildOnWin.bat` script:

1. Install Go on your Windows system
2. Install Git Bash or similar UNIX-like environment
3. Open command prompt in the project directory
4. Run `BuildOnWin.bat`

The script will:
- Set up the proper environment variables for cross-compilation
- Build executables for Windows, Linux, and FreeBSD platforms
- Place the compiled binaries in the `./bin` directory

```cmd
BuildOnWin.bat
```

## Sample Output

When run on a TrueNAS system, the tool provides detailed disk and controller health information:

```
=== TrueNAS磁盘健康监控 ===

系统摘要:
- 总磁盘数: 20 (SSD: 17, HDD: 2)
- 警告数: 0
- 错误数: 0
- 控制器数: 7

--- SAS/SATA 固态硬盘 ---

| 名称 | 型号              | 容量    | 存储池     | 温度  | 通电时间    | 已用寿命 | SMART状态 | ...
|------|-------------------|---------|------------|-------|-------------|----------|-----------|----
| sda  | PA33N3T8_EMC3840  | 3.8 TB  | MediaFiles | 43°C  | 6y 6m 16d   | 4%       | 正常      | ...
...

--- SAS/SATA 机械硬盘 ---

...

--- NVMe 固态硬盘 ---

...

--- 虚拟设备 ---

...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

---

# TrueNAS磁盘健康监控工具

一个基于Go语言的TrueNAS系统磁盘健康状态监控工具，以表格形式展示所有磁盘的SMART健康状态信息。

![许可证](https://img.shields.io/github/license/MaurUppi/disk-health-monitor)
![Go版本](https://img.shields.io/badge/Go-1.18%2B-blue)

## 功能特点

- 收集并显示所有类型磁盘的SMART健康状态（SAS/SATA固态硬盘、SAS/SATA机械硬盘、NVMe固态硬盘）
- 获取存储控制器信息（LSI/SAS和NVMe）
- 显示ZFS存储池与磁盘的映射关系
- 展示关键健康指标：温度、通电时间、读写量、错误计数等
- 按类型组织磁盘，支持自定义输出格式
- 计算两次运行之间的读写数据增量
- 支持控制台输出和文件输出

## 系统要求

- Go 1.18或更高版本（从源代码构建时需要）
- TrueNAS/FreeBSD/Linux环境以获得完整功能支持
- 必需工具：`smartctl`（来自smartmontools）、`midclt`（TrueNAS系统）、`lspci`

## 安装方法

### 预编译二进制文件

从[Releases](https://github.com/MaurUppi/disk-health-monitor/releases)页面下载最新的发布包。

### 从源代码构建

```bash
# 克隆仓库
git clone https://github.com/MaurUppi/disk-health-monitor.git
cd disk-health-monitor

# 构建
go build -o disk-health-monitor

# 安装（可选）
sudo mv disk-health-monitor /usr/local/bin/
```

## 使用方法

### 基本用法

```bash
# 显示所有磁盘和控制器信息
./disk-health-monitor

# 将输出保存到文件
./disk-health-monitor -o report.txt

# 仅显示控制器信息
./disk-health-monitor --controller-only

# 使用紧凑模式（显示更少的列）
./disk-health-monitor --compact
```

### 命令行选项

```
用法: disk-health-monitor [选项...]

选项:
  基本选项:
    -h, --help             显示帮助信息
    -v, --version          显示版本信息
    -d, --debug            启用调试模式
    --verbose              显示详细信息

  输出选项:
    -o, --output 文件名    将输出保存到指定文件
    -f, --format FORMAT    指定输出格式 (text, html)
    --compact              使用紧凑模式（减少显示列数）
    --quiet                安静模式，减少屏幕输出

  显示选项:
    --no-group             不按类型分组显示磁盘
    --no-controller        不显示控制器信息
    --controller-only      仅显示控制器信息

  高级选项:
    --data-file 文件名     指定历史数据文件
    --log-file 文件名      指定日志文件
```

## 在Windows上构建

该工具主要为TrueNAS/FreeBSD/Linux系统设计，但可以在Windows系统上交叉编译后部署。使用随附的`BuildOnWin.bat`脚本：

1. 在Windows系统上安装Go
2. 安装Git Bash或类似的UNIX环境
3. 在项目目录中打开命令提示符
4. 运行`BuildOnWin.bat`

该脚本将：
- 设置跨平台编译的环境变量
- 为Windows、Linux和FreeBSD平台构建可执行文件
- 将编译后的二进制文件放置在`./bin`目录中

```cmd
BuildOnWin.bat
```

## 输出示例

在TrueNAS系统上运行时，该工具提供详细的磁盘和控制器健康信息：

```
=== TrueNAS磁盘健康监控 ===

系统摘要:
- 总磁盘数: 20 (SSD: 17, HDD: 2)
- 警告数: 0
- 错误数: 0
- 控制器数: 7

--- SAS/SATA 固态硬盘 ---

| 名称 | 型号              | 容量    | 存储池     | 温度  | 通电时间    | 已用寿命 | SMART状态 | ...
|------|-------------------|---------|------------|-------|-------------|----------|-----------|----
| sda  | PA33N3T8_EMC3840  | 3.8 TB  | MediaFiles | 43°C  | 6y 6m 16d   | 4%       | 正常      | ...
...

--- SAS/SATA 机械硬盘 ---

...

--- NVMe 固态硬盘 ---

...

--- 虚拟设备 ---

...
```

## 参与贡献

欢迎贡献！请随时提交Pull Request。

1. Fork本仓库
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m '添加一些很棒的功能'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开Pull Request

## 许可证

该项目采用MIT许可证 - 详情请参阅LICENSE文件。
