package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// PoolCollector 实现存储池信息收集
type PoolCollector struct {
	config        *model.Config
	logger        system.Logger
	commandRunner system.CommandRunner
}

// NewPoolCollector 创建一个新的存储池收集器
func NewPoolCollector(config *model.Config, logger system.Logger, runner system.CommandRunner) *PoolCollector {
	return &PoolCollector{
		config:        config,
		logger:        logger,
		commandRunner: runner,
	}
}

// Collect 收集存储池信息
func (p *PoolCollector) Collect(ctx context.Context) (map[string]string, error) {
	// 首先尝试从midclt获取
	poolInfo, err := p.GetPoolInfo(ctx)
	if err != nil || len(poolInfo) == 0 {
		p.logger.Info("从midclt获取池信息失败，尝试从zfs命令获取")

		// 如果失败，尝试从zfs命令获取
		poolInfo, err = p.GetPoolNameFromZFS(ctx)
		if err != nil || len(poolInfo) == 0 {
			p.logger.Error("无法获取存储池信息: %v", err)
			return make(map[string]string), err
		}
	}

	// 记录找到的池和磁盘数量
	pools := make(map[string]bool)
	for _, pool := range poolInfo {
		pools[pool] = true
	}

	poolNames := make([]string, 0, len(pools))
	for pool := range pools {
		poolNames = append(poolNames, pool)
	}

	p.logger.Info("找到存储池: %v", poolNames)
	p.logger.Info("找到%d个磁盘与池的关联", len(poolInfo))

	return poolInfo, nil
}

// GetPoolInfo 使用midclt获取池和磁盘之间的关系
func (p *PoolCollector) GetPoolInfo(ctx context.Context) (map[string]string, error) {
	p.logger.Info("获取磁盘池信息...")

	output, err := p.commandRunner.Run(ctx, "midclt call pool.query")
	if err != nil {
		p.logger.Error("获取池信息失败: %v", err)
		return nil, err
	}

	var poolsData []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &poolsData); err != nil {
		p.logger.Error("解析池信息失败: %v", err)
		return nil, err
	}

	diskToPool := make(map[string]string)

	// 记录找到的池数量
	p.logger.Debug("获取到%d个存储池", len(poolsData))

	// 处理每个池
	for _, pool := range poolsData {
		poolName, ok := pool["name"].(string)
		if !ok || poolName == "" {
			continue
		}

		p.logger.Debug("处理存储池: %s", poolName)

		// 获取拓扑信息
		topology, ok := pool["topology"].(map[string]interface{})
		if !ok {
			continue
		}

		// 记录拓扑类型
		topoTypes := make([]string, 0, len(topology))
		for k := range topology {
			topoTypes = append(topoTypes, k)
		}
		p.logger.Debug("存储池 %s 的拓扑类型: %v", poolName, topoTypes)

		// 处理不同类型的设备（data、cache、log等）
		for vdevTypeName, vdevs := range topology {
			vdevsList, ok := vdevs.([]interface{})
			if !ok || len(vdevsList) == 0 {
				continue
			}

			// 使用 vdevTypeName 记录当前处理的 vdev 类型
			p.logger.Debug("处理 %s 类型的 vdev", vdevTypeName)

			for _, vdev := range vdevsList {
				vdevMap, ok := vdev.(map[string]interface{})
				if !ok {
					continue
				}

				// 获取vdev类型
				vdevTypeInfo, _ := vdevMap["type"].(string)
				p.logger.Debug("处理vdev类型: %s", vdevTypeInfo)

				// 处理children，对于RAID和镜像配置
				if children, ok := vdevMap["children"].([]interface{}); ok && len(children) > 0 {
					for _, child := range children {
						childMap, ok := child.(map[string]interface{})
						if !ok {
							continue
						}

						// 获取实际磁盘设备名称
						if disk, ok := childMap["disk"].(string); ok && disk != "" {
							diskToPool[disk] = poolName
							p.logger.Debug("将磁盘 %s 分配到存储池 %s (来自children)", disk, poolName)
							continue
						}

						// 尝试从路径或设备获取
						var diskPath string
						if path, ok := childMap["path"].(string); ok && path != "" {
							diskPath = path
						} else if device, ok := childMap["device"].(string); ok && device != "" {
							diskPath = device
						}

						if diskPath != "" {
							// 提取磁盘名称
							diskName := filepath.Base(diskPath)
							// 移除分区号
							baseDiskName := regexp.MustCompile(`p?\d+$`).ReplaceAllString(diskName, "")
							diskToPool[baseDiskName] = poolName
							p.logger.Debug("将磁盘 %s 分配到存储池 %s (来自路径)", baseDiskName, poolName)
						}
					}
				}

				// 处理直接设备
				if disk, ok := vdevMap["disk"].(string); ok && disk != "" {
					diskToPool[disk] = poolName
					p.logger.Debug("将磁盘 %s 分配到存储池 %s (直接磁盘)", disk, poolName)
					continue
				}

				// 尝试从路径或设备获取
				var diskPath string
				if path, ok := vdevMap["path"].(string); ok && path != "" {
					diskPath = path
				} else if device, ok := vdevMap["device"].(string); ok && device != "" {
					diskPath = device
				}

				if diskPath != "" {
					diskName := filepath.Base(diskPath)
					// 移除分区号
					baseDiskName := regexp.MustCompile(`p?\d+$`).ReplaceAllString(diskName, "")
					diskToPool[baseDiskName] = poolName
					p.logger.Debug("将磁盘 %s 分配到存储池 %s (直接路径)", baseDiskName, poolName)
				}
			}
		}
	}

	return diskToPool, nil
}

