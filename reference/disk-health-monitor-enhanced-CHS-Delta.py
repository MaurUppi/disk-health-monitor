#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
# TrueNAS磁盘健康监控工具
# 以表格方式展示所有磁盘的SMART健康状态信息
#

import json
import subprocess
import argparse
import os
import sys
import glob
from datetime import datetime
import re
from collections import defaultdict

# 尝试导入美化表格的库，如果不存在则使用简单格式化
try:
    from prettytable import PrettyTable
    HAS_PRETTYTABLE = True
except ImportError:
    HAS_PRETTYTABLE = False
    print("提示: 安装 prettytable 可以获得更好的表格显示效果 (pip install prettytable)")

# 配置
DEBUG = False  # 设置为True启用调试输出
VERBOSE = False  # 设置为True显示信息日志
LOG_FILE = "/var/log/disk_health_monitor.log"
DATA_FILE = "/var/log/disk_health_monitor_data.json"  # 保存上次运行的数据

# 设备类型分类
DEVICE_TYPES = {
    "SAS_SSD": "SAS/SATA 固态硬盘",
    "SAS_HDD": "SAS/SATA 机械硬盘",
    "NVME_SSD": "NVMe 固态硬盘",
    "VIRTUAL": "虚拟设备"
}

# 磁盘类型特定的属性，格式为（属性名，显示名称，单位）
DISK_TYPE_ATTRIBUTES = {
    "SAS_SSD": [
        ("Temperature", "温度", "°C"),
        ("Trip_Temperature", "警告温度", "°C"),
        ("Power_On_Hours", "通电时间", "小时"),
        ("Power_Cycles", "通电周期", "次"),
        ("Percentage_Used", "已用寿命", "%"),
        ("Smart_Status", "SMART状态", ""),
        ("Data_Read", "已读数据", ""),
        ("Data_Written", "已写数据", ""),
        ("Non_Medium_Errors", "非介质错误", "个"),
        ("Uncorrected_Errors", "未修正错误", "个")
    ],
    "SAS_HDD": [
        ("Temperature", "温度", "°C"),
        ("Trip_Temperature", "警告温度", "°C"),
        ("Power_On_Hours", "通电时间", "小时"),
        ("Power_Cycles", "通电周期", "次"),
        ("Smart_Status", "SMART状态", ""),
        ("Data_Read", "已读数据", ""),
        ("Data_Written", "已写数据", ""),
        ("Non_Medium_Errors", "非介质错误", "个"),
        ("Uncorrected_Errors", "未修正错误", "个")
    ],
    "NVME_SSD": [
        ("Temperature", "温度", "°C"),
        ("Warning_Temperature", "警告温度", "°C"),
        ("Critical_Temperature", "临界温度", "°C"),
        ("Power_On_Hours", "通电时间", "小时"),
        ("Power_Cycles", "通电周期", "次"),
        ("Percentage_Used", "已用寿命", "%"),
        ("Available_Spare", "可用备件", "%"),
        ("Smart_Status", "SMART状态", ""),
        ("Data_Read", "已读数据", ""),
        ("Data_Written", "已写数据", "")
    ],
    "VIRTUAL": [
        ("Type", "设备类型", ""),
    ]
}

# 日志函数
def log_debug(message):
    """调试日志函数"""
    if DEBUG:
        print(f"[DEBUG] {message}")

def log_info(message):
    """信息日志函数"""
    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    log_message = f"[INFO] {timestamp} - {message}"
    
    # 只在启用verbose模式时打印到控制台
    if VERBOSE:
        print(log_message)
    
    # 尝试写入日志文件
    try:
        with open(LOG_FILE, 'a') as f:
            f.write(log_message + "\n")
    except Exception as e:
        if DEBUG:
            print(f"[DEBUG] 无法写入日志文件: {e}")

def log_error(message):
    """错误日志函数"""
    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    log_message = f"[ERROR] {timestamp} - {message}"
    print(log_message, file=sys.stderr)
    
    # 尝试写入日志文件
    try:
        with open(LOG_FILE, 'a') as f:
            f.write(log_message + "\n")
    except Exception as e:
        if DEBUG:
            print(f"[DEBUG] 无法写入日志文件: {e}")

def format_hours(hours_str):
    """将小时数转换为年、月、日、小时的格式"""
    try:
        hours = int(hours_str)  # 将输入转换为整数
        years, hours = divmod(hours, 365 * 24)  # 计算年份和剩余小时
        months, hours = divmod(hours, 30 * 24)  # 计算月份和剩余小时
        days, hours = divmod(hours, 24)  # 计算天数和剩余小时
        
        result = []
        if years > 0:
            result.append(f"{years}y")  # 添加年份
        if months > 0:
            result.append(f"{months}m")  # 添加月份
        if days > 0:
            result.append(f"{days}d")  # 添加天数
        if hours > 0 or not result:  # 如果有剩余小时或结果为空，添加小时
            result.append(f"{hours}hrs")
        
        return " ".join(result)  # 用空格连接各部分
    except (ValueError, TypeError):
        return hours_str  # 如果转换失败，返回原始值
        
def run_command(command, ignore_errors=False):
    """执行命令并返回输出"""
    try:
        log_debug(f"执行命令: {command}")
        result = subprocess.run(command, shell=True, capture_output=True, text=True, check=not ignore_errors)
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        if not ignore_errors:
            log_error(f"命令执行失败: {e}, 错误输出: {e.stderr}")
        return None

def find_storcli_path():
    """查找storcli工具的路径"""
    # 常见的storcli路径
    storcli_paths = [
        "storcli64",
        "storcli",
        "/opt/MegaRAID/storcli/storcli64",
        "/usr/local/sbin/storcli64",
        "/usr/sbin/storcli64",
        "/sbin/storcli64",
        "/usr/local/bin/storcli64",
        "/usr/bin/storcli64",
        "/bin/storcli64"
    ]
    
    for path in storcli_paths:
        if run_command(f"command -v {path} >/dev/null 2>&1 && echo 'exists'", ignore_errors=True) == "exists":
            return path
    
    return None

