package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

const taskName = "NovaKey"

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func whoami() (string, error) {
	out, err := exec.Command("whoami").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  novakey-installer-helper install <AppDir> <DataDir>")
	fmt.Println("  novakey-installer-helper uninstall")
}

func taskXML(user, exePath, workDir, configPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>NovaKey secure secret transfer service (per-user)</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>%s</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>%s</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>3</Count>
    </RestartOnFailure>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <Arguments>--config "%s"</Arguments>
      <WorkingDirectory>%s</WorkingDirectory>
    </Exec>
  </Actions>
</Task>`, user, user, exePath, configPath, workDir)
}

func toUTF16LEWithBOM(s string) []byte {
	// UTF-16LE BOM
	out := make([]byte, 2)
	binary.LittleEndian.PutUint16(out[0:2], 0xFEFF)

	u16 := utf16.Encode([]rune(s))
	buf := make([]byte, 2*len(u16))
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(buf[i*2:(i+1)*2], v)
	}
	return append(out, buf...)
}

func installTask(appDir, dataDir string) error {
	user, err := whoami()
	if err != nil {
		return fmt.Errorf("whoami failed: %w", err)
	}

	exePath := filepath.Join(appDir, "novakey.exe")
	cfgPath := filepath.Join(dataDir, "server_config.yaml")

	// Ensure data dir exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}

	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("missing exe: %s", exePath)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		return fmt.Errorf("missing config: %s", cfgPath)
	}

	xml := taskXML(user, exePath, dataDir, cfgPath)
	tmp := filepath.Join(os.TempDir(), "novakey-task.xml")

	// schtasks can be picky; UTF-16LE with BOM is most compatible.
	if err := os.WriteFile(tmp, toUTF16LEWithBOM(xml), 0644); err != nil {
		return fmt.Errorf("write xml: %w", err)
	}
	defer os.Remove(tmp)

	_ = run("schtasks", "/Delete", "/TN", taskName, "/F") // ignore

	if err := run("schtasks", "/Create", "/TN", taskName, "/XML", tmp, "/F"); err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	_ = run("schtasks", "/Run", "/TN", taskName) // best effort
	return nil
}

func uninstallTask() {
	_ = run("schtasks", "/End", "/TN", taskName)
	_ = run("schtasks", "/Delete", "/TN", taskName, "/F")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "install":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		if err := installTask(os.Args[2], os.Args[3]); err != nil {
			fmt.Fprintln(os.Stderr, "Install failed:", err)
			os.Exit(1)
		}
		fmt.Println("NovaKey Scheduled Task installed and started.")
	case "uninstall":
		uninstallTask()
		fmt.Println("NovaKey Scheduled Task removed.")
	default:
		usage()
		os.Exit(2)
	}
}

