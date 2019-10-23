package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoad(t *testing.T) {
	viper.Reset()
	os.Setenv("XDG_CONFIG_HOME", "./../testdata")
	Load()
	want := true
	got := viper.GetBool("history.enable")
	if want != got {
		t.Errorf("Load(): got = %v ,want = %v", got, want)
	}
	want2 := "/path/to/filt"
	got2 := viper.GetString("history.path")
	if want2 != got2 {
		t.Errorf("Load(): got = %v, want = %v", got2, want2)
	}
}

func TestSave(t *testing.T) {
	viper.Reset()
	dir, err := ioutil.TempDir("", "filt")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("XDG_CONFIG_HOME", dir)
	Load()
	err = Set("history.path", "/path/to/save/history")
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = Save()
	if err != nil {
		t.Fatalf("%v", err)
	}
	content, err := ioutil.ReadFile(filepath.Join(dir, "filt", "config.toml"))
	if err != nil {
		t.Fatalf("%v", err)
	}
	got := string(content)
	want := `[history]
enable = false
path = "/path/to/save/history"
`
	if want != got {
		t.Errorf("Save():\ngot = %v,\nwant = %v", got, want)
	}
}
