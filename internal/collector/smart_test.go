package collector

import (
	"context"
	"testing"

	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

func TestSMARTCollector_GetSMARTData(t *testing.T) {
	// 创建模拟命令执行器
	mockRunner := system.NewMockCommandRunner()
	mockLogger := system.NewMockLogger()
	config := model.NewDefaultConfig()

	collector := NewSMARTCollector(config, mockLogger, mockRunner)

	// 测试 1: NVMe SSD (/dev/nvme5n1)
	mockRunner.SetMockOutput("smartctl -H /dev/nvme5n1", "SMART overall-health self-assessment test result: PASSED")
	mockRunner.SetMockOutput("smartctl -a /dev/nvme5n1", `
=== START OF SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED

SMART/Health Information (NVMe Log 0x02)
Critical Warning:                   0x00
Temperature:                        42 Celsius
Available Spare:                    100%
Available Spare Threshold:          10%
Percentage Used:                    0%
Data Units Read:                    10,970,743 [5.61 TB]
Data Units Written:                 6,402,614 [3.27 TB]
Host Read Commands:                 1,265,944,386
Host Write Commands:                738,303,902
Controller Busy Time:               22
Power Cycles:                       219
Power On Hours:                     20,662
Unsafe Shutdowns:                   157
Media and Data Integrity Errors:    0
Error Information Log Entries:      0
Warning  Comp. Temperature Time:    0
Critical Comp. Temperature Time:    0
`)

	smartData, err := collector.GetSMARTData(context.Background(), "nvme5n1", "SSD", "INTEL SSDPF2KX038TZ")
	if err != nil {
		t.Errorf("Expected no error for NVMe SSD, got %v", err)
	}

	expectedNVMeData := map[string]string{
		"Data_Read":          "5.61 TB",
		"Data_Written":       "3.27 TB",
		"Power_On_Hours":     "20662",
		"Power_Cycles":       "219",
		"Uncorrected_Errors": "0", // NVMe 使用 Media and Data Integrity Errors
	}

	for key, expected := range expectedNVMeData {
		if smartData[key] != expected {
			t.Errorf("NVMe SSD %s: Expected '%s', got '%s'", key, expected, smartData[key])
		}
	}

	// 测试 2: SAS HDD (/dev/sdd)
	mockRunner.SetMockOutput("smartctl -H /dev/sdd", "SMART Health Status: OK")
	mockRunner.SetMockOutput("smartctl -a /dev/sdd", `
=== START OF READ SMART DATA SECTION ===
SMART Health Status: OK

Current Drive Temperature:     37 C
Drive Trip Temperature:        68 C

Accumulated power on time, hours:minutes 36491:23
Manufactured in week  of year 20
Specified cycle count over device lifetime:  10000
Accumulated start-stop cycles:  276
Specified load-unload count over device lifetime:  300000
Accumulated load-unload cycles:  2224
Elements in grown defect list: 0

Vendor (Seagate/Hitachi) factory information
  number of hours powered up = 36491.38
  number of minutes until next internal SMART test = 24

Error counter log:
           Errors Corrected by           Total   Correction     Gigabytes    Total
               ECC          rereads/    errors   algorithm      processed    uncorrected
           fast | delayed   rewrites  corrected  invocations   [10^9 bytes]  errors
read:   3095384993       13         0  3095385006         13     280210.005           0
write:         0        0        22        22         24     183549.238           0
verify: 2031052729        0         0  2031052729          1      37272.521           1

Non-medium error count:      157
`)

	smartData, err = collector.GetSMARTData(context.Background(), "sdd", "HDD", "SEAGATE ST600MM0006")
	if err != nil {
		t.Errorf("Expected no error for SAS HDD, got %v", err)
	}

	expectedSASData := map[string]string{
		"Power_On_Hours":     "36491.38",
		"Power_Cycles":       "276",
		"Data_Read":          "280.21 TB", // 280210.005 GB 转换为 TB，保留两位小数
		"Data_Written":       "183.55 TB", // 183549.238 GB 转换为 TB，保留两位小数
		"Uncorrected_Errors": "0",
	}

	for key, expected := range expectedSASData {
		if smartData[key] != expected {
			t.Errorf("SAS HDD %s: Expected '%s', got '%s'", key, expected, smartData[key])
		}
	}

	// 测试 3: SAS SSD (/dev/sde)
	mockRunner.SetMockOutput("smartctl -H /dev/sde", "SMART Health Status: OK")
	mockRunner.SetMockOutput("smartctl -a /dev/sde", `
=== START OF READ SMART DATA SECTION ===
SMART Health Status: OK

Percentage used endurance indicator: 0%
Current Drive Temperature:     39 C
Drive Trip Temperature:        70 C

Accumulated power on time, hours:minutes 23002:02
Manufactured in week 38 of year 2020
Accumulated start-stop cycles:  122
Specified load-unload count over device lifetime:  0
Accumulated load-unload cycles:  0
Elements in grown defect list: 0

Error counter log:
           Errors Corrected by           Total   Correction     Gigabytes    Total
               ECC          rereads/    errors   algorithm      processed    uncorrected
           fast | delayed   rewrites  corrected  invocations   [10^9 bytes]  errors
read:          0        0         0         0          0      71532.573           0
write:         0        0         0         0          0      16924.206           0
verify:        0        0         0         0          0         85.043           0

Non-medium error count:     1221
`)

	smartData, err = collector.GetSMARTData(context.Background(), "sde", "SSD", "SAMSUNG MZILT3T8HALS/007")
	if err != nil {
		t.Errorf("Expected no error for SAS SSD, got %v", err)
	}

	expectedSASSSDData := map[string]string{
		"Power_On_Hours":     "23002", // 从 23002:02 中提取小时数
		"Power_Cycles":       "122",
		"Data_Read":          "71.53 TB", // 71532.573 GB 转换为 TB，保留两位小数
		"Data_Written":       "16.92 TB", // 16924.206 GB 转换为 TB，保留两位小数
		"Uncorrected_Errors": "0",
	}

	for key, expected := range expectedSASSSDData {
		if smartData[key] != expected {
			t.Errorf("SAS SSD %s: Expected '%s', got '%s'", key, expected, smartData[key])
		}
	}
}
