package model

import (
	"testing"
	"time"
)

func TestClassifyDiskType(t *testing.T) {
	// 测试NVMe类型判断
	if ClassifyDiskType("nvme0n1", "SSD", "Samsung 980 Pro") != DiskTypeNVMESSD {
		t.Error("NVMe disk not correctly classified")
	}
	
	// 测试HDD类型判断
	if ClassifyDiskType("sda", "HDD", "WDC WD40EFRX") != DiskTypeSASHDD {
		t.Error("HDD disk not correctly classified")
	}
	
	// 测试SSD类型判断
	if ClassifyDiskType("sdb", "SSD", "Samsung 870 EVO") != DiskTypeSASSSD {
		t.Error("SSD disk not correctly classified")
	}
	
	// 测试虚拟设备
	if ClassifyDiskType("sdc", "SSD", "VMware Virtual Disk") != DiskTypeVirtual {
		t.Error("Virtual disk not correctly classified")
	}
}

func TestNewDisk(t *testing.T) {
	// 创建一个新的磁盘对象
	disk := NewDisk("sda", "SSD", "Samsung 870 EVO", "1 TB")
	
	if disk.Name != "sda" {
		t.Errorf("Expected name 'sda', got '%s'", disk.Name)
	}
	
	if disk.RawType != "SSD" {
		t.Errorf("Expected RawType 'SSD', got '%s'", disk.RawType)
	}
	
	if disk.Model != "Samsung 870 EVO" {
		t.Errorf("Expected model 'Samsung 870 EVO', got '%s'", disk.Model)
	}
	
	if disk.Size != "1 TB" {
		t.Errorf("Expected size '1 TB', got '%s'", disk.Size)
	}
	
	if disk.Type != DiskTypeSASSSD {
		t.Errorf("Expected Type '%s', got '%s'", DiskTypeSASSSD, disk.Type)
	}
	
	if disk.Pool != "未分配" {
		t.Errorf("Expected pool '未分配', got '%s'", disk.Pool)
	}
	
	if disk.Status != DiskStatusUnknown {
		t.Errorf("Expected status '%s', got '%s'", DiskStatusUnknown, disk.Status)
	}
}

func TestDisk_GetStatus(t *testing.T) {
	// 测试状态推断
	disk := NewDisk("sda", "SSD", "Samsung 870 EVO", "1 TB")
	
	// 状态未知时
	if disk.GetStatus() != DiskStatusUnknown {
		t.Errorf("Expected unknown status, got '%s'", disk.GetStatus())
	}
	
	// 设置SMART状态为PASSED
	disk.SMARTData["Smart_Status"] = "PASSED"
	if disk.GetStatus() != DiskStatusOK {
		t.Errorf("Expected OK status, got '%s'", disk.GetStatus())
	}
	
	// 设置SMART状态为WARNING
	disk.SMARTData["Smart_Status"] = "WARNING"
	if disk.GetStatus() != DiskStatusWarning {
		t.Errorf("Expected warning status, got '%s'", disk.GetStatus())
	}
	
	// 设置SMART状态为FAILED
	disk.SMARTData["Smart_Status"] = "FAILED"
	if disk.GetStatus() != DiskStatusError {
		t.Errorf("Expected error status, got '%s'", disk.GetStatus())
	}
	
	// 测试未修正错误导致的警告状态
	disk = NewDisk("sdb", "SSD", "Samsung 870 EVO", "1 TB")
	disk.SMARTData["Uncorrected_Errors"] = "5"
	if disk.GetStatus() != DiskStatusWarning {
		t.Errorf("Expected warning status due to errors, got '%s'", disk.GetStatus())
	}
}

func TestDisk_GetDisplayTemperature(t *testing.T) {
	// 测试温度显示
	disk := NewDisk("sda", "SSD", "Samsung 870 EVO", "1 TB")
	
	// 没有温度数据时
	if disk.GetDisplayTemperature() != "N/A" {
		t.Errorf("Expected 'N/A' for missing temperature, got '%s'", disk.GetDisplayTemperature())
	}
	
	// 设置温度
	disk.SMARTData["Temperature"] = "35"
	if disk.GetDisplayTemperature() != "35°C" {
		t.Errorf("Expected '35°C', got '%s'", disk.GetDisplayTemperature())
	}
}

