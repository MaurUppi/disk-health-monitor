package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// DiskCollector 实现磁盘信息收集
type DiskCollector struct {
	config          *model.Config
	logger          system.Logger
	commandRunner   system.CommandRunner
	smartCollector  *SMARTCollector
	poolCollector   *PoolCollector
}

// NewDiskCollector 创建一个新的磁盘收集器
func NewDiskCollector(config *model.Config, logger system.Logger, runner system.CommandRunner) *DiskCollector {
	smartCollector := NewSMARTCollector(config, logger, runner)
	poolCollector := NewPoolCollector(config, logger, runner)
	
	return &DiskCollector{
		config:         config,
		logger:         logger,
		commandRunner:  runner,
		smartCollector: smartCollector,
		poolCollector:  poolCollector,
	}
}

// Collect 收集所有磁盘信息
func (d *DiskCollector) Collect(ctx context.Context) (*model.DiskData, error) {
	// 创建磁盘数据对象
	diskData := model.NewDiskData()
	var collectionErrors []error
	
	// 获取磁盘列表
	disks, err := d.GetDisksFromMidclt(ctx)
	if err != nil || len(disks) == 0 {
		d.logger.Info("从midclt获取磁盘列表失败，尝试使用lsblk")
		disks, err = d.GetDisksFromLsblk(ctx)
		if err != nil || len(disks) == 0 {
			d.logger.Error("无法获取磁盘列表: %v", err)
			collectionErrors = append(collectionErrors, fmt.Errorf("disk list collection failed: %w", err))
		}
	}
	
	// 如果无法获取磁盘列表，返回错误
	if len(disks) == 0 {
		return diskData, fmt.Errorf("failed to get disk list")
	}
	
	// 获取存储池信息
	poolInfo, err := d.poolCollector.Collect(ctx)
	if err != nil {
		d.logger.Error("获取存储池信息失败: %v", err)
		collectionErrors = append(collectionErrors, fmt.Errorf("pool info collection failed: %w", err))
	}
	
	// 加载历史数据
	prevData, prevTime := d.LoadPreviousDiskData()
	diskData.SetPreviousData(prevData, prevTime)
	
	// 并发收集SMART数据
	disksWithSMART, err := d.collectSMARTData(ctx, disks, poolInfo)
	if err != nil {
		d.logger.Error("收集SMART数据时发生错误: %v", err)
		collectionErrors = append(collectionErrors, fmt.Errorf("SMART data collection failed: %w", err))
	}
	
	// 处理读写增量
	disksWithSMART = d.processIncrements(disksWithSMART, prevData)
	
	// 将磁盘添加到磁盘数据对象
	for _, disk := range disksWithSMART {
		diskData.AddDisk(disk)
	}
	
	// 排序磁盘
	diskData.SortDisks()
	
	// 保存当前数据供下次比较
	if err := d.SaveDiskData(disksWithSMART); err != nil {
		d.logger.Error("保存磁盘数据失败: %v", err)
	}
	
	// 如果有错误，返回结果但包含错误信息
	if len(collectionErrors) > 0 {
		if len(collectionErrors) == 1 {
			return diskData, collectionErrors[0]
		}
		return diskData, fmt.Errorf("multiple errors during disk data collection: %v", collectionErrors)
	}
	
	return diskData, nil
}

// GetDisksFromMidclt 使用midclt获取磁盘列表
func (d *DiskCollector) GetDisksFromMidclt(ctx context.Context) ([]*model.Disk, error) {
	d.logger.Info("获取磁盘列表...")
	
	output, err := d.commandRunner.Run(ctx, "midclt call disk.query")
	if err != nil {
		d.logger.Error("midclt调用失败: %v", err)
		return nil, err
	}
	
	var disksData []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &disksData); err != nil {
		d.logger.Error("解析midclt输出失败: %v", err)
		return nil, err
	}
	
	var disks []*model.Disk
	for _, diskData := range disksData {
		name, _ := diskData["name"].(string)
		if name == "" {
			continue
		}
		
		diskModel, _ := diskData["model"].(string)
		size := fmt.Sprintf("%v", diskData["size"])
		diskType, _ := diskData["type"].(string)
		
		disk := model.NewDisk(name, diskType, diskModel, size)
		disks = append(disks, disk)
	}
	
	d.logger.Info("找到%d个磁盘", len(disks))
	return disks, nil
}

