package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	queueDirRel = "Library/Application Support/queue"
	stateName   = "state.json"
	tmpName     = "state.tmp"
)

type item struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		//Print error and exit
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: queue <add|ls|list|rm|remove|clear> ...")
	}

	switch args[0] {
	case "add":
		return add(args[1:])
	case "ls", "list":
		return list(stdout)
	case "rm", "remove":
		return remove(args[1:])
	case "clear":
		return clear()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func add(args []string) error {
	name, description, position, err := parseAddArgs(args)
	if err != nil {
		return err
	}

	items, err := loadItems()
	if err != nil {
		return err
	}

	newItem := item{Name: name, Description: description}
	if position > -1 {
		if position > len(items) {
			return errors.New("insert position out of range")
		}
		items = append(items[:position], append([]item{newItem}, items[position:]...)...)
	} else {
		items = append(items, newItem)
	}
	return writeItems(items)
}

func parseAddArgs(args []string) (string, string, int, error) {
	var taskParts []string
	var description string
	position := -1

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-d", "--description":
			if i+1 >= len(args) {
				return "", "", -1, errors.New("description argument cannot be empty")
			}
			i++
			description = strings.TrimSpace(args[i])
		case "-i", "--insert":
			if i+1 >= len(args) {
				return "", "", -1, errors.New("insert argument cannot be empty")
			}
			i++
			parsedPosition, err := strconv.Atoi(args[i])
			if err != nil {
				return "", "", -1, errors.New("insert argument must be an integer")
			}
			if parsedPosition < 0 {
				return "", "", -1, errors.New("insert argument cannot be negative")
			}
			position = parsedPosition
		case "--":
			taskParts = append(taskParts, args[i+1:]...)
			i = len(args)
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", -1, fmt.Errorf("unknown add option %q", arg)
			}
			taskParts = append(taskParts, arg)
		}
	}

	task := strings.TrimSpace(strings.Join(taskParts, " "))
	if task == "" {
		return "", "", -1, errors.New("task cannot be empty")
	}

	return task, description, position, nil
}

func list(stdout io.Writer) error {
	items, err := loadItems()
	if err != nil {
		return err
	}
	if err := writeItems(items); err != nil {
		return err
	}

	for i, item := range items {
		fmt.Fprintf(stdout, "[%d] %s\n", i, item.Name)
	}

	return nil
}

func remove(args []string) error {
	if len(args) == 0 {
		return errors.New("remove index cannot be empty")
	}
	if len(args) > 1 {
		return errors.New("remove accepts one integer argument")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.New("remove index must be an integer")
	}

	items, err := loadItems()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(items) {
		return errors.New("remove index out of range")
	}

	items = append(items[:index], items[index+1:]...)
	return writeItems(items)
}

func clear() error {
	dir, _, _, err := statePaths()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func loadItems() ([]item, error) {
	_, statePath, _, err := statePaths()
	if err != nil {
		return nil, err
	}
	if err := ensureStateFile(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []item{}, nil
	}

	var items []item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	if items == nil {
		return []item{}, nil
	}

	return items, nil
}

func ensureStateFile() error {
	_, statePath, _, err := statePaths()
	if err != nil {
		return err
	}

	if _, err := os.Stat(statePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return writeItems([]item{})
}

func writeItems(items []item) error {
	dir, statePath, tmpPath, err := statePaths()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, statePath)
}

func statePaths() (string, string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", err
	}

	dir := filepath.Join(home, queueDirRel)
	return dir, filepath.Join(dir, stateName), filepath.Join(dir, tmpName), nil
}