def get_lsi_controller_info():
    """获取LSI控制器信息，包括温度"""
    controller_info = {}
    
    storcli_path = find_storcli_path()
    if storcli_path:
        log_info(f"找到storcli工具: {storcli_path}，获取LSI控制器信息")
        
        # 先获取控制器列表
        controllers_output = run_command(f"{storcli_path} show", ignore_errors=True)
        if controllers_output:
            # 提取控制器ID
            controller_ids = []
            for line in controllers_output.splitlines():
                if "Controller = " in line:
                    controller_id = line.split("=")[1].strip()
                    controller_ids.append(controller_id)
                    log_debug(f"发现控制器ID: {controller_id}")
            
            if not controller_ids:
                # 如果没有找到明确的控制器ID，假设存在controller 0
                controller_ids = ["0"]
                log_debug("未找到明确的控制器ID，尝试使用默认ID 0")
            
            # 处理每个控制器
            for controller_id in controller_ids:
                # 获取控制器基本信息
                controller_output = run_command(f"{storcli_path} /c{controller_id} show", ignore_errors=True)
                
                if controller_output:
                    log_debug(f"控制器{controller_id}信息输出: {controller_output[:200]}...")
                    
                    controller_info[f"LSI_Controller_{controller_id}"] = {
                        "Type": "LSI SAS HBA",
                        "Status": "正常"
                    }
                    
                    # 提取关键信息
                    for line in controller_output.splitlines():
                        # 提取产品名称
                        if "Product Name" in line and "=" in line:
                            model = line.split("=")[1].strip()
                            controller_info[f"LSI_Controller_{controller_id}"]["Model"] = model
                            log_debug(f"找到控制器型号: {model}")
                        
                        # 提取固件版本 - 使用FW Version字段（注意冒号和等号）
                        elif "FW Version" in line and "=" in line:
                            if "FirmwareVersion" not in controller_info[f"LSI_Controller_{controller_id}"]:
                                fw_version = line.split("=")[1].strip()
                                controller_info[f"LSI_Controller_{controller_id}"]["FirmwareVersion"] = fw_version
                                log_debug(f"找到固件版本: {fw_version}")
                        
                        # 提取驱动版本
                        elif "Driver Version" in line and "=" in line:
                            driver_version = line.split("=")[1].strip()
                            controller_info[f"LSI_Controller_{controller_id}"]["DriverVersion"] = driver_version
                            log_debug(f"找到驱动版本: {driver_version}")
                        
                        # 提取设备数量
                        elif "Physical Drives" in line and "=" in line:
                            device_count = line.split("=")[1].strip()
                            controller_info[f"LSI_Controller_{controller_id}"]["DeviceCount"] = device_count
                            log_debug(f"找到物理设备数量: {device_count}")
                    
                    # 提取PD LIST部分以获取更多磁盘信息
                    pd_list_section = False
                    pd_list_info = []
                    ssd_count = 0
                    hdd_count = 0
                    
                    for line in controller_output.splitlines():
                        if "PD LIST :" in line:
                            pd_list_section = True
                            continue
                        
                        if pd_list_section and line.strip() and "---" not in line and "EID:Slt" not in line:
                            pd_list_info.append(line.strip())
                            # 计算SSD和HDD数量
                            if "SSD" in line:
                                ssd_count += 1
                            elif "HDD" in line:
                                hdd_count += 1
                    
                    # 保存设备类型统计
                    if ssd_count > 0:
                        controller_info[f"LSI_Controller_{controller_id}"]["SSDCount"] = str(ssd_count)
                    if hdd_count > 0:
                        controller_info[f"LSI_Controller_{controller_id}"]["HDDCount"] = str(hdd_count)
                    
                    # 获取温度信息 - 使用专门的温度命令
                    temp_output = run_command(f"{storcli_path} /c{controller_id} show temperature", ignore_errors=True)
                    if temp_output:
                        log_debug(f"控制器{controller_id}温度输出: {temp_output}")
                        # 匹配ROC temperature(Degree Celsius)后面的数字
                        temp_match = re.search(r"ROC temperature\(Degree Celsius\)\s+(\d+)", temp_output)
                        if temp_match:
                            temperature = temp_match.group(1)
                            controller_info[f"LSI_Controller_{controller_id}"]["ROCTemperatureDegreeCelsius"] = temperature
                            log_debug(f"找到控制器{controller_id}温度: {temperature}°C")
        
        # 如果找到了控制器信息，直接返回
        if controller_info:
            return controller_info
    
    # 如果storcli未找到或失败，尝试其他方法
    log_debug("通过storcli获取LSI控制器信息失败，尝试备用方法")
    
    # 通过lspci识别控制器
    lspci_output = run_command("lspci | grep -i 'LSI\\|MegaRAID\\|SAS\\|RAID'", ignore_errors=True)
    if lspci_output:
        log_debug("通过lspci识别LSI控制器")
        for line in lspci_output.splitlines():
            if "MegaRAID" in line or "LSI" in line or "SAS" in line:
                bus_id = line.split()[0]
                model = re.search(r":\s(.+)$", line)
                model_name = model.group(1).strip() if model else "LSI SAS HBA"
                
                controller_info[f"LSI_Controller_{bus_id}"] = {
                    "Type": "LSI SAS HBA",
                    "Model": model_name,
                    "Bus": bus_id,
                    "Status": "正常",
                    "Source": "lspci"
                }
    
    return controller_info

def get_nvme_controller_info():
    """获取NVMe控制器信息，主要使用lspci"""
    controller_info = {}
    
    # 使用lspci识别NVMe控制器
    lspci_exists = run_command("command -v lspci >/dev/null 2>&1 && echo 'exists'", ignore_errors=True)
    if lspci_exists and "exists" in lspci_exists:
        log_debug("使用lspci查找NVMe控制器")
        
        # 获取NVMe控制器
        nvme_controllers = run_command("lspci | grep -i 'nvme\\|non-volatile memory'", ignore_errors=True)
        if nvme_controllers:
            log_debug(f"通过lspci找到NVMe控制器: {nvme_controllers}")
            
            # 提取PCIe总线ID和控制器信息
            for line in nvme_controllers.splitlines():
                parts = line.split(" ", 1)
                if len(parts) >= 2:
                    bus_id = parts[0]
                    description = parts[1]
                    
                    # 创建控制器信息项
                    controller_info[f"NVMe_Controller_{bus_id}"] = {
                        "Type": "PCIe NVMe控制器",
                        "Description": description,
                        "Bus": bus_id
                    }
                    
                    # 只尝试从hwmon获取温度 - 最简单可靠的方法
                    try:
                        # 格式化总线ID (替换 : 为 .)
                        sysfs_bus_id = bus_id.replace(":", ".")
                        temp_file = run_command(f"find /sys/bus/pci/devices/0000:{sysfs_bus_id}/hwmon*/temp1_input 2>/dev/null | head -1", ignore_errors=True)
                        if temp_file and os.path.exists(temp_file):
                            try:
                                with open(temp_file, 'r') as f:
                                    temp_value = int(f.read().strip()) / 1000  # 从毫度转换为度
                                    controller_info[f"NVMe_Controller_{bus_id}"]["Temperature"] = str(int(temp_value))
                            except Exception as e:
                                log_debug(f"读取温度文件{temp_file}失败: {e}")
                    except Exception as e:
                        log_debug(f"访问hwmon温度信息失败: {e}")
    
    return controller_info