func TestDisk_GetAttribute(t *testing.T) {
	// 测试获取属性
	disk := NewDisk("sda", "SSD", "Samsung 870 EVO", "1 TB")
	
	// 不存在的属性
	if disk.GetAttribute("NonExistentAttribute") != "N/A" {
		t.Errorf("Expected 'N/A' for missing attribute, got '%s'", disk.GetAttribute("NonExistentAttribute"))
	}
	
	// 设置属性
	disk.SMARTData["Power_On_Hours"] = "8760"
	if disk.GetAttribute("Power_On_Hours") != "8760" {
		t.Errorf("Expected '8760', got '%s'", disk.GetAttribute("Power_On_Hours"))
	}
}

func TestDiskData(t *testing.T) {
	// 创建磁盘数据集合
	dd := NewDiskData()
	
	// 验证初始状态
	if dd.GetDiskCount() != 0 {
		t.Errorf("Expected disk count 0, got %d", dd.GetDiskCount())
	}
	
	// 添加磁盘
	disk1 := NewDisk("sda", "SSD", "Samsung 870 EVO", "1 TB")
	disk2 := NewDisk("sdb", "HDD", "WDC WD40EFRX", "4 TB")
	disk3 := NewDisk("nvme0n1", "SSD", "Samsung 980 Pro", "1 TB")
	
	dd.AddDisk(disk1)
	dd.AddDisk(disk2)
	dd.AddDisk(disk3)
	
	// 验证磁盘计数
	if dd.GetDiskCount() != 3 {
		t.Errorf("Expected disk count 3, got %d", dd.GetDiskCount())
	}
	
	if dd.GetSSDCount() != 2 {
		t.Errorf("Expected SSD count 2, got %d", dd.GetSSDCount())
	}
	
	if dd.GetHDDCount() != 1 {
		t.Errorf("Expected HDD count 1, got %d", dd.GetHDDCount())
	}
	
	// 测试分组
	if len(dd.GroupedDisks[DiskTypeSASSSD]) != 1 {
		t.Errorf("Expected 1 SAS SSD, got %d", len(dd.GroupedDisks[DiskTypeSASSSD]))
	}
	
	if len(dd.GroupedDisks[DiskTypeSASHDD]) != 1 {
		t.Errorf("Expected 1 SAS HDD, got %d", len(dd.GroupedDisks[DiskTypeSASHDD]))
	}
	
	if len(dd.GroupedDisks[DiskTypeNVMESSD]) != 1 {
		t.Errorf("Expected 1 NVMe SSD, got %d", len(dd.GroupedDisks[DiskTypeNVMESSD]))
	}
	
	// 测试排序
	dd.SortDisks()
	if dd.Disks[0].Name != "nvme0n1" || dd.Disks[1].Name != "sda" || dd.Disks[2].Name != "sdb" {
		t.Errorf("Disks not sorted correctly: %v", []string{dd.Disks[0].Name, dd.Disks[1].Name, dd.Disks[2].Name})
	}
	
	// 测试状态统计
	// 设置一个磁盘为警告状态
	disk1.SMARTData["Smart_Status"] = "WARNING"
	disk1.UpdateStatus()
	
	// 设置一个磁盘为错误状态
	disk2.SMARTData["Smart_Status"] = "FAILED"
	disk2.UpdateStatus()
	
	if dd.GetWarningCount() != 1 {
		t.Errorf("Expected warning count 1, got %d", dd.GetWarningCount())
	}
	
	if dd.GetErrorCount() != 1 {
		t.Errorf("Expected error count 1, got %d", dd.GetErrorCount())
	}
	
	// 测试磁盘属性获取
	sasssdAttrs := dd.GetDiskAttributes(DiskTypeSASSSD)
	if len(sasssdAttrs) != 10 {
		t.Errorf("Expected 10 SAS SSD attributes, got %d", len(sasssdAttrs))
	}
	
	nvmessdAttrs := dd.GetDiskAttributes(DiskTypeNVMESSD)
	if len(nvmessdAttrs) != 10 {
		t.Errorf("Expected 10 NVMe SSD attributes, got %d", len(nvmessdAttrs))
	}
	
	// 测试历史数据
	if dd.HasPreviousData() {
		t.Error("Expected no previous data")
	}
	
	prevData := map[string]map[string]string{
		"sda": {
			"Data_Read": "10 TB",
			"Data_Written": "5 TB",
		},
	}
	dd.SetPreviousData(prevData, "2023-01-01 00:00:00")
	
	if !dd.HasPreviousData() {
		t.Error("Expected to have previous data")
	}
	
	// 验证收集时间 - 使用更宽松的检查方式
	collectionTime := dd.GetCollectionTime()
	
	// 验证格式正确
	_, err := time.Parse("2006-01-02 15:04:05", collectionTime)
	if err != nil {
		t.Errorf("Invalid collection time format: %s", collectionTime)
	}
}
