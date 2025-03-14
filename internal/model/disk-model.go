package model

import (
	"sort"
	"strings"
	"time"
)

// DiskType 磁盘类型
type DiskType string

const (
	// DiskTypeSASHDD SAS/SATA机械硬盘
	DiskTypeSASHDD DiskType = "SAS_HDD"
	// DiskTypeSASSSD SAS/SATA固态硬盘
	DiskTypeSASSSD DiskType = "SAS_SSD"
	// DiskTypeNVMESSD NVMe固态硬盘
	DiskTypeNVMESSD DiskType = "NVME_SSD"
	// DiskTypeVirtual 虚拟设备
	DiskTypeVirtual DiskType = "VIRTUAL"
)

// DiskStatus 磁盘状态
type DiskStatus string

const (
	// DiskStatusOK 磁盘状态正常
	DiskStatusOK DiskStatus = "PASSED"
	// DiskStatusWarning 磁盘状态警告
	DiskStatusWarning DiskStatus = "WARNING"
	// DiskStatusError 磁盘状态错误
	DiskStatusError DiskStatus = "FAILED"
	// DiskStatusUnknown 磁盘状态未知
	DiskStatusUnknown DiskStatus = "UNKNOWN"
)

// SMARTData SMART数据
type SMARTData map[string]string

// DiskAttribute 磁盘属性
type DiskAttribute struct {
	Name        string // 属性名
	DisplayName string // 显示名称
	Unit        string // 单位
}

// Disk 磁盘基本信息
type Disk struct {
	Name          string       // 设备名称
	Type          DiskType     // 设备类型
	RawType       string       // 原始类型字符串
	Model         string       // 设备型号
	Size          string       // 设备容量
	Pool          string       // 所属存储池
	SMARTData     SMARTData    // SMART数据
	Status        DiskStatus   // 磁盘状态
	ReadIncrement string       // 读增量
	WriteIncrement string      // 写增量
}

// NewDisk 创建一个新的磁盘对象
func NewDisk(name, rawType, model, size string) *Disk {
	disk := &Disk{
		Name:      name,
		RawType:   rawType,
		Model:     model,
		Size:      size,
		Pool:      "未分配",
		SMARTData: make(SMARTData),
		Status:    DiskStatusUnknown,
	}
	
	// 根据原始类型确定磁盘类型
	disk.Type = ClassifyDiskType(name, rawType, model)
	
	return disk
}

// ClassifyDiskType 将磁盘分类为SAS SSD、SAS HDD或NVMe SSD
func ClassifyDiskType(diskName, diskType, diskModel string) DiskType {
	// 检查是否为虚拟设备
	if strings.Contains(diskModel, "VMware") || strings.Contains(diskModel, "Virtual") {
		return DiskTypeVirtual
	}
	
	// 检查是否为NVMe设备
	if strings.HasPrefix(diskName, "nvme") {
		return DiskTypeNVMESSD
	}
	
	// 检查是否为机械硬盘
	if strings.ToUpper(diskType) == "HDD" {
		return DiskTypeSASHDD
	}
	
	// 默认为SAS/SATA SSD
	return DiskTypeSASSSD
}

// GetStatus 根据SMART数据推断磁盘状态
func (d *Disk) GetStatus() DiskStatus {
	// 如果已经设置了状态，直接返回
	if d.Status != DiskStatusUnknown {
		return d.Status
	}
	
	// 从SMART数据获取状态
	if smartStatus, ok := d.SMARTData["Smart_Status"]; ok {
		switch strings.ToUpper(smartStatus) {
		case "PASSED", "OK":
			return DiskStatusOK
		case "WARNING", "警告":
			return DiskStatusWarning
		case "FAILED", "错误":
			return DiskStatusError
		}
	}
	
	// 其他状态判断逻辑，如检查未修正错误、温度等
	if errors, ok := d.SMARTData["Uncorrected_Errors"]; ok {
		if errors != "0" && errors != "" {
			return DiskStatusWarning
		}
	}
	
	// 没有足够信息判断状态
	return DiskStatusUnknown
}

// UpdateStatus 更新磁盘状态
func (d *Disk) UpdateStatus() {
	d.Status = d.GetStatus()
}

// GetDisplayTemperature 获取可显示的温度
func (d *Disk) GetDisplayTemperature() string {
	if temp, ok := d.SMARTData["Temperature"]; ok && temp != "" {
		return temp + "°C"
	}
	return "N/A"
}

// GetAttribute 获取特定属性的值
func (d *Disk) GetAttribute(name string) string {
	if value, ok := d.SMARTData[name]; ok && value != "" {
		return value
	}
	return "N/A"
}

// DiskData 磁盘数据集合
type DiskData struct {
	Disks         []*Disk                   // 磁盘列表
	GroupedDisks  map[DiskType][]*Disk      // 按类型分组的磁盘
	PreviousData  map[string]map[string]string // 上次运行的数据
	PreviousTime  string                    // 上次运行的时间
	CollectedTime time.Time                 // 收集数据的时间
}

// NewDiskData 创建一个新的磁盘数据集合
func NewDiskData() *DiskData {
	return &DiskData{
		Disks:        make([]*Disk, 0),
		GroupedDisks: make(map[DiskType][]*Disk),
		PreviousData: make(map[string]map[string]string),
		CollectedTime: time.Now(),
	}
}

// AddDisk 添加磁盘到集合
func (dd *DiskData) AddDisk(disk *Disk) {
	dd.Disks = append(dd.Disks, disk)
	
	// 更新分组
	if _, ok := dd.GroupedDisks[disk.Type]; !ok {
		dd.GroupedDisks[disk.Type] = make([]*Disk, 0)
	}
	dd.GroupedDisks[disk.Type] = append(dd.GroupedDisks[disk.Type], disk)
}