// GetPoolNameFromZFS 从zfs命令获取磁盘到池的映射（备用方法）
func (p *PoolCollector) GetPoolNameFromZFS(ctx context.Context) (map[string]string, error) {
	p.logger.Info("尝试从zfs命令获取池信息")

	// 获取所有zpool的状态
	output, err := p.commandRunner.Run(ctx, "zpool status")
	if err != nil {
		return nil, fmt.Errorf("执行zpool status命令失败: %w", err)
	}

	// 解析输出
	diskToPool := make(map[string]string)
	var currentPool string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检查是否是池名称行
		if strings.HasPrefix(line, "pool:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentPool = strings.TrimSpace(parts[1])
				p.logger.Debug("找到存储池: %s", currentPool)
			}
			continue
		}

		// 跳过不相关的行
		if currentPool == "" ||
			strings.Contains(line, "state:") ||
			strings.Contains(line, "scan:") ||
			strings.Contains(line, "config:") ||
			line == "" ||
			strings.HasPrefix(line, "NAME") ||
			strings.HasPrefix(line, "errors:") {
			continue
		}

		// 检查该行是否包含磁盘信息
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}

		diskName := parts[0]
		
		// 跳过池名和RAID配置相关的行
		if diskName == currentPool || 
		   strings.HasPrefix(diskName, currentPool+"-") ||
		   contains([]string{"mirror", "mirror-", "raidz", "raidz1", "raidz2", "raidz3", "cache", "log", "spare"}, diskName) ||
		   strings.HasPrefix(diskName, "mirror-") ||
		   strings.HasPrefix(diskName, "raidz") {
			continue
		}

		// 有些zpool输出可能会在磁盘前加上路径
		if strings.Contains(diskName, "/") {
			diskName = filepath.Base(diskName)
		}

		diskToPool[diskName] = currentPool
		p.logger.Debug("将磁盘 %s 分配到存储池 %s (从zpool)", diskName, currentPool)
	}

	p.logger.Debug("从zpool status获取到%d个磁盘与池的关联", len(diskToPool))
	return diskToPool, nil
}

// contains 检查切片中是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