// GetDisksFromLsblk 使用lsblk获取磁盘列表（备用方法）
func (d *DiskCollector) GetDisksFromLsblk(ctx context.Context) ([]*model.Disk, error) {
	d.logger.Info("尝试使用lsblk获取磁盘列表")
	
	output, err := d.commandRunner.Run(ctx, "lsblk -d -o NAME,TYPE,MODEL,SIZE -n | grep 'disk'")
	if err != nil {
		d.logger.Error("使用lsblk获取磁盘列表失败: %v", err)
		return nil, err
	}
	
	var disks []*model.Disk
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		
		name := parts[0]
		diskType := "HDD" // 默认为HDD
		
		if strings.Contains(strings.ToLower(name), "nvme") {
			diskType = "SSD"
		}
		
		var diskModel string
		if len(parts) > 3 {
			diskModel = strings.Join(parts[2:len(parts)-1], " ")
		} else {
			diskModel = parts[2]
		}
		
		size := parts[len(parts)-1]
		
		disk := model.NewDisk(name, diskType, diskModel, size)
		disks = append(disks, disk)
	}
	
	d.logger.Info("使用lsblk找到%d个磁盘", len(disks))
	return disks, nil
}

// collectSMARTData 并发收集所有磁盘的SMART数据
func (d *DiskCollector) collectSMARTData(ctx context.Context, disks []*model.Disk, poolInfo map[string]string) ([]*model.Disk, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	resultDisks := make([]*model.Disk, 0, len(disks))
	errorsChan := make(chan error, len(disks))
	
	// 使用信号量限制并发数量
	semaphore := make(chan struct{}, 5) // 最多5个并发
	
	for _, disk := range disks {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量
		
		go func(disk *model.Disk) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量
			
			diskName := disk.Name
			diskType := string(disk.RawType)
			diskModel := disk.Model
			
			// 设置存储池信息
			if pool, ok := poolInfo[diskName]; ok {
				disk.Pool = pool
			} else {
				disk.Pool = "未分配"
			}
			
			d.logger.Info("处理磁盘: %s (类型: %s, 型号: %s, 池: %s)",
				diskName, diskType, diskModel, disk.Pool)
			
			// 收集SMART数据
			smartData, err := d.smartCollector.GetSMARTData(ctx, diskName, diskType, diskModel)
			if err != nil {
				d.logger.Error("获取磁盘%s的SMART数据失败: %v", diskName, err)
				errorsChan <- fmt.Errorf("failed to collect SMART data for %s: %w", diskName, err)
				return
			}
			
			// 设置SMART数据
			for k, v := range smartData {
				disk.SMARTData[k] = v
			}
			
			// 更新磁盘状态
			disk.UpdateStatus()
			
			// 添加到结果
			mu.Lock()
			resultDisks = append(resultDisks, disk)
			mu.Unlock()
		}(disk)
	}
	
	wg.Wait()
	close(errorsChan)
	
	// 收集所有错误
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}
	
	// 返回结果，即使有错误也返回已收集的数据
	if len(errors) > 0 {
		if len(errors) == 1 {
			return resultDisks, errors[0]
		}
		return resultDisks, fmt.Errorf("multiple errors during SMART data collection: %v", errors)
	}
	
	return resultDisks, nil
}

