package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddListRemoveAndClear(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := run([]string{"add", "Clean desk"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("add first task: %v", err)
	}
	if err := run([]string{"add", "Reply to emails", "-d", "Needs agenda"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("add second task: %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"ls"}, &out); err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if got, want := out.String(), "[0] Clean desk\n[1] Reply to emails\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}

	raw, err := os.ReadFile(testStatePath(home))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}

	var stored []map[string]string
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored item count = %d, want 2", len(stored))
	}
	if len(stored[1]) != 2 || stored[1]["name"] != "Reply to emails" || stored[1]["description"] != "Needs agenda" {
		t.Fatalf("stored item = %#v, want name and description only", stored[1])
	}

	if err := run([]string{"rm", "0"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("remove task: %v", err)
	}

	out.Reset()
	if err := run([]string{"list"}, &out); err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if got, want := out.String(), "[0] Reply to emails\n"; got != want {
		t.Fatalf("list after remove = %q, want %q", got, want)
	}

	if err := run([]string{"clear"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("clear: %v", err)
	}

	entries, err := os.ReadDir(testQueueDir(home))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read queue dir after clear: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("queue dir entry count after clear = %d, want 0", len(entries))
	}
}

func TestListCreatesEmptyStateFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var out bytes.Buffer
	if err := run([]string{"list"}, &out); err != nil {
		t.Fatalf("list empty queue: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("empty list output = %q, want empty", out.String())
	}

	raw, err := os.ReadFile(testStatePath(home))
	if err != nil {
		t.Fatalf("read created state: %v", err)
	}
	if got, want := string(raw), "[]\n"; got != want {
		t.Fatalf("created state = %q, want %q", got, want)
	}
}

func TestAddRejectsEmptyTask(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := run([]string{"add", "   "}, &bytes.Buffer{}); err == nil {
		t.Fatal("add empty task succeeded, want error")
	}
}

func TestAddInsertsAtPosition(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := run([]string{"add", "Clean desk"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("add first task: %v", err)
	}
	if err := run([]string{"add", "Finish homework"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("add second task: %v", err)
	}
	if err := run([]string{"add", "Reply to emails", "-i", "1"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"list"}, &out); err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if got, want := out.String(), "[0] Clean desk\n[1] Reply to emails\n[2] Finish homework\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}
}

func TestAddValidatesInsertPosition(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := run([]string{"add", "Clean desk"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("add task: %v", err)
	}
	if err := run([]string{"add", "Reply to emails", "-i", "x"}, &bytes.Buffer{}); err == nil {
		t.Fatal("add with non-integer insert position succeeded, want error")
	}
	if err := run([]string{"add", "Reply to emails", "-i", "-1"}, &bytes.Buffer{}); err == nil {
		t.Fatal("add with negative insert position succeeded, want error")
	}
	if err := run([]string{"add", "Reply to emails", "-i", "2"}, &bytes.Buffer{}); err == nil {
		t.Fatal("add with out-of-range insert position succeeded, want error")
	}
}

func TestRemoveValidatesIndex(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := run([]string{"add", "Clean desk"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("add task: %v", err)
	}
	if err := run([]string{"remove", "x"}, &bytes.Buffer{}); err == nil {
		t.Fatal("remove non-integer index succeeded, want error")
	}
	if err := run([]string{"remove", "1"}, &bytes.Buffer{}); err == nil {
		t.Fatal("remove out-of-range index succeeded, want error")
	}
}

func testQueueDir(home string) string {
	return filepath.Join(home, "Library", "Application Support", "queue")
}

func testStatePath(home string) string {
	return filepath.Join(testQueueDir(home), "state.json")
}