def get_disks_from_midclt():
    """使用midclt获取磁盘列表"""
    try:
        output = run_command("midclt call disk.query")
        if not output:
            log_error("midclt调用失败")
            return []
        
        return json.loads(output)
    except json.JSONDecodeError as e:
        log_error(f"解析midclt输出失败: {e}")
        return []
    except Exception as e:
        log_error(f"获取磁盘列表失败: {e}")
        return []

def get_pool_info():
    """获取池和磁盘之间的关系"""
    try:
        # 获取池配置
        output = run_command("midclt call pool.query")
        if not output:
            log_error("获取池信息失败")
            return {}
        
        pools_data = json.loads(output)
        disk_to_pool = {}
        
        if DEBUG:
            log_debug(f"获取到{len(pools_data)}个存储池")
            for pool in pools_data:
                log_debug(f"存储池: {pool.get('name', 'Unknown')}")
        
        # 处理每个池
        for pool in pools_data:
            pool_name = pool.get("name", "")
            if not pool_name:
                continue
                
            # 获取拓扑信息
            topology = pool.get("topology", {})
            if DEBUG:
                log_debug(f"存储池 {pool_name} 的拓扑类型: {list(topology.keys())}")
            
            # 处理不同类型的设备（data、cache、log等）
            for vdev_type, vdevs in topology.items():
                if not vdevs:
                    continue
                
                for vdev in vdevs:
                    # 获取vdev类型
                    vdev_type_name = vdev.get("type", "")
                    
                    # 处理children，对于RAID和镜像配置
                    if "children" in vdev and vdev["children"]:
                        for child in vdev["children"]:
                            # 获取实际磁盘设备名称
                            device_name = child.get("disk", "")
                            if device_name:
                                disk_to_pool[device_name] = pool_name
                                if DEBUG:
                                    log_debug(f"将磁盘 {device_name} 分配到存储池 {pool_name} (来自children)")
                                continue
                                
                            # 尝试从路径或设备获取
                            disk_path = child.get("path", "")
                            if not disk_path:
                                disk_path = child.get("device", "")
                            
                            if disk_path:
                                # 提取磁盘名称
                                if "/" in disk_path:
                                    disk_name = disk_path.split("/")[-1]
                                else:
                                    disk_name = disk_path
                                
                                # 移除分区号
                                base_disk_name = re.sub(r'p?\d+$', '', disk_name)
                                disk_to_pool[base_disk_name] = pool_name
                                if DEBUG:
                                    log_debug(f"将磁盘 {base_disk_name} 分配到存储池 {pool_name} (来自children路径)")
                    
                    # 处理直接设备
                    device_name = vdev.get("disk", "")
                    if device_name:
                        disk_to_pool[device_name] = pool_name
                        if DEBUG:
                            log_debug(f"将磁盘 {device_name} 分配到存储池 {pool_name} (直接磁盘)")
                        continue
                        
                    # 尝试从路径或设备获取    
                    disk_path = vdev.get("path", "")
                    if not disk_path:
                        disk_path = vdev.get("device", "")
                        
                    if disk_path:
                        if "/" in disk_path:
                            disk_name = disk_path.split("/")[-1]
                        else:
                            disk_name = disk_path
                            
                        # 移除分区号
                        base_disk_name = re.sub(r'p?\d+$', '', disk_name)
                        disk_to_pool[base_disk_name] = pool_name
                        if DEBUG:
                            log_debug(f"将磁盘 {base_disk_name} 分配到存储池 {pool_name} (直接设备路径)")
        
        # 检查找到的磁盘池
        if DEBUG:
            log_debug(f"找到{len(disk_to_pool)}个磁盘与池的关联: {disk_to_pool}")
            
        return disk_to_pool
    except Exception as e:
        log_error(f"获取池信息失败: {e}")
        return {}

def get_pool_name_from_zfs():
    """从zfs命令获取磁盘到池的映射（备用方法）"""
    try:
        # 获取所有zpool的状态
        output = run_command("zpool status")
        if not output:
            return {}
        
        # 解析输出
        disk_to_pool = {}
        current_pool = None
        
        lines = output.split('\n')
        for line in lines:
            line = line.strip()
            
            # 检查是否是池名称行
            if line.startswith('pool:'):
                current_pool = line.split(':', 1)[1].strip()
                continue
            
            # 跳过不相关的行
            if not current_pool or 'state:' in line or 'scan:' in line or 'config:' in line or not line or line.startswith('NAME'):
                continue
            
            # 检查该行是否包含磁盘信息
            parts = line.split()
            if len(parts) >= 1 and not parts[0].startswith(current_pool) and not parts[0] in ('mirror', 'raidz1', 'raidz2', 'raidz3'):
                disk_name = parts[0]
                
                # 有些zpool输出可能会在磁盘前加上路径
                if '/' in disk_name:
                    disk_name = disk_name.split('/')[-1]
                
                disk_to_pool[disk_name] = current_pool
                
        if DEBUG:
            log_debug(f"从zpool status获取到{len(disk_to_pool)}个磁盘与池的关联")
            
        return disk_to_pool
    except Exception as e:
        log_error(f"从zfs获取池信息失败: {e}")
        return {}

def get_disks_from_lsblk():
    """使用lsblk获取磁盘列表（备用方法）"""
    try:
        output = run_command("lsblk -d -o NAME,TYPE,MODEL,SIZE -n | grep 'disk'")
        if not output:
            return []
        
        disks = []
        for line in output.splitlines():
            parts = line.split()
            if len(parts) >= 3:
                name = parts[0]
                disk_type = "HDD"  # 默认为HDD
                if "nvme" in name.lower():
                    disk_type = "SSD"
                model = " ".join(parts[2:-1])
                size = parts[-1]
                
                disks.append({
                    "name": name,
                    "type": disk_type,
                    "model": model,
                    "size": size
                })
        return disks
    except Exception as e:
        log_error(f"使用lsblk获取磁盘列表失败: {e}")
        return []

def normalize_size_unit(size_str):
    """将数据大小标准化为合适的单位"""
    # 如果输入为None或空字符串，直接返回
    if not size_str:
        return "N/A"
    
    # 如果输入已经是格式化字符串（如 "16.0 TB"），解析并重新格式化
    if isinstance(size_str, str):
        match = re.match(r'(\d+\.?\d*)\s*([KMGTP]?B)', size_str)
        if match:
            value = float(match.group(1))
            unit = match.group(2).upper()
            
            # 根据单位将值转换为字节
            units = {'B': 1, 'KB': 1024, 'MB': 1024**2, 'GB': 1024**3, 'TB': 1024**4, 'PB': 1024**5}
            if unit in units:
                bytes_value = value * units[unit]
                return format_size(bytes_value)
    
    # 如果是纯数字，视为字节数
    try:
        return format_size(float(size_str))
    except (ValueError, TypeError):
        return str(size_str)