// processIncrements 处理读写增量数据
func (d *DiskCollector) processIncrements(disks []*model.Disk, prevData map[string]map[string]string) []*model.Disk {
	// 如果没有历史数据，直接返回
	if len(prevData) == 0 {
		return disks
	}
	
	for _, disk := range disks {
		diskName := disk.Name
		
		// 查找上次的数据
		prevDiskData, ok := prevData[diskName]
		if !ok {
			continue
		}
		
		// 计算读增量
		if dataRead, ok := disk.SMARTData["Data_Read"]; ok && dataRead != "" {
			if prevDataRead, ok := prevDiskData["Data_Read"]; ok && prevDataRead != "" {
				disk.ReadIncrement = d.calculateSizeIncrement(prevDataRead, dataRead)
			}
		}
		
		// 计算写增量
		if dataWritten, ok := disk.SMARTData["Data_Written"]; ok && dataWritten != "" {
			if prevDataWritten, ok := prevDiskData["Data_Written"]; ok && prevDataWritten != "" {
				disk.WriteIncrement = d.calculateSizeIncrement(prevDataWritten, dataWritten)
			}
		}
	}
	
	return disks
}

// calculateSizeIncrement 计算两个大小字符串之间的增量
func (d *DiskCollector) calculateSizeIncrement(oldValue, newValue string) string {
	oldBytes, errOld := d.parseSizeToBytes(oldValue)
	newBytes, errNew := d.parseSizeToBytes(newValue)
	
	if errOld != nil || errNew != nil {
		return "N/A"
	}
	
	diffBytes := newBytes - oldBytes
	if diffBytes < 0 {
		// 如果是负值，可能是设备重置了计数器
		return "重置"
	}
	
	return d.smartCollector.formatSize(diffBytes)
}

// parseSizeToBytes 将大小字符串解析为字节数
func (d *DiskCollector) parseSizeToBytes(sizeStr string) (float64, error) {
	// 使用SMART收集器的标准化方法
	normalizedSize := d.smartCollector.normalizeSize(sizeStr)
	if normalizedSize == "N/A" {
		return 0, fmt.Errorf("invalid size string: %s", sizeStr)
	}
	
	// 解析标准化后的大小
	var value float64
	var unit string
	_, err := fmt.Sscanf(normalizedSize, "%f %s", &value, &unit)
	if err != nil {
		return 0, err
	}
	
	// 将值转换为字节
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
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
	
	return bytes, nil
}

// SaveDiskData 保存当前磁盘数据，用于下次比较
func (d *DiskCollector) SaveDiskData(disks []*model.Disk) error {
	// 构建磁盘数据映射
	diskData := make(map[string]map[string]string)
	
	for _, disk := range disks {
		// 只保存需要的属性
		diskData[disk.Name] = map[string]string{
			"Data_Read":    disk.SMARTData["Data_Read"],
			"Data_Written": disk.SMARTData["Data_Written"],
		}
	}
	
	// 构建数据存储结构
	data := struct {
		Timestamp string                            `json:"timestamp"`
		Disks     map[string]map[string]string `json:"disks"`
	}{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Disks:     diskData,
	}
	
	// 将数据转换为JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshaling failed: %w", err)
	}
	
	// 写入文件
	if err := os.WriteFile(d.config.DataFile, jsonData, 0644); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}
	
	d.logger.Debug("成功保存磁盘数据到: %s", d.config.DataFile)
	return nil
}

// LoadPreviousDiskData 加载上次运行的磁盘数据
func (d *DiskCollector) LoadPreviousDiskData() (map[string]map[string]string, string) {
	d.logger.Info("加载上次运行的磁盘数据以计算增量...")
	
	var data struct {
		Timestamp string                            `json:"timestamp"`
		Disks     map[string]map[string]string `json:"disks"`
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(d.config.DataFile); os.IsNotExist(err) {
		d.logger.Info("未找到上次运行的数据，将只显示当前状态")
		return make(map[string]map[string]string), ""
	}
	
	// 读取文件
	fileData, err := os.ReadFile(d.config.DataFile)
	if err != nil {
		d.logger.Error("读取上次运行的磁盘数据失败: %v", err)
		return make(map[string]map[string]string), ""
	}
	
	// 解析JSON
	if err := json.Unmarshal(fileData, &data); err != nil {
		d.logger.Error("解析上次运行的磁盘数据失败: %v", err)
		return make(map[string]map[string]string), ""
	}
	
	// 记录时间戳
	if data.Timestamp != "" {
		d.logger.Info("上次运行时间: %s", data.Timestamp)
	}
	
	return data.Disks, data.Timestamp
}