// SortDisks 对磁盘集合按名称排序
func (dd *DiskData) SortDisks() {
	// 排序主列表
	sort.Slice(dd.Disks, func(i, j int) bool {
		return dd.Disks[i].Name < dd.Disks[j].Name
	})
	
	// 排序分组列表
	for diskType := range dd.GroupedDisks {
		sort.Slice(dd.GroupedDisks[diskType], func(i, j int) bool {
			return dd.GroupedDisks[diskType][i].Name < dd.GroupedDisks[diskType][j].Name
		})
	}
}

// GetDiskCount 获取磁盘总数
func (dd *DiskData) GetDiskCount() int {
	return len(dd.Disks)
}

// GetDiskCountByType 获取特定类型的磁盘数量
func (dd *DiskData) GetDiskCountByType(diskType DiskType) int {
	if disks, ok := dd.GroupedDisks[diskType]; ok {
		return len(disks)
	}
	return 0
}

// GetSSDCount 获取SSD数量
func (dd *DiskData) GetSSDCount() int {
	return dd.GetDiskCountByType(DiskTypeSASSSD) + dd.GetDiskCountByType(DiskTypeNVMESSD)
}

// GetHDDCount 获取HDD数量
func (dd *DiskData) GetHDDCount() int {
	return dd.GetDiskCountByType(DiskTypeSASHDD)
}

// GetWarningCount 获取警告状态的磁盘数量
func (dd *DiskData) GetWarningCount() int {
	count := 0
	for _, disk := range dd.Disks {
		if disk.GetStatus() == DiskStatusWarning {
			count++
		}
	}
	return count
}

// GetErrorCount 获取错误状态的磁盘数量
func (dd *DiskData) GetErrorCount() int {
	count := 0
	for _, disk := range dd.Disks {
		if disk.GetStatus() == DiskStatusError {
			count++
		}
	}
	return count
}

// GetDiskAttributes 获取特定类型磁盘的属性列表
func (dd *DiskData) GetDiskAttributes(diskType DiskType) []DiskAttribute {
	// 按磁盘类型返回属性列表
	switch diskType {
	case DiskTypeSASSSD:
		return []DiskAttribute{
			{Name: "Temperature", DisplayName: "温度", Unit: "°C"},
			{Name: "Trip_Temperature", DisplayName: "警告温度", Unit: "°C"},
			{Name: "Power_On_Hours", DisplayName: "通电时间", Unit: "小时"},
			{Name: "Power_Cycles", DisplayName: "通电周期", Unit: "次"},
			{Name: "Percentage_Used", DisplayName: "已用寿命", Unit: "%"},
			{Name: "Smart_Status", DisplayName: "SMART状态", Unit: ""},
			{Name: "Data_Read", DisplayName: "已读数据", Unit: ""},
			{Name: "Data_Written", DisplayName: "已写数据", Unit: ""},
			{Name: "Non_Medium_Errors", DisplayName: "非介质错误", Unit: "个"},
			{Name: "Uncorrected_Errors", DisplayName: "未修正错误", Unit: "个"},
		}
	case DiskTypeSASHDD:
		return []DiskAttribute{
			{Name: "Temperature", DisplayName: "温度", Unit: "°C"},
			{Name: "Trip_Temperature", DisplayName: "警告温度", Unit: "°C"},
			{Name: "Power_On_Hours", DisplayName: "通电时间", Unit: "小时"},
			{Name: "Power_Cycles", DisplayName: "通电周期", Unit: "次"},
			{Name: "Smart_Status", DisplayName: "SMART状态", Unit: ""},
			{Name: "Data_Read", DisplayName: "已读数据", Unit: ""},
			{Name: "Data_Written", DisplayName: "已写数据", Unit: ""},
			{Name: "Non_Medium_Errors", DisplayName: "非介质错误", Unit: "个"},
			{Name: "Uncorrected_Errors", DisplayName: "未修正错误", Unit: "个"},
		}
	case DiskTypeNVMESSD:
		return []DiskAttribute{
			{Name: "Temperature", DisplayName: "温度", Unit: "°C"},
			{Name: "Warning_Temperature", DisplayName: "警告温度", Unit: "°C"},
			{Name: "Critical_Temperature", DisplayName: "临界温度", Unit: "°C"},
			{Name: "Power_On_Hours", DisplayName: "通电时间", Unit: "小时"},
			{Name: "Power_Cycles", DisplayName: "通电周期", Unit: "次"},
			{Name: "Percentage_Used", DisplayName: "已用寿命", Unit: "%"},
			{Name: "Available_Spare", DisplayName: "可用备件", Unit: "%"},
			{Name: "Smart_Status", DisplayName: "SMART状态", Unit: ""},
			{Name: "Data_Read", DisplayName: "已读数据", Unit: ""},
			{Name: "Data_Written", DisplayName: "已写数据", Unit: ""},
		}
	case DiskTypeVirtual:
		return []DiskAttribute{
			{Name: "Type", DisplayName: "设备类型", Unit: ""},
		}
	default:
		return []DiskAttribute{}
	}
}

// SetPreviousData 设置历史数据
func (dd *DiskData) SetPreviousData(data map[string]map[string]string, timestamp string) {
	dd.PreviousData = data
	dd.PreviousTime = timestamp
}

// HasPreviousData 检查是否有历史数据
func (dd *DiskData) HasPreviousData() bool {
	return dd.PreviousTime != "" && len(dd.PreviousData) > 0
}

// GetCollectionTime 获取数据收集时间的格式化字符串
func (dd *DiskData) GetCollectionTime() string {
	return dd.CollectedTime.Format("2006-01-02 15:04:05")
}