def format_size(size_bytes):
    """格式化容量大小"""
    try:
        size = float(size_bytes)
        units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
        unit_index = 0
        
        while size >= 1024 and unit_index < len(units) - 1:
            size /= 1024
            unit_index += 1
        
        # 根据大小值和单位进行优化显示
        if units[unit_index] == 'GB' and size >= 1000:
            size /= 1024
            unit_index += 1
            
        if units[unit_index] == 'MB' and size >= 1000:
            size /= 1024
            unit_index += 1
        
        if size < 10:
            return f"{size:.2f} {units[unit_index]}"
        elif size < 100:
            return f"{size:.1f} {units[unit_index]}"
        else:
            return f"{int(size)} {units[unit_index]}"
    except (ValueError, TypeError):
        return str(size_bytes)

def get_nvme_smart_data(disk_name):
    """获取NVMe磁盘的SMART数据"""
    smart_data = {}
    
    # 检查是否为VMware虚拟设备
    vendor_check = run_command(f"smartctl -i /dev/{disk_name} | grep 'PCI Vendor'", ignore_errors=True)
    if vendor_check and "0x15ad" in vendor_check:
        log_debug(f"{disk_name}是VMware虚拟设备，跳过详细SMART数据收集")
        smart_data["Smart_Status"] = "虚拟设备"
        smart_data["Type"] = "虚拟NVMe设备"
        return smart_data
    
    # 获取健康状态
    health_output = run_command(f"smartctl -H /dev/{disk_name}", ignore_errors=True)
    if health_output:
        smart_status = "未知"
        if "PASSED" in health_output:
            smart_status = "PASSED"
        elif "OK" in health_output:
            smart_status = "OK"
        elif "FAILED" in health_output:
            smart_status = "FAILED"
        smart_data["Smart_Status"] = smart_status
    
    # 获取SMART详情
    output = run_command(f"smartctl -a /dev/{disk_name}", ignore_errors=True)
    if not output:
        return smart_data
    
    # 提取温度
    temp_match = re.search(r"Temperature:\s+(\d+)\s+Celsius", output)
    if temp_match:
        temp = temp_match.group(1)
        # 检查是否可能是开氏度(>200通常是开氏度)
        if int(temp) > 200:
            temp_c = round(int(temp) - 273.15, 1)
            smart_data["Temperature"] = str(temp_c)
        else:
            smart_data["Temperature"] = temp
    
    # 提取警告温度和临界温度
    warning_temp_match = re.search(r"Warning\s+Comp\.\s+Temp\.\s+Threshold:\s+(\d+)\s+Celsius", output)
    if warning_temp_match:
        smart_data["Warning_Temperature"] = warning_temp_match.group(1)
    
    critical_temp_match = re.search(r"Critical\s+Comp\.\s+Temp\.\s+Threshold:\s+(\d+)\s+Celsius", output)
    if critical_temp_match:
        smart_data["Critical_Temperature"] = critical_temp_match.group(1)
    
    # 提取通电时间 - 改进匹配模式
    hours_patterns = [
        r"Power On Hours:\s+(\d+[,\d]*)",  # NVMe格式
        r"Power_On_Hours.*?(\d+)",  # SMART属性格式
        r"Accumulated power on time.*?(\d+)[:\s]", # SAS/SATA格式
        r"number of hours powered up\s*=\s*(\d+\.?\d*)" # Seagate特有格式
    ]
    
    for pattern in hours_patterns:
        hours_match = re.search(pattern, output, re.IGNORECASE)
        if hours_match:
            smart_data["Power_On_Hours"] = hours_match.group(1).replace(',', '')
            break
    
    # 提取其他关键指标
    patterns = {
        "Available_Spare": r"Available Spare:\s+(\d+)%",
        "Percentage_Used": r"Percentage Used:\s+(\d+)%",
        "Power_Cycles": r"Power Cycles:\s+(\d+[,\d]*)",
    }
    
    for key, pattern in patterns.items():
        match = re.search(pattern, output)
        if match:
            value = match.group(1)
            # 移除千位分隔符
            value = value.replace(',', '')
            smart_data[key] = value
    
    # 提取数据读写量
    read_match = re.search(r"Data Units Read:\s+(\d+[,\d]*)\s+\[([^\]]+)\]", output)
    if read_match:
        size_read = read_match.group(2).strip()
        # 标准化单位显示
        smart_data["Data_Read"] = normalize_size_unit(size_read)
    
    write_match = re.search(r"Data Units Written:\s+(\d+[,\d]*)\s+\[([^\]]+)\]", output)
    if write_match:
        size_written = write_match.group(2).strip()
        # 标准化单位显示
        smart_data["Data_Written"] = normalize_size_unit(size_written)
    
    return smart_data

