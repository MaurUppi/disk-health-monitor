package model

import (
	"testing"
)

func TestNewController(t *testing.T) {
	// 测试创建控制器
	controller := NewController("test_id", ControllerTypeLSI)
	
	if controller.ID != "test_id" {
		t.Errorf("Expected ID 'test_id', got '%s'", controller.ID)
	}
	
	if controller.Type != ControllerTypeLSI {
		t.Errorf("Expected Type '%s', got '%s'", ControllerTypeLSI, controller.Type)
	}
	
	if controller.Status != ControllerStatusUnknown {
		t.Errorf("Expected Status '%s', got '%s'", ControllerStatusUnknown, controller.Status)
	}
}

func TestController_SetStatus(t *testing.T) {
	// 测试设置状态
	controller := NewController("test_id", ControllerTypeLSI)
	controller.SetStatus(ControllerStatusOK)
	
	if controller.Status != ControllerStatusOK {
		t.Errorf("Expected Status '%s', got '%s'", ControllerStatusOK, controller.Status)
	}
}

func TestController_GetDisplayTemperature(t *testing.T) {
	// 测试获取可显示温度
	controller := NewController("test_id", ControllerTypeLSI)
	
	// 温度为空的情况
	if controller.GetDisplayTemperature() != "N/A" {
		t.Errorf("Expected 'N/A' for empty temperature, got '%s'", controller.GetDisplayTemperature())
	}
	
	// 设置温度
	controller.Temperature = "45"
	if controller.GetDisplayTemperature() != "45°C" {
		t.Errorf("Expected '45°C', got '%s'", controller.GetDisplayTemperature())
	}
}

func TestNewLSIController(t *testing.T) {
	// 测试创建LSI控制器
	controller := NewLSIController("lsi_controller_0")
	
	if controller.ID != "lsi_controller_0" {
		t.Errorf("Expected ID 'lsi_controller_0', got '%s'", controller.ID)
	}
	
	if controller.Type != ControllerTypeLSI {
		t.Errorf("Expected Type '%s', got '%s'", ControllerTypeLSI, controller.Type)
	}
}

func TestNewNVMeController(t *testing.T) {
	// 测试创建NVMe控制器
	controller := NewNVMeController("nvme_controller_0")
	
	if controller.ID != "nvme_controller_0" {
		t.Errorf("Expected ID 'nvme_controller_0', got '%s'", controller.ID)
	}
	
	if controller.Type != ControllerTypeNVMe {
		t.Errorf("Expected Type '%s', got '%s'", ControllerTypeNVMe, controller.Type)
	}
}

func TestControllerData(t *testing.T) {
	// 测试控制器数据集合
	data := NewControllerData()
	
	// 初始状态：空集合
	if data.GetLSIControllerCount() != 0 {
		t.Errorf("Expected LSI controller count 0, got %d", data.GetLSIControllerCount())
	}
	
	if data.GetNVMeControllerCount() != 0 {
		t.Errorf("Expected NVMe controller count 0, got %d", data.GetNVMeControllerCount())
	}
	
	if data.GetTotalControllerCount() != 0 {
		t.Errorf("Expected total controller count 0, got %d", data.GetTotalControllerCount())
	}
	
	// 添加控制器
	lsiController := data.GetLSIController("lsi_0")
	if lsiController == nil {
		t.Fatal("GetLSIController returned nil")
	}
	
	nvmeController := data.GetNVMeController("nvme_0")
	if nvmeController == nil {
		t.Fatal("GetNVMeController returned nil")
	}
	
	// 验证计数
	if data.GetLSIControllerCount() != 1 {
		t.Errorf("Expected LSI controller count 1, got %d", data.GetLSIControllerCount())
	}
	
	if data.GetNVMeControllerCount() != 1 {
		t.Errorf("Expected NVMe controller count 1, got %d", data.GetNVMeControllerCount())
	}
	
	if data.GetTotalControllerCount() != 2 {
		t.Errorf("Expected total controller count 2, got %d", data.GetTotalControllerCount())
	}
	
	// 再次获取相同ID的控制器，应返回已存在的对象
	lsiController2 := data.GetLSIController("lsi_0")
	if lsiController != lsiController2 {
		t.Error("GetLSIController should return the same object for the same ID")
	}
	
	nvmeController2 := data.GetNVMeController("nvme_0")
	if nvmeController != nvmeController2 {
		t.Error("GetNVMeController should return the same object for the same ID")
	}
	
	// 验证计数未变
	if data.GetLSIControllerCount() != 1 || data.GetNVMeControllerCount() != 1 {
		t.Error("Controller counts should not change when getting existing controllers")
	}
}
