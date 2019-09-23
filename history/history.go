package history

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

type History struct {
	path string
	data []string
	save bool
}

// New ...
func New(path string) *History {
	return &History{
		path: path,
		data: []string{},
		save: false,
	}
}

func (h *History) UseHistoryFile() error {
	h.save = true
	if _, err := os.Lstat(h.path); err != nil {
		return err
	}
	f, err := os.Open(h.path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		h.data = append([]string{scanner.Text()}, h.data...)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (h *History) Raw() []string {
	return h.data
}

func (h *History) Append(inputStr string) error {
	h.data = unique(append([]string{inputStr}, h.data...))
	if h.save {
		dir := filepath.Dir(h.path)
		if err := os.MkdirAll(dir, os.ModeDir|0700); err != nil {
			return err
		}
		f, err := os.OpenFile(h.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
		defer f.Close()
		fmt.Fprintln(f, inputStr)
	}
	return nil
}

func unique(strs []string) []string {
	keys := make(map[string]bool)
	uniqStrs := []string{}
	for _, s := range strs {
		if s == "" {
			continue
		}
		if _, value := keys[s]; !value {
			keys[s] = true
			uniqStrs = append(uniqStrs, s)
		}
	}
	return uniqStrs
}
