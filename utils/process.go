package utils

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CheckPortOccupied 检查端口是否被占用
func CheckPortOccupied(port int) (bool, string, error) {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return false, "", err
	}

	portStr := fmt.Sprintf(":%d", port)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, portStr) {
			// 提取 PID
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				pid := parts[4]
				return true, pid, nil
			}
		}
	}

	return false, "", nil
}

// KillProcess 终止指定 PID 的进程
func KillProcess(pid string) error {
	cmd := exec.Command("taskkill", "/F", "/PID", pid)
	_, err := cmd.CombinedOutput()
	return err
}

// EnsurePortAvailable 确保端口可用，如果被占用则终止占用进程
func EnsurePortAvailable(port int) error {
	occupied, pid, err := CheckPortOccupied(port)
	if err != nil {
		return fmt.Errorf("检查端口占用失败: %w", err)
	}

	if occupied {
		log.Printf("端口 %d 被进程 %s 占用，正在终止...", port, pid)
		err := KillProcess(pid)
		if err != nil {
			return fmt.Errorf("终止占用进程失败: %w", err)
		}
		log.Printf("成功终止占用端口 %d 的进程 %s", port, pid)
	}

	return nil
}
