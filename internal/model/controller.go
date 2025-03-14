package model

// ControllerType 定义控制器类型
type ControllerType string

const (
	// ControllerTypeLSI LSI SAS控制器
	ControllerTypeLSI ControllerType = "LSI_SAS_HBA"
	// ControllerTypeNVMe NVMe控制器
	ControllerTypeNVMe ControllerType = "PCIe_NVMe"
)

// ControllerStatus 定义控制器状态
type ControllerStatus string

const (
	// ControllerStatusOK 控制器状态正常
	ControllerStatusOK ControllerStatus = "正常"
	// ControllerStatusWarning 控制器状态警告
	ControllerStatusWarning ControllerStatus = "警告"
	// ControllerStatusError 控制器状态错误
	ControllerStatusError ControllerStatus = "错误"
	// ControllerStatusUnknown 控制器状态未知
	ControllerStatusUnknown ControllerStatus = "未知"
)

// Controller 控制器信息
type Controller struct {
	ID             string          // 控制器ID
	Type           ControllerType  // 控制器类型
	Model          string          // 控制器型号
	Bus            string          // 总线ID
	FirmwareVersion string         // 固件版本
	DriverVersion  string          // 驱动版本
	Temperature    string          // 温度
	DeviceCount    string          // 设备数量
	SSDCount       string          // SSD数量
	HDDCount       string          // HDD数量
	Status         ControllerStatus // 状态
	Description    string          // 描述信息
	Source         string          // 信息来源
}

// NewController 创建一个新的控制器对象
func NewController(id string, controllerType ControllerType) *Controller {
	return &Controller{
		ID:             id,
		Type:           controllerType,
		Status:         ControllerStatusUnknown,
	}
}

// SetStatus 设置控制器状态
func (c *Controller) SetStatus(status ControllerStatus) {
	c.Status = status
}

// GetDisplayTemperature 获取可显示的温度值
func (c *Controller) GetDisplayTemperature() string {
	if c.Temperature == "" {
		return "N/A"
	}
	return c.Temperature + "°C"
}

// LSIController LSI SAS控制器
type LSIController struct {
	Controller
	ROCTemperature string   // ROC温度
	ProductName    string   // 产品名称
}

// NewLSIController 创建一个新的LSI控制器
func NewLSIController(id string) *LSIController {
	return &LSIController{
		Controller: Controller{
			ID:     id,
			Type:   ControllerTypeLSI,
			Status: ControllerStatusUnknown,
		},
	}
}

// NVMeController NVMe控制器
type NVMeController struct {
	Controller
	PCIAddress string   // PCI地址
}

// NewNVMeController 创建一个新的NVMe控制器
func NewNVMeController(id string) *NVMeController {
	return &NVMeController{
		Controller: Controller{
			ID:     id,
			Type:   ControllerTypeNVMe,
			Status: ControllerStatusUnknown,
		},
	}
}

// ControllerData 控制器数据集合
type ControllerData struct {
	LSIControllers  map[string]*LSIController  // LSI控制器
	NVMeControllers map[string]*NVMeController // NVMe控制器
}

// NewControllerData 创建一个新的控制器数据集合
func NewControllerData() *ControllerData {
	return &ControllerData{
		LSIControllers:  make(map[string]*LSIController),
		NVMeControllers: make(map[string]*NVMeController),
	}
}

// GetLSIController 获取指定ID的LSI控制器，如不存在则创建
func (cd *ControllerData) GetLSIController(id string) *LSIController {
	if controller, exists := cd.LSIControllers[id]; exists {
		return controller
	}
	
	controller := NewLSIController(id)
	cd.LSIControllers[id] = controller
	return controller
}

// GetNVMeController 获取指定ID的NVMe控制器，如不存在则创建
func (cd *ControllerData) GetNVMeController(id string) *NVMeController {
	if controller, exists := cd.NVMeControllers[id]; exists {
		return controller
	}
	
	controller := NewNVMeController(id)
	cd.NVMeControllers[id] = controller
	return controller
}

// GetLSIControllerCount 获取LSI控制器数量
func (cd *ControllerData) GetLSIControllerCount() int {
	return len(cd.LSIControllers)
}

// GetNVMeControllerCount 获取NVMe控制器数量
func (cd *ControllerData) GetNVMeControllerCount() int {
	return len(cd.NVMeControllers)
}

// GetTotalControllerCount 获取控制器总数
func (cd *ControllerData) GetTotalControllerCount() int {
	return cd.GetLSIControllerCount() + cd.GetNVMeControllerCount()
}
