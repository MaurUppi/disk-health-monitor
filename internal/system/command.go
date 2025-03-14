package system

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CommandRunner 定义命令执行接口
type CommandRunner interface {
	// Run 执行命令并返回输出，如果命令失败则返回错误
	Run(ctx context.Context, command string) (string, error)
	
	// RunIgnoreError 执行命令并返回输出，忽略命令执行错误
	RunIgnoreError(ctx context.Context, command string) string
	
	// RunWithTimeout 执行命令并设置超时，返回输出，如果命令失败或超时则返回错误
	RunWithTimeout(command string, timeout time.Duration) (string, error)
}

// DefaultCommandRunner 实现CommandRunner接口的默认执行器
type DefaultCommandRunner struct{}

// Run 执行命令并返回输出
func (r DefaultCommandRunner) Run(ctx context.Context, command string) (string, error) {
	// 根据操作系统选择命令执行方式
	var cmd *exec.Cmd
	
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
	}
	
	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed [%s]: %w, output: %s", 
			command, err, string(output))
	}
	
	// 移除首尾空格并返回
	return strings.TrimSpace(string(output)), nil
}

// RunIgnoreError 执行命令并忽略错误
func (r DefaultCommandRunner) RunIgnoreError(ctx context.Context, command string) string {
	output, _ := r.Run(ctx, command)
	return output
}

// RunWithTimeout 使用指定的超时时间执行命令
func (r DefaultCommandRunner) RunWithTimeout(command string, timeout time.Duration) (string, error) {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return r.Run(ctx, command)
}

// MockCommandRunner 用于测试的模拟命令执行器
type MockCommandRunner struct {
	MockOutputs map[string]string // 命令到输出的映射
	MockErrors  map[string]error  // 命令到错误的映射
	CalledCommands []string      // 记录被调用的命令
}

// NewMockCommandRunner 创建一个新的模拟命令执行器
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		MockOutputs: make(map[string]string),
		MockErrors:  make(map[string]error),
		CalledCommands: []string{},
	}
}

// Run 返回预定义的模拟输出或错误
func (m *MockCommandRunner) Run(ctx context.Context, command string) (string, error) {
	m.CalledCommands = append(m.CalledCommands, command)
	
	// 检查是否有预定义的错误
	if err, ok := m.MockErrors[command]; ok && err != nil {
		return "", err
	}
	
	// 返回预定义的输出
	if output, ok := m.MockOutputs[command]; ok {
		return output, nil
	}
	
	// 默认返回空字符串
	return "", nil
}

// RunIgnoreError 模拟执行命令并忽略错误
func (m *MockCommandRunner) RunIgnoreError(ctx context.Context, command string) string {
	output, _ := m.Run(ctx, command)
	return output
}

// RunWithTimeout 模拟带超时的命令执行
func (m *MockCommandRunner) RunWithTimeout(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return m.Run(ctx, command)
}

// SetMockOutput 设置命令的模拟输出
func (m *MockCommandRunner) SetMockOutput(command, output string) {
	m.MockOutputs[command] = output
}

// SetMockError 设置命令的模拟错误
func (m *MockCommandRunner) SetMockError(command string, err error) {
	m.MockErrors[command] = err
}
