package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

func TestPoolCollector_GetPoolInfo(t *testing.T) {
	// 创建模拟命令执行器
	mockRunner := system.NewMockCommandRunner()
	mockLogger := system.NewMockLogger()
	config := model.NewDefaultConfig()

	// 模拟midclt输出 - 简化的JSON格式
	mockRunner.SetMockOutput("midclt call pool.query", `[
		{
			"name": "tank",
			"topology": {
				"data": [
					{
						"type": "mirror",
						"children": [
							{
								"disk": "sda"
							},
							{
								"disk": "sdb"
							}
						]
					}
				],
				"cache": [
					{
						"disk": "sdc"
					}
				]
			}
		},
		{
			"name": "backup",
			"topology": {
				"data": [
					{
						"path": "/dev/sdd"
					},
					{
						"device": "/dev/sde"
					}
				]
			}
		}
	]`)

	collector := NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err := collector.GetPoolInfo(context.Background())

	// 验证没有错误
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证找到的磁盘池映射
	expected := map[string]string{
		"sda": "tank",
		"sdb": "tank",
		"sdc": "tank",
		"sdd": "backup",
		"sde": "backup",
	}

	if len(poolInfo) != len(expected) {
		t.Errorf("Expected %d disk-pool mappings, got %d", len(expected), len(poolInfo))
	}

	for disk, pool := range expected {
		if poolInfo[disk] != pool {
			t.Errorf("Expected disk %s to be in pool %s, got %s", disk, pool, poolInfo[disk])
		}
	}

	// 测试midclt失败的情况
	mockRunner = system.NewMockCommandRunner()
	mockRunner.SetMockError("midclt call pool.query", errors.New("command failed"))

	collector = NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err = collector.GetPoolInfo(context.Background())

	// 验证返回错误
	if err == nil {
		t.Error("Expected error when midclt fails")
	}

	if len(poolInfo) != 0 {
		t.Errorf("Expected empty pool info when midclt fails, got %d mappings", len(poolInfo))
	}
}

func TestPoolCollector_GetPoolNameFromZFS(t *testing.T) {
	// 创建模拟命令执行器
	mockRunner := system.NewMockCommandRunner()
	mockLogger := system.NewMockLogger()
	config := model.NewDefaultConfig()

	// 模拟zpool status输出 - 精确匹配预期的5个磁盘映射
	mockRunner.SetMockOutput("zpool status", `  pool: tank
 state: ONLINE
  scan: scrub repaired 0B in 01:30:53 with 0 errors on Sat Oct 21 03:44:53 2023
config:

	NAME                        STATE     READ WRITE CKSUM
	tank                        ONLINE       0     0     0
	  mirror-0                  ONLINE       0     0     0
	    sda                     ONLINE       0     0     0
	    sdb                     ONLINE       0     0     0
	cache
	  sdc                       ONLINE       0     0     0

errors: No known data errors

  pool: backup
 state: ONLINE
  scan: scrub repaired 0B in 00:45:21 with 0 errors on Sat Oct 21 04:30:12 2023
config:

	NAME                        STATE     READ WRITE CKSUM
	backup                      ONLINE       0     0     0
	  sdd                       ONLINE       0     0     0
	  sde                       ONLINE       0     0     0

errors: No known data errors
`)

	collector := NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err := collector.GetPoolNameFromZFS(context.Background())

	// 验证没有错误
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证找到的磁盘池映射
	expected := map[string]string{
		"sda": "tank",
		"sdb": "tank",
		"sdc": "tank",
		"sdd": "backup",
		"sde": "backup",
	}

	if len(poolInfo) != len(expected) {
		t.Errorf("Expected %d disk-pool mappings, got %d", len(expected), len(poolInfo))
		
		// 打印实际找到的映射，以帮助调试
		t.Logf("Actual mappings:")
		for disk, pool := range poolInfo {
			t.Logf("  %s: %s", disk, pool)
		}
	}

	for disk, pool := range expected {
		if poolInfo[disk] != pool {
			t.Errorf("Expected disk %s to be in pool %s, got %s", disk, pool, poolInfo[disk])
		}
	}

	// 测试zpool命令失败的情况
	mockRunner = system.NewMockCommandRunner()
	mockRunner.SetMockError("zpool status", errors.New("command failed"))

	collector = NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err = collector.GetPoolNameFromZFS(context.Background())

	// 验证返回错误
	if err == nil {
		t.Error("Expected error when zpool command fails")
	}

	if poolInfo != nil && len(poolInfo) != 0 {
		t.Errorf("Expected nil or empty pool info when zpool fails, got %d mappings", len(poolInfo))
	}
}

func TestPoolCollector_Collect(t *testing.T) {
	// 创建模拟命令执行器
	mockRunner := system.NewMockCommandRunner()
	mockLogger := system.NewMockLogger()
	config := model.NewDefaultConfig()

	// 模拟成功的midclt调用
	mockRunner.SetMockOutput("midclt call pool.query", `[
		{
			"name": "tank",
			"topology": {
				"data": [
					{
						"type": "mirror",
						"children": [
							{
								"disk": "sda"
							},
							{
								"disk": "sdb"
							}
						]
					}
				]
			}
		}
	]`)

	collector := NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err := collector.Collect(context.Background())

	// 验证没有错误
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证找到了存储池信息
	if len(poolInfo) == 0 {
		t.Error("Expected to find pool information")
	}

	// 验证找到了正确的映射
	if poolInfo["sda"] != "tank" || poolInfo["sdb"] != "tank" {
		t.Errorf("Expected correct pool mapping, got %v", poolInfo)
	}

	// 测试midclt失败但zpool成功的情况
	mockRunner = system.NewMockCommandRunner()
	mockRunner.SetMockError("midclt call pool.query", errors.New("midclt failed"))
	mockRunner.SetMockOutput("zpool status", `  pool: backup
 state: ONLINE
config:

	NAME                        STATE     READ WRITE CKSUM
	backup                      ONLINE       0     0     0
	  sdc                       ONLINE       0     0     0
	  sdd                       ONLINE       0     0     0

errors: No known data errors
`)

	collector = NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err = collector.Collect(context.Background())

	// 验证没有错误（因为zpool成功）
	if err != nil {
		t.Errorf("Expected no error when midclt fails but zpool succeeds, got %v", err)
	}

	// 验证找到了存储池信息
	if len(poolInfo) == 0 {
		t.Error("Expected to find pool information from zpool")
	}

	// 验证找到了正确的映射
	if poolInfo["sdc"] != "backup" || poolInfo["sdd"] != "backup" {
		t.Errorf("Expected correct pool mapping from zpool, got %v", poolInfo)
	}

	// 测试都失败的情况
	mockRunner = system.NewMockCommandRunner()
	mockRunner.SetMockError("midclt call pool.query", errors.New("midclt failed"))
	mockRunner.SetMockError("zpool status", errors.New("zpool failed"))

	collector = NewPoolCollector(config, mockLogger, mockRunner)
	poolInfo, err = collector.Collect(context.Background())

	// 验证返回错误
	if err == nil {
		t.Error("Expected error when both midclt and zpool fail")
	}

	// 验证返回空映射
	if len(poolInfo) != 0 {
		t.Errorf("Expected empty mapping when both commands fail, got %d entries", len(poolInfo))
	}
}
