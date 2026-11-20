package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	startupTaskName = "\\MatchRoute AutoStart"
	startupArgument = "--start-on-logon"
)

type startupTask struct {
	name, executable, sid string
}

func installedStartupTask() (*startupTask, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	programFiles, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFilesX64, 0)
	if err != nil {
		return nil, err
	}
	expected := filepath.Join(programFiles, "MatchRoute", "MatchRoute.exe")
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil || !strings.EqualFold(exe, expected) || !strings.EqualFold(resolved, expected) {
		return nil, errors.New("Автозапуск доступен в версии, установленной в стандартную папку Program Files.")
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\MatchRoute`, registry.QUERY_VALUE|registry.WOW64_32KEY)
	if err != nil {
		return nil, errors.New("Установите MatchRoute через установщик Windows.")
	}
	defer key.Close()
	location, _, err := key.GetStringValue("InstallLocation")
	if err != nil || !strings.EqualFold(filepath.Clean(location), filepath.Dir(expected)) {
		return nil, errors.New("Установите MatchRoute в стандартную папку Program Files.")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return &startupTask{name: startupTaskName, executable: expected, sid: user.User.Sid.String()}, nil
}

func runScheduler(args ...string) ([]byte, error) {
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(system, "schtasks.exe"), append(args, "/HRESULT")...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Output()
}

func taskNotFound(err error) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && uint32(exit.ExitCode()) == 0x80070002
}

func decodeTaskXML(data []byte) []byte {
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) ||
		(len(data) >= 2 && data[0] == '<' && data[1] == 0) {
		data = bytes.TrimPrefix(data, []byte{0xff, 0xfe})
		units := make([]uint16, 0, len(data)/2)
		for len(data) >= 2 {
			units = append(units, binary.LittleEndian.Uint16(data))
			data = data[2:]
		}
		data = []byte(string(utf16.Decode(units)))
	}
	data = bytes.TrimSpace(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}))
	if bytes.HasPrefix(data, []byte("<?xml")) {
		if end := bytes.Index(data, []byte("?>")); end >= 0 {
			return data[end+2:]
		}
	}
	return data
}

func (t *startupTask) status() (exists, enabled bool, err error) {
	data, err := runScheduler("/Query", "/TN", t.name, "/XML")
	if taskNotFound(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("не удалось прочитать задачу автозапуска: %w", err)
	}
	var task struct {
		UserID  string `xml:"Principals>Principal>UserId"`
		Enabled string `xml:"Settings>Enabled"`
		Actions []struct {
			Command   string `xml:"Command"`
			Arguments string `xml:"Arguments"`
		} `xml:"Actions>Exec"`
	}
	if err := xml.Unmarshal(decodeTaskXML(data), &task); err != nil {
		return true, false, fmt.Errorf("не удалось прочитать параметры автозапуска: %w", err)
	}
	if len(task.Actions) != 1 ||
		!strings.EqualFold(task.UserID, t.sid) ||
		!strings.EqualFold(task.Actions[0].Command, t.executable) ||
		task.Actions[0].Arguments != startupArgument {
		return true, false, errors.New("Задача автозапуска уже занята другой установкой или учётной записью.")
	}
	return true, task.Enabled != "false", nil
}

func xmlText(value string) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}

func (t *startupTask) definition() []byte {
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>Start MatchRoute in the notification area at sign-in.</Description></RegistrationInfo>
  <Triggers><LogonTrigger><Enabled>true</Enabled><UserId>%[1]s</UserId><Delay>PT15S</Delay></LogonTrigger></Triggers>
  <Principals><Principal id="Owner"><UserId>%[1]s</UserId><LogonType>InteractiveToken</LogonType><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <StartWhenAvailable>true</StartWhenAvailable>
    <Enabled>true</Enabled>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
  </Settings>
  <Actions Context="Owner"><Exec><Command>%[2]s</Command><Arguments>%[3]s</Arguments><WorkingDirectory>%[4]s</WorkingDirectory></Exec></Actions>
</Task>`, xmlText(t.sid), xmlText(t.executable), startupArgument, xmlText(filepath.Dir(t.executable)))
	data := []byte{0xff, 0xfe}
	for _, unit := range utf16.Encode([]rune(body)) {
		data = binary.LittleEndian.AppendUint16(data, unit)
	}
	return data
}

func (t *startupTask) set(enabled bool) error {
	exists, previous, err := t.status()
	if err != nil {
		return err
	}
	if enabled && previous || !enabled && !exists {
		return nil
	}
	if !enabled {
		_, err := runScheduler("/Delete", "/TN", t.name, "/F")
		if err != nil && !taskNotFound(err) {
			return fmt.Errorf("не удалось отключить автозапуск: %w", err)
		}
		return nil
	}
	file, err := os.CreateTemp(filepath.Dir(t.executable), "matchroute-startup-*.xml")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(t.definition())
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	_, err = runScheduler("/Create", "/TN", t.name, "/XML", file.Name(), "/F")
	if err != nil {
		return fmt.Errorf("не удалось включить автозапуск: %w", err)
	}
	return nil
}
