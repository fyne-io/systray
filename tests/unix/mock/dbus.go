package mock

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

type DbusDaemon struct {
	address    string
	cmd        *exec.Cmd
	monitorCmd *exec.Cmd
}

func (d *DbusDaemon) Close() {
	// Kill the daemon
	if d.monitorCmd != nil && d.monitorCmd.Process != nil {
		d.monitorCmd.Process.Kill()
		d.monitorCmd.Process.Wait()
		d.monitorCmd = nil
	}
	fmt.Println("D-Bus monitor stopped")

	os.Unsetenv("DBUS_SESSION_BUS_ADDRESS")
	if d.cmd != nil && d.cmd.Process != nil {
		d.cmd.Process.Kill()
		d.cmd.Wait()
	}
}

func startDbusDaemon(t *testing.T) *DbusDaemon {
	t.Helper()

	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start dbus-daemon: %v", err)
	}

	// read the printed address (single line)
	r := bufio.NewReader(stdout)
	addr, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read address: %v", err)
	}
	addr = strings.TrimSpace(addr)

	fmt.Println("Address:", addr)
	// set env so dbus.SessionBus() connects to it
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", addr)

	monitorCmd := startBusMonitor(t, addr)

	// small pause for daemon to become ready (adjust if needed)
	time.Sleep(100 * time.Millisecond)

	return &DbusDaemon{
		address:    addr,
		cmd:        cmd,
		monitorCmd: monitorCmd,
	}
}

func startBusMonitor(t *testing.T, address string) *exec.Cmd {
	cmd := exec.Command("dbus-monitor", "--address", address)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}

	// Start the process BEFORE reading its output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		// Increase buffer if long lines possible
		buf := make([]byte, 0, 1024*64)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			fmt.Println("[OUT]", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("stdout error:", err)
		}
	}()

	// Read stderr line-by-line
	go func() {
		scanner := bufio.NewScanner(stderr)
		buf := make([]byte, 0, 1024*64)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			fmt.Println("[ERR]", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("stderr error:", err)
		}
	}()

	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Println("dbus-monitor exited with error:", err)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	return cmd
}

// createDBusConnection connects to the D-Bus session bus using DBUS_SESSION_BUS_ADDRESS
func createDBusConnection() (*dbus.Conn, error) {
	address := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if address == "" {
		return nil, fmt.Errorf("DBUS_SESSION_BUS_ADDRESS not set")
	}

	conn, err := dbus.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial D-Bus at %s: %w", address, err)
	}

	err = conn.Auth(nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	err = conn.Hello()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	return conn, nil
}