def get_sata_smart_data(disk_name, disk_type):
    """获取SATA/SAS磁盘的SMART数据"""
    smart_data = {}
    
    # 确定是SSD还是HDD
    is_ssd = disk_type.upper() == "SSD"
    
    # 获取健康状态
    health_output = run_command(f"smartctl -H /dev/{disk_name}", ignore_errors=True)
    if health_output:
        smart_status = "未知"
        if "PASSED" in health_output:
            smart_status = "PASSED"
        elif "OK" in health_output:
            smart_status = "OK"
        elif "FAILED" in health_output:
            smart_status = "FAILED"
        smart_data["Smart_Status"] = smart_status
    
    # 检查是否存在"Percentage used endurance indicator"
    if health_output and "Percentage used endurance indicator:" in health_output:
        endurance_match = re.search(r"Percentage used endurance indicator:\s+(\d+)%", health_output)
        if endurance_match:
            smart_data["Percentage_Used"] = endurance_match.group(1)
    
    # 获取SMART详情
    output = run_command(f"smartctl -a /dev/{disk_name}", ignore_errors=True)
    if not output:
        return smart_data
    
    # 提取温度 - 尝试多种模式
    temp_patterns = [
        r"Current Drive Temperature:\s+(\d+)\s+C",
        r"Temperature:\s+(\d+)\s+Celsius",
        r"Temperature_Celsius.*?(\d+)",
        r"Temperature.*?(\d+)"
    ]
    
    for pattern in temp_patterns:
        temp_match = re.search(pattern, output)
        if temp_match:
            smart_data["Temperature"] = temp_match.group(1)
            break
    
    # 提取警告温度
    trip_temp_patterns = [
        r"Drive Trip Temperature:\s+(\d+)\s+C",
        r"Warning\s+Comp\.\s+Temp\.\s+Threshold:\s+(\d+)",
    ]
    
    for pattern in trip_temp_patterns:
        trip_match = re.search(pattern, output)
        if trip_match:
            smart_data["Trip_Temperature"] = trip_match.group(1)
            break
    
    # 获取通电时间 - 改进匹配模式
    hours_patterns = [
        r"Power On Hours:\s+(\d+[,\d]*)",  # 标准格式
        r"Power_On_Hours.*?(\d+)",  # SMART属性格式
        r"Accumulated power on time.*?(\d+)[:\s]", # SAS/SATA格式
        r"power on time.*?(\d+)\s+hours",  # 另一种格式
        r"number of hours powered up\s*=\s*(\d+\.?\d*)", # Seagate特有格式
        r"Accumulated power on time, hours:minutes (\d+):" # SAS特有格式
    ]
    
    for pattern in hours_patterns:
        hours_match = re.search(pattern, output, re.IGNORECASE)
        if hours_match:
            smart_data["Power_On_Hours"] = hours_match.group(1).replace(',', '')
            break
    
    # 获取通电周期 - 改进匹配模式
    cycles_patterns = [
        r"Power Cycles:\s+(\d+[,\d]*)",  # NVMe格式
        r"Power_Cycle_Count.*?(\d+)",  # SMART属性格式
        r"Accumulated start-stop cycles:\s+(\d+)",  # SAS格式
        r"start-stop cycles:\s+(\d+)",  # 另一种格式
        r"Power Cycle Count:\s+(\d+)",  # 标准格式
        r"Specified cycle count over device lifetime:\s+(\d+)" # 另一种Seagate特有格式
    ]
    
    for pattern in cycles_patterns:
        cycles_match = re.search(pattern, output, re.IGNORECASE)
        if cycles_match:
            smart_data["Power_Cycles"] = cycles_match.group(1).replace(',', '')
            break
    
    # 提取非介质错误数量
    non_medium_match = re.search(r"Non-medium error count:\s+(\d+)", output)
    if non_medium_match:
        smart_data["Non_Medium_Errors"] = non_medium_match.group(1)
    
    # 提取Error counter log中的读写信息和未修正错误
        # 首先尝试提取直接显示格式如 "[10^9 bytes]" 的数据
        error_log_section = re.search(r"Error counter log:.*?Gigabytes\s+processed.*?errors\s*\n\s*read:.*?\n\s*write:", output, re.DOTALL)
        if error_log_section:
            error_log_text = error_log_section.group(0)
            
            # 查找字节单位
            bytes_unit_match = re.search(r"\[(\d+)\^(\d+)\s+bytes\]", error_log_text)
            unit = "GB"  # 默认单位
            if bytes_unit_match:
                base = int(bytes_unit_match.group(1))
                exponent = int(bytes_unit_match.group(2))
                if base == 10 and exponent == 9:
                    unit = "GB"
                elif base == 10 and exponent == 12:
                    unit = "TB"
            
            # 尝试匹配读写数据
            read_match = re.search(r"read:.*?processed\s+\[[^\]]+\]\s+uncorrected\s*\n", error_log_text, re.IGNORECASE)
            if read_match:
                read_line = read_match.group(0)
                read_gb_match = re.search(r"(\d+\.\d+)", read_line.split('[')[0])
                if read_gb_match:
                    value = float(read_gb_match.group(1))
                    # 计算字节数并格式化为合适单位
                    if unit == "GB":
                        bytes_value = value * (1024**3)
                    elif unit == "TB":
                        bytes_value = value * (1024**4)
                    else:
                        bytes_value = value
                        
                    smart_data["Data_Read"] = normalize_size_unit(f"{value} {unit}")
            
            write_match = re.search(r"write:.*?processed\s+\[[^\]]+\]\s+uncorrected\s*\n", error_log_text, re.IGNORECASE)
            if write_match:
                write_line = write_match.group(0)
                write_gb_match = re.search(r"(\d+\.\d+)", write_line.split('[')[0])
                if write_gb_match:
                    value = float(write_gb_match.group(1))
                    # 计算字节数并格式化为合适单位
                    if unit == "GB":
                        bytes_value = value * (1024**3)
                    elif unit == "TB": 
                        bytes_value = value * (1024**4)
                    else:
                        bytes_value = value
                        
                    smart_data["Data_Written"] = normalize_size_unit(f"{value} {unit}")
        
        # 如果上面的方法失败，使用备用方法
        if "Data_Read" not in smart_data or "Data_Written" not in smart_data:
            # 直接匹配Gigabytes列（使用正确的表格对齐方式）
            error_log_lines = re.findall(r"(read|write):(?:\s+\d+){5}\s+(\d+\.\d+)", output)
            for io_type, size in error_log_lines:
                size_float = float(size)
                if io_type == "read" and "Data_Read" not in smart_data:
                    # 转换成字节数进行标准化处理
                    bytes_value = size_float * (1024**3)  # GB to bytes
                    smart_data["Data_Read"] = normalize_size_unit(str(bytes_value))
                elif io_type == "write" and "Data_Written" not in smart_data:
                    # 转换成字节数进行标准化处理
                    bytes_value = size_float * (1024**3)  # GB to bytes
                    smart_data["Data_Written"] = normalize_size_unit(str(bytes_value))
    
    # 查找未修正错误总数（从Error counter log）
    uncorrected_errors_match = re.search(r"errors\s*\n.*?(\d+)\s*$", output, re.MULTILINE)
    if uncorrected_errors_match:
        smart_data["Uncorrected_Errors"] = uncorrected_errors_match.group(1)
    
    # 提取Error counter log中的读写信息 - 对SSD和HDD都处理
    error_log_section = re.search(r"Error counter log:.*?Gigabytes\s+processed.*?errors\s*\n\s*read:.*?\n\s*write:.*?\n", output, re.DOTALL)
    if error_log_section:
        error_log_text = error_log_section.group(0)
        
        # 查找字节单位
        bytes_unit_match = re.search(r"\[(\d+)\^(\d+)\s+bytes\]", error_log_text)
        unit = "GB"  # 默认单位
        if bytes_unit_match:
            base = int(bytes_unit_match.group(1))
            exponent = int(bytes_unit_match.group(2))
            if base == 10 and exponent == 9:
                unit = "GB"
            elif base == 10 and exponent == 12:
                unit = "TB"
        
        # 提取读数据量
        read_match = re.search(r"read:.*?(\d+\.\d+)\s*$", error_log_text, re.MULTILINE)
        if read_match:
            value = float(read_match.group(1))
            if unit == "GB":
                bytes_value = value * (1024**3)
            elif unit == "TB":
                bytes_value = value * (1024**4)
            else:
                bytes_value = value
                
            smart_data["Data_Read"] = normalize_size_unit(f"{value} {unit}")
        
        # 提取写数据量
        write_match = re.search(r"write:.*?(\d+\.\d+)\s*$", error_log_text, re.MULTILINE)
        if write_match:
            value = float(write_match.group(1))
            if unit == "GB":
                bytes_value = value * (1024**3)
            elif unit == "TB": 
                bytes_value = value * (1024**4)
            else:
                bytes_value = value
                
            smart_data["Data_Written"] = normalize_size_unit(f"{value} {unit}")
    
    return smart_data

