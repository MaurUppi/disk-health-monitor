# 数据存储层开发计划

## 1. 概述

存储层是磁盘健康监控工具的关键组成部分，主要实现历史数据的持久化存储、加载与增量计算功能。这一层的稳定性将直接影响工具的数据分析能力和用户体验。

本文档详细规划了存储层 (`storage/history.go`) 的开发路线、接口设计、实现策略以及潜在问题的解决方案。

## 2. 核心功能需求

1. **历史数据序列化与反序列化**
   - 将磁盘数据转换为结构化JSON格式
   - 从JSON文件加载并解析历史数据
   - 支持版本兼容性和格式扩展

2. **增量数据计算**
   - 比较历史数据和当前数据
   - 计算读写字节增量
   - 处理单位转换和特殊情况（如设备重置）

3. **数据文件管理**
   - 安全的文件读写操作
   - 备份与轮转策略
   - 错误恢复机制
   - 存储完整性验证

## 3. 接口设计

```go
// HistoryStorage 定义历史数据存储接口
type HistoryStorage interface {
    // 保存磁盘数据到文件
    SaveDiskData(data map[string]map[string]string) error
    
    // 从文件加载磁盘数据，返回数据和时间戳
    LoadDiskData() (data map[string]map[string]string, timestamp string, err error)
    
    // 计算增量数据
    CalculateIncrements(oldData, newData map[string]string) map[string]string
    
    // 设置存储路径
    SetStoragePath(path string) error
    
    // 创建数据备份
    CreateBackup() error
    
    // 检查存储完整性
    VerifyIntegrity() (bool, error)
}
```

## 4. 数据结构设计

为确保数据格式的灵活性和向前兼容性，我们需要一个带有元数据的结构：

```go
// HistoryData 表示存储的历史数据结构
type HistoryData struct {
    Version   string                            `json:"version"`   // 数据格式版本号
    Timestamp string                            `json:"timestamp"` // 数据收集时间戳
    Disks     map[string]map[string]string      `json:"disks"`     // 磁盘数据映射
    Meta      map[string]interface{}            `json:"meta"`      // 元数据（可扩展字段）
}
```

## 5. 实现策略

### 5.1 历史数据存储

**关键实现点**：

1. **安全写入策略**
   - 使用临时文件写入，成功后原子重命名
   - 防止进程中断导致的数据损坏
   - 设置适当的文件权限

2. **版本管理**
   - 每次写入时添加版本号和时间戳
   - 允许向后兼容性检查

```go
func (s *historyStorage) SaveDiskData(data map[string]map[string]string) error {
    // 构建数据结构
    historyData := HistoryData{
        Version:   "1.0",
        Timestamp: time.Now().Format(time.RFC3339),
        Disks:     data,
        Meta:      map[string]interface{}{"generator": "disk-health-monitor"},
    }
    
    // 序列化为JSON
    jsonData, err := json.MarshalIndent(historyData, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to serialize data: %w", err)
    }
    
    // 使用临时文件安全写入
    tempFile := s.path + ".tmp"
    if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
        return fmt.Errorf("failed to write temporary file: %w", err)
    }
    
    // 原子重命名确保数据完整性
    if err := os.Rename(tempFile, s.path); err != nil {
        return fmt.Errorf("failed to rename temporary file: %w", err)
    }
    
    return nil
}
```

### 5.2 历史数据加载

**关键实现点**：

1. **健壮的错误处理**
   - 处理文件不存在的情况
   - 处理JSON解析错误
   - 自动尝试备份恢复

2. **版本兼容**
   - 检测并处理不同版本的数据格式
   - 提供数据迁移路径

```go
func (s *historyStorage) LoadDiskData() (map[string]map[string]string, string, error) {
    // 检查文件是否存在
    if _, err := os.Stat(s.path); os.IsNotExist(err) {
        // 文件不存在，返回空数据
        return make(map[string]map[string]string), "", nil
    }
    
    // 读取文件
    fileData, err := os.ReadFile(s.path)
    if err != nil {
        return nil, "", fmt.Errorf("failed to read history file: %w", err)
    }
    
    // 尝试解析JSON
    var historyData HistoryData
    if err := json.Unmarshal(fileData, &historyData); err != nil {
        // 解析失败，尝试恢复
        return s.attemptRecovery(err)
    }
    
    // 版本兼容性检查
    if historyData.Version != "1.0" {
        // 实现版本迁移逻辑
        historyData = s.migrateDataFormat(historyData)
    }
    
    return historyData.Disks, historyData.Timestamp, nil
}
```

