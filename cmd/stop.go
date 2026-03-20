package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"comfy-swap/internal/config"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running comfy-swap server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopServer()
		},
	}
	rootCmd.AddCommand(cmd)
}

func stopServer() error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:" + config.DefaultPort + "/api/health")
	if err != nil {
		fmt.Println("No comfy-swap server running")
		return nil
	}
	resp.Body.Close()

	pid, err := findServerPID()
	if err != nil || pid == 0 {
		fmt.Println("Server is running but could not find PID")
		fmt.Println("Try: taskkill /F /IM comfy-swap.exe (Windows) or pkill comfy-swap (Linux/macOS)")
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	} else {
		if err := proc.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	fmt.Printf("Stopped comfy-swap server (PID: %d)\n", pid)
	return nil
}

func findServerPID() (int, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("wmic", "process", "where", "name='comfy-swap.exe'", "get", "processid")
	} else {
		cmd = exec.Command("pgrep", "-f", "comfy-swap.*serve")
	}

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if pid, err := strconv.Atoi(line); err == nil && pid > 0 {
			return pid, nil
		}
	}
	return 0, nil
}
