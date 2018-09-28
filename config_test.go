package roll

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestConfigNoFile(t *testing.T) {
	_, err := LoadConfig("")
	if err == nil {
		t.Errorf("Did not get error when trying to load blank file name config.")
	}
}

func TestConfigBadJson(t *testing.T) {
	f, err := ioutil.TempFile("", "config-*.json")
	if err != nil {
		t.Fatalf("Can't create temp file: %v", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	f.Write([]byte("{"))
	f.Sync()

	_, err = LoadConfig(f.Name())
	if err == nil {
		t.Errorf("Did not get error when trying to invalid.")
	}

	f.Write([]byte("}"))
	f.Sync()

	_, err = LoadConfig(f.Name())
	if err != nil {
		t.Errorf("Got error when loading config: %v", err)
	}
}