def create_simple_table(headers, rows):
    """创建简单的ASCII表格"""
    # 计算每列的最大宽度
    widths = [max(len(str(row[i])) for row in rows + [headers]) for i in range(len(headers))]
    
    # 创建表头
    header_line = " | ".join(f"{headers[i]:{widths[i]}}" for i in range(len(headers)))
    separator = "-+-".join("-" * width for width in widths)
    
    # 创建表格内容
    table = [header_line, separator]
    for row in rows:
        table.append(" | ".join(f"{str(row[i]):{widths[i]}}" for i in range(len(row))))
    
    return "\n".join(table)

def format_value(value, attribute):
    """Format attribute value"""
    if value is None:
        return "N/A"

    if attribute == "Power_On_Hours":
        return format_hours(value)  # 对通电时间进行转换
        
    return value

def classify_disk(disk_name, disk_type, disk_model):
    """将磁盘分类为SAS SSD、SAS HDD或NVMe SSD"""
    if "VMware" in disk_model or "Virtual" in disk_model:
        return "VIRTUAL"
    elif disk_name.startswith("nvme"):
        return "NVME_SSD"
    elif disk_type.upper() == "HDD":
        return "SAS_HDD"
    else:
        return "SAS_SSD"  # 默认SAS/SATA SSD

def load_previous_disk_data():
    """加载上次运行的磁盘数据"""
    try:
        if os.path.exists(DATA_FILE):
            with open(DATA_FILE, 'r') as f:
                data = json.load(f)
                log_debug(f"成功加载上次运行的磁盘数据: {DATA_FILE}")
                return data
    except Exception as e:
        log_error(f"加载上次运行的磁盘数据失败: {e}")
    
    return {"timestamp": "", "disks": {}}

def save_disk_data(disk_data):
    """保存当前磁盘数据，用于下次比较"""
    try:
        # 创建目录（如果不存在）
        os.makedirs(os.path.dirname(DATA_FILE), exist_ok=True)
        
        data = {
            "timestamp": datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            "disks": disk_data
        }
        
        with open(DATA_FILE, 'w') as f:
            json.dump(data, f, indent=2)
            
        log_debug(f"成功保存磁盘数据到: {DATA_FILE}")
    except Exception as e:
        log_error(f"保存磁盘数据失败: {e}")

def parse_size_to_bytes(size_str):
    """将大小字符串（如 '1.5 TB'）解析为字节数"""
    if not size_str or size_str == "N/A":
        return None
        
    try:
        # 匹配数字和单位
        match = re.match(r"(\d+\.?\d*)\s*([KMGTP]?B)", size_str)
        if not match:
            return None
            
        value = float(match.group(1))
        unit = match.group(2).upper()
        
        # 单位转换为字节
        multipliers = {
            'B': 1,
            'KB': 1024,
            'MB': 1024**2,
            'GB': 1024**3,
            'TB': 1024**4,
            'PB': 1024**5
        }
        
        if unit in multipliers:
            return value * multipliers[unit]
        return value
    except Exception as e:
        log_debug(f"解析大小字符串失败: {size_str}, 错误: {e}")
        return None

def calculate_size_increment(old_value, new_value):
    """计算两个大小字符串之间的增量"""
    old_bytes = parse_size_to_bytes(old_value)
    new_bytes = parse_size_to_bytes(new_value)
    
    if old_bytes is None or new_bytes is None:
        return "N/A"
        
    diff_bytes = new_bytes - old_bytes
    if diff_bytes < 0:
        # 如果是负值，可能是设备重置了计数器
        return "重置"
        
    return normalize_size_unit(str(diff_bytes))

def process_controller_info(args):
    """处理和显示控制器信息"""
    controller_tables = []
    
    if not args.no_controller:
        log_info("获取存储控制器信息...")
        
        # 获取LSI和NVMe控制器信息
        lsi_controllers = get_lsi_controller_info()
        nvme_controllers = get_nvme_controller_info()
        
        if lsi_controllers:
            log_info(f"找到{len(lsi_controllers)}个LSI控制器")
            # 创建LSI控制器表格
            title = "\n--- LSI SAS HBA控制器信息 ---\n"
            
            # 创建LSI控制器表格
            headers = ["控制器名称", "型号", "固件版本", "驱动版本", "温度", "设备数", "状态"]
            
            lsi_rows = []
            for controller_name, info in lsi_controllers.items():
                row = [
                    controller_name,
                    info.get("Model", "N/A"),
                    info.get("FirmwareVersion", "N/A"),
                    info.get("DriverVersion", "N/A"),
                    f"{info.get('ROCTemperatureDegreeCelsius', 'N/A')}°C" if info.get('ROCTemperatureDegreeCelsius') else "N/A",
                    info.get("DeviceCount", "N/A"),
                    info.get("Status", "未知")
                ]
                lsi_rows.append(row)
            
            if HAS_PRETTYTABLE:
                table = PrettyTable(headers)
                for row in lsi_rows:
                    table.add_row(row)
                controller_tables.append(title + table.get_string())
            else:
                controller_tables.append(title + create_simple_table(headers, lsi_rows))
        
        if nvme_controllers:
            log_info(f"找到{len(nvme_controllers)}个NVMe控制器")
            # 创建NVMe控制器表格
            title = "\n--- NVMe控制器信息 ---\n"
            
            # 创建NVMe控制器表格
            headers = ["总线ID", "控制器描述", "温度"]
            
            nvme_rows = []
            for controller_name, info in nvme_controllers.items():
                row = [
                    info.get("Bus", "N/A"),
                    info.get("Description", "N/A"),
                    f"{info.get('Temperature', 'N/A')}°C" if info.get('Temperature') else "N/A"
                ]
                nvme_rows.append(row)
            
            if HAS_PRETTYTABLE:
                table = PrettyTable(headers)
                for row in nvme_rows:
                    table.add_row(row)
                controller_tables.append(title + table.get_string())
            else:
                controller_tables.append(title + create_simple_table(headers, nvme_rows))
    
    # 显示控制器表格
    for table in controller_tables:
        print(table)
    
    return controller_tables