### 5.3 增量数据计算

**关键实现点**：

1. **单位处理**
   - 处理不同单位（KB、MB、GB、TB）的转换
   - 确保一致的比较基础

2. **异常情况处理**
   - 处理设备重置（新值小于旧值）
   - 处理缺失数据

```go
func (s *historyStorage) CalculateIncrements(oldData, newData map[string]string) map[string]string {
    increments := make(map[string]string)
    
    // 处理读写数据增量
    for _, key := range []string{"Data_Read", "Data_Written"} {
        newValue, newExists := newData[key]
        oldValue, oldExists := oldData[key]
        
        // 检查两个数据点是否都存在
        if !newExists || !oldExists {
            increments[key+"_Increment"] = "N/A"
            continue
        }
        
        // 解析为字节数
        newBytes, newErr := parseStorageSizeToBytes(newValue)
        oldBytes, oldErr := parseStorageSizeToBytes(oldValue)
        
        // 检查解析错误
        if newErr != nil || oldErr != nil {
            increments[key+"_Increment"] = "解析错误"
            continue
        }
        
        // 计算差值
        if newBytes >= oldBytes {
            // 正常情况 - 新值大于等于旧值
            diffBytes := newBytes - oldBytes
            increments[key+"_Increment"] = formatBytes(diffBytes)
        } else {
            // 可能是设备重置或更换
            increments[key+"_Increment"] = "重置"
        }
    }
    
    return increments
}
```

### 5.4 备份管理

**关键实现点**：

1. **备份策略**
   - 实现时间戳命名的备份
   - 自动轮转旧备份

2. **存储完整性**
   - 验证备份的完整性
   - 提供恢复选项

```go
func (s *historyStorage) CreateBackup() error {
    // 检查源文件是否存在
    if _, err := os.Stat(s.path); os.IsNotExist(err) {
        return nil // 无需备份
    }
    
    // 创建带时间戳的备份文件名
    timestamp := time.Now().Format("20060102150405")
    backupPath := fmt.Sprintf("%s.%s.bak", s.path, timestamp)
    
    // 复制文件内容
    input, err := os.ReadFile(s.path)
    if err != nil {
        return fmt.Errorf("failed to read source file: %w", err)
    }
    
    // 写入备份文件
    if err := os.WriteFile(backupPath, input, 0644); err != nil {
        return fmt.Errorf("failed to write backup file: %w", err)
    }
    
    // 删除旧备份（保留最近5个）
    return s.rotateBackups(5)
}
```

## 6. 异常处理策略

### 6.1 数据损坏处理

当发现数据文件损坏时，实现以下恢复流程：

1. 查找最近的备份文件
2. 尝试从备份中恢复数据
3. 如果没有可用备份，初始化新的数据存储
4. 记录详细的错误信息，帮助诊断问题

```go
func (s *historyStorage) attemptRecovery(originalErr error) (map[string]map[string]string, string, error) {
    logger.Warning("数据文件损坏，尝试从备份恢复: %v", originalErr)
    
    // 查找备份文件
    pattern := s.path + ".*.bak"
    backups, err := filepath.Glob(pattern)
    if err != nil || len(backups) == 0 {
        return make(map[string]map[string]string), "", fmt.Errorf("无法恢复: %w", originalErr)
    }
    
    // 按名称排序（时间戳从新到旧）
    sort.Sort(sort.Reverse(sort.StringSlice(backups)))
    
    // 尝试加载最新的备份
    for _, backup := range backups {
        data, timestamp, err := s.loadFromFile(backup)
        if err == nil {
            logger.Info("成功从备份恢复数据: %s", backup)
            
            // 恢复成功，更新主文件
            if data, err := os.ReadFile(backup); err == nil {
                _ = os.WriteFile(s.path, data, 0644)
            }
            
            return data, timestamp, nil
        }
    }
    
    // 所有备份都无法恢复
    return make(map[string]map[string]string), "", fmt.Errorf("所有备份恢复失败: %w", originalErr)
}
```

