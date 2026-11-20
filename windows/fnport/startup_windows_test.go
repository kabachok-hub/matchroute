package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestStartupDefinition(t *testing.T) {
	task := startupTask{
		executable: filepath.Join("C:\\", "Program Files", "MatchRoute & Co", "MatchRoute.exe"),
		sid:        "S-1-5-21-1000",
	}
	var fields struct {
		TriggerUser string `xml:"Triggers>LogonTrigger>UserId"`
		Principal   string `xml:"Principals>Principal>UserId"`
		LogonType   string `xml:"Principals>Principal>LogonType"`
		Limit       string `xml:"Settings>ExecutionTimeLimit"`
		Command     string `xml:"Actions>Exec>Command"`
		Arguments   string `xml:"Actions>Exec>Arguments"`
		Directory   string `xml:"Actions>Exec>WorkingDirectory"`
	}
	if err := xml.Unmarshal(decodeTaskXML(task.definition()), &fields); err != nil {
		t.Fatal(err)
	}
	if fields.TriggerUser != task.sid || fields.Principal != task.sid ||
		fields.LogonType != "InteractiveToken" || fields.Limit != "PT0S" ||
		fields.Command != task.executable || fields.Arguments != startupArgument ||
		fields.Directory != filepath.Dir(task.executable) {
		t.Fatalf("unexpected startup definition: %+v", fields)
	}
}

func TestStartupSchedulerLifecycle(t *testing.T) {
	exe := os.Getenv("MATCHROUTE_TEST_INSTALLED_EXE")
	if exe == "" {
		t.Skip("set MATCHROUTE_TEST_INSTALLED_EXE to an existing protected installation")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	task := startupTask{
		name:       fmt.Sprintf("\\MatchRoute Startup Test %d", time.Now().UnixNano()),
		executable: exe,
		sid:        user.User.Sid.String(),
	}
	t.Cleanup(func() {
		if _, err := runScheduler("/Delete", "/TN", task.name, "/F"); err != nil && !taskNotFound(err) {
			t.Error(err)
		}
	})
	assert := func(want bool) {
		t.Helper()
		exists, enabled, err := task.status()
		if err != nil || exists != want || enabled != want {
			t.Fatalf("task exists=%t enabled=%t, want %t: %v", exists, enabled, want, err)
		}
	}
	assert(false)
	if err := task.set(true); err != nil {
		t.Fatal(err)
	}
	assert(true)
	if err := task.set(true); err != nil {
		t.Fatal(err)
	}
	if err := task.set(false); err != nil {
		t.Fatal(err)
	}
	assert(false)
}