def process_disk_info(args, controller_tables):
    """处理和显示磁盘信息"""
    output_tables = []
    
    # 如果只显示控制器信息，直接返回
    if args.controller_only:
        if args.output:
            try:
                with open(args.output, 'w') as f:
                    # 写入控制器信息
                    for table in controller_tables:
                        f.write(table + "\n\n")
                log_info(f"控制器信息已保存到: {args.output}")
            except Exception as e:
                log_error(f"保存输出到文件失败: {e}")
        
        return []
    
    # 获取磁盘列表
    log_info("获取磁盘列表...")
    disks = get_disks_from_midclt()
    
    if not disks:
        log_info("从midclt获取磁盘列表失败，尝试使用lsblk")
        disks = get_disks_from_lsblk()
        if not disks:
            log_error("无法获取磁盘列表，退出")
            return []
    
    # 获取磁盘池信息
    log_info("获取磁盘池信息...")
    disk_to_pool = get_pool_info()
    
    # 如果从midclt获取失败，尝试从zfs命令获取
    if not disk_to_pool:
        log_info("从midclt获取池信息失败，尝试从zfs命令获取")
        disk_to_pool = get_pool_name_from_zfs()
    
    # 加载上次运行的磁盘数据
    log_info("加载上次运行的磁盘数据以计算增量...")
    previous_data = load_previous_disk_data()
    previous_disks = previous_data.get("disks", {})
    previous_time = previous_data.get("timestamp", "")
    
    if previous_time:
        log_info(f"上次运行时间: {previous_time}")
    else:
        log_info("未找到上次运行的数据，将只显示当前状态")
    
    # 如果启用了调试模式，显示找到的池信息
    if DEBUG:
        log_debug(f"找到的磁盘到池映射: {disk_to_pool}")
    else:
        # 即使在非调试模式，也简要显示池信息
        pools = set(disk_to_pool.values())
        if pools:
            log_info(f"找到的存储池: {', '.join(pools)}")
            mapped_disks = len(disk_to_pool)
            log_info(f"找到{mapped_disks}个磁盘与池的关联")
    
    # 按类型分组的磁盘数据
    grouped_disks = defaultdict(list)
    all_disk_data = []
    
    # 用于保存当前运行的磁盘数据
    current_disk_data = {}
    
    # 处理每个磁盘
    for disk in disks:
        disk_name = disk.get("name", "")
        disk_type = disk.get("type", "")
        disk_model = disk.get("model", "")
        disk_size = disk.get("size", 0)
        disk_pool = disk_to_pool.get(disk_name, "未分配")
        
        formatted_size = format_size(disk_size)
        
        log_info(f"处理磁盘: {disk_name} (类型: {disk_type}, 型号: {disk_model}, 容量: {formatted_size}, 池: {disk_pool})")
        
        # 确定设备类型分类
        device_class = classify_disk(disk_name, disk_type, disk_model)
        
        # 根据磁盘类型获取SMART数据
        smart_data = {}
        if device_class == "NVME_SSD":
            smart_data = get_nvme_smart_data(disk_name)
        elif device_class == "VIRTUAL":
            smart_data = {"Type": "虚拟设备", "Smart_Status": "N/A"}
        else:
            smart_data = get_sata_smart_data(disk_name, disk_type)
        
        # 获取该类型磁盘的属性列表
        attributes = DISK_TYPE_ATTRIBUTES.get(device_class, [])
        
        # 准备行数据
        row = [disk_name, disk_type, disk_model, formatted_size, disk_pool]
        
        # 保存当前磁盘数据用于下次比较
        current_disk_data[disk_name] = {}
        
        # 添加特定于设备类型的属性值
        for attr_name, _, _ in attributes:
            value = smart_data.get(attr_name, "N/A")
            formatted_value = format_value(value, attr_name)
            row.append(formatted_value)
            
            # 保存读写数据用于下次比较
            if attr_name in ["Data_Read", "Data_Written"]:
                current_disk_data[disk_name][attr_name] = formatted_value
        
        # 添加读写增量列
        prev_disk = previous_disks.get(disk_name, {})
        
        # 计算读增量
        read_increment = "N/A"
        if "Data_Read" in smart_data and "Data_Read" in prev_disk:
            read_increment = calculate_size_increment(prev_disk["Data_Read"], smart_data["Data_Read"])
        row.append(read_increment)
        
        # 计算写增量
        write_increment = "N/A"
        if "Data_Written" in smart_data and "Data_Written" in prev_disk:
            write_increment = calculate_size_increment(prev_disk["Data_Written"], smart_data["Data_Written"])
        row.append(write_increment)
        
        # 添加到总列表
        all_disk_data.append((device_class, disk_name, row))
    
    # 保存当前磁盘数据用于下次比较
    save_disk_data(current_disk_data)
    
    # 按磁盘名称字母顺序排序，并按设备类型分组
    all_disk_data.sort(key=lambda x: x[1])  # 按磁盘名称排序
    
    # 重建分组数据
    for device_class, disk_name, row in all_disk_data:
        grouped_disks[device_class].append(row)
    
    # 如果指定不分组显示，则显示单个表格
    if args.no_group:
        # 准备表头
        headers = ["磁盘名称", "类型", "型号", "容量", "存储池"]
        
        # 收集所有可能的属性
        all_attributes = []
        for device_type, attributes in DISK_TYPE_ATTRIBUTES.items():
            for _, display_name, _ in attributes:
                if display_name not in all_attributes:
                    all_attributes.append(display_name)
        
        # 添加所有属性到表头
        headers.extend(all_attributes)
        
        if HAS_PRETTYTABLE:
            table = PrettyTable(headers)
            for _, _, row_data in all_disk_data:
                # 确保行长度与表头相同
                row = row_data.copy()
                # 填充缺失的列
                while len(row) < len(headers):
                    row.append("N/A")
                # 截断超出的列数据
                if len(row) > len(headers):
                    row = row[:len(headers)]
                table.add_row(row)
            output_tables.append(table.get_string())
        else:
            # 填充每行缺失的列
            rows_for_table = []
            for _, _, row_data in all_disk_data:
                row = row_data.copy()
                while len(row) < len(headers):
                    row.append("N/A")
                # 截断超出的列数据
                if len(row) > len(headers):
                    row = row[:len(headers)]
                rows_for_table.append(row)
            output_tables.append(create_simple_table(headers, rows_for_table))
    else:
        # 按设备类型创建并显示分组表格
        display_order = ["SAS_SSD", "SAS_HDD", "NVME_SSD", "VIRTUAL"]
        
        for device_class in display_order:
            if device_class in grouped_disks and grouped_disks[device_class]:
                title = f"\n--- {DEVICE_TYPES[device_class]} ---\n"
                
                # 为该设备类型创建表头
                headers = ["磁盘名称", "类型", "型号", "容量", "存储池"]
                for _, display_name, _ in DISK_TYPE_ATTRIBUTES.get(device_class, []):
                    headers.append(display_name)
                
                if HAS_PRETTYTABLE:
                    table = PrettyTable(headers)
                    for row_data in grouped_disks[device_class]:
                        # 确保行长度与表头相同
                        row = row_data.copy()
                        # 填充缺失的列
                        while len(row) < len(headers):
                            row.append("N/A")
                        # 截断超出的列数据
                        if len(row) > len(headers):
                            row = row[:len(headers)]
                        table.add_row(row)
                    output_tables.append(title + table.get_string())
                else:
                    # 处理每行数据以确保行列匹配
                    rows_for_table = []
                    for row_data in grouped_disks[device_class]:
                        row = row_data.copy()
                        # 填充缺失的列
                        while len(row) < len(headers):
                            row.append("N/A")
                        # 截断超出的列数据
                        if len(row) > len(headers):
                            row = row[:len(headers)]
                        rows_for_table.append(row)
                    output_tables.append(title + create_simple_table(headers, rows_for_table))
    
    # 生成读写增量表格
    if previous_time:
        log_info(f"生成读写增量表格（相比于 {previous_time}）")
        increment_tables = []
        
        # 收集有增量信息的磁盘数据
        increment_disk_data = defaultdict(list)
        
        for device_class, disk_name, row in all_disk_data:
            disk = None
            for d in disks:
                if d.get("name") == disk_name:
                    disk = d
                    break
            
            if disk is None:
                continue
                
            # 确定磁盘类型
            disk_type = disk.get("type", "")
            disk_model = disk.get("model", "")
            disk_pool = disk_to_pool.get(disk_name, "未分配")
            
            # 获取智能数据
            smart_data = {}
            if device_class == "NVME_SSD":
                smart_data = get_nvme_smart_data(disk_name)
            elif device_class != "VIRTUAL":
                smart_data = get_sata_smart_data(disk_name, disk_type)
            
            # 只有存在增量数据的磁盘才显示
            read_incr = smart_data.get("Read_Increment", "N/A")
            write_incr = smart_data.get("Write_Increment", "N/A")
            
            if read_incr != "N/A" or write_incr != "N/A":
                # 创建增量表行
                incr_row = [
                    disk_name, 
                    disk_type, 
                    disk_model, 
                    disk_pool,
                    smart_data.get("Data_Read", "N/A"),
                    read_incr,
                    smart_data.get("Data_Written", "N/A"),
                    write_incr
                ]
                increment_disk_data[device_class].append(incr_row)
        
        # 创建增量表格
        if args.no_group:
            # 单个表格显示所有增量
            all_incr_rows = []
            for device_class, rows in increment_disk_data.items():
                all_incr_rows.extend(rows)
                
            if all_incr_rows:
                # 排序按磁盘名称
                all_incr_rows.sort(key=lambda row: row[0])
                
                title = f"\n--- 磁盘读写增量信息 (自 {previous_time}) ---\n"
                headers = ["磁盘名称", "类型", "型号", "存储池", "当前读取总量", "读取增量", "当前写入总量", "写入增量"]
                
                if HAS_PRETTYTABLE:
                    table = PrettyTable(headers)
                    for row in all_incr_rows:
                        table.add_row(row)
                    increment_tables.append(title + table.get_string())
                else:
                    increment_tables.append(title + create_simple_table(headers, all_incr_rows))
            else:
                increment_tables.append("\n--- 无可用的读写增量数据 ---\n")
        else:
            # 按设备类型分组显示增量
            display_order = ["SAS_SSD", "SAS_HDD", "NVME_SSD", "VIRTUAL"]
            
            for device_class in display_order:
                if device_class in increment_disk_data and increment_disk_data[device_class]:
                    title = f"\n--- {DEVICE_TYPES[device_class]} 读写增量 (自 {previous_time}) ---\n"
                    headers = ["磁盘名称", "类型", "型号", "存储池", "当前读取总量", "读取增量", "当前写入总量", "写入增量"]
                    
                    # 排序按磁盘名称
                    rows = sorted(increment_disk_data[device_class], key=lambda row: row[0])
                    
                    if HAS_PRETTYTABLE:
                        table = PrettyTable(headers)
                        for row in rows:
                            table.add_row(row)
                        increment_tables.append(title + table.get_string())
                    else:
                        increment_tables.append(title + create_simple_table(headers, rows))
            
            if not increment_tables:
                increment_tables.append("\n--- 无可用的读写增量数据 ---\n")
        
        # 添加增量表格到输出
        output_tables.extend(increment_tables)
    else:
        output_tables.append("\n--- 无法生成读写增量表格：未找到上次运行数据 ---\n")
    
    # 显示所有表格
    for table in output_tables:
        print(table)
    
    return output_tables

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description="TrueNAS磁盘健康监控工具")
    parser.add_argument("-d", "--debug", action="store_true", help="启用调试模式")
    parser.add_argument("-v", "--verbose", action="store_true", help="显示信息日志")
    parser.add_argument("-o", "--output", help="输出结果到文件")
    parser.add_argument("--no-group", action="store_true", help="不按类型分组显示")
    parser.add_argument("--no-controller", action="store_true", help="不显示控制器信息")
    parser.add_argument("--controller-only", action="store_true", help="只显示控制器信息")
    args = parser.parse_args()
    
    global DEBUG, VERBOSE
    DEBUG = args.debug
    VERBOSE = args.verbose
    
    # 创建日志目录
    os.makedirs(os.path.dirname(LOG_FILE), exist_ok=True)
    
    log_info("=== 磁盘健康监控工具开始运行 ===")
    
    # 检查smartctl是否已安装
    if not run_command("command -v smartctl"):
        log_error("未找到smartctl。请安装smartmontools包。")
        return 1
    
    # 处理控制器信息
    controller_tables = process_controller_info(args)
    
    # 处理磁盘信息
    output_tables = process_disk_info(args, controller_tables)
    
    # 如果指定了输出文件并且有磁盘表格，保存结果
    if args.output and not args.controller_only and output_tables:
        try:
            with open(args.output, 'w') as f:
                # 先写入控制器信息
                for table in controller_tables:
                    f.write(table + "\n\n")
                # 再写入磁盘信息
                for table in output_tables:
                    f.write(table + "\n\n")
            log_info(f"表格数据已保存到: {args.output}")
        except Exception as e:
            log_error(f"保存输出到文件失败: {e}")
    
    log_info("磁盘健康监控工具执行完成")
    return 0

if __name__ == "__main__":
    sys.exit(main())