### 6.2 数据迁移

处理不同版本的数据格式：

```go
func (s *historyStorage) migrateDataFormat(oldData HistoryData) HistoryData {
    // 根据版本号实现迁移策略
    switch oldData.Version {
    case "0.1":
        // 从0.1版本升级到1.0版本
        return migrateFrom01To10(oldData)
    case "0.2":
        // 从0.2版本升级到1.0版本
        return migrateFrom02To10(oldData)
    default:
        // 未知版本，尝试兼容处理
        logger.Warning("未知的数据版本: %s，尝试兼容处理", oldData.Version)
        return oldData
    }
}
```

## 7. 单元测试计划

为确保存储层的可靠性，我们需要编写全面的单元测试，包括：

1. **基本功能测试**
   - 测试数据保存和加载
   - 测试增量计算逻辑
   - 测试备份和恢复

2. **边缘情况测试**
   - 空数据处理
   - 不兼容版本处理
   - 大文件处理

3. **错误处理测试**
   - 文件权限错误
   - 损坏的JSON数据
   - 文件不存在的情况

## 8. 性能优化考虑

### 8.1 文件大小优化

对于大型系统，历史数据文件可能会变得很大，我们可以：

1. **实现数据压缩**
   - 使用gzip压缩存储文件
   - 只存储关键字段，忽略不重要的信息

2. **增量更新**
   - 只更新有变化的部分，而不是整个文件

### 8.2 内存优化

处理大文件时的内存使用优化：

1. **流式解析**
   - 用流式JSON解析取代一次性加载
   - 减少大文件的内存占用

2. **懒加载**
   - 只在需要时才加载完整数据
   - 提供部分数据加载选项

## 9. 安全考虑

### 9.1 数据完整性

确保数据不被意外损坏：

1. **校验和验证**
   - 存储数据的校验和
   - 验证数据完整性

2. **原子操作**
   - 使用临时文件和重命名确保原子性
   - 防止部分写入造成的损坏

### 9.2 权限管理

确保适当的文件权限：

1. **最小权限原则**
   - 使用适当的文件权限（推荐0644）
   - 确保目录访问权限

## 10. 潜在挑战与解决方案

### 10.1 大数据挑战

**挑战**：大型系统可能有数百个磁盘，导致数据文件很大。

**解决方案**：
- 实现数据压缩
- 分片存储，按日期或磁盘组分隔文件
- 流式处理而非一次性加载

### 10.2 版本兼容性

**挑战**：随着应用的发展，数据格式可能会变化。

**解决方案**：
- 清晰的版本标记机制
- 数据迁移路径
- 向后兼容处理

### 10.3 文件系统限制

**挑战**：不同操作系统的文件系统行为可能不同。

**解决方案**：
- 跨平台测试
- 明确的错误处理
- 备用存储策略

## 11. 开发时间表

1. **基础架构（1天）**
   - 设计接口和数据结构
   - 实现基本的文件IO功能
   - 创建测试框架

2. **核心功能（1-2天）**
   - 实现数据序列化/反序列化
   - 实现增量计算
   - 编写单元测试

3. **异常处理（1天）**
   - 实现备份和恢复机制
   - 添加错误处理和日志
   - 测试异常情况

4. **优化和集成（1天）**
   - 性能优化
   - 集成测试
   - 文档完善

## 12. 结论

存储层的稳定性对整个磁盘健康监控工具至关重要。通过精心设计的接口、健壮的实现策略和全面的测试，我们可以确保这一关键组件的可靠性和性能。重点关注数据完整性、恢复机制和向前兼容性，将为应用提供坚实的数据基础。

完成存储层开发后，我们将拥有一个强大的历史数据管理系统，为后续的分析和报告功能提供重要支持。
