package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDnfAndCreateDev(t *testing.T) {
	err := CheckDnfAndCreateDev("/tmp/test")
	if err != nil {
		t.Errorf("TestCheckDnfAndCreateDev failed:%v", err)
	}
	_, err = os.Stat("/tmp/test/dev")
	if err != nil {
		t.Errorf("TestCheckDnfAndCreateDev failed to create /dev director:%v", err)
	}
	os.RemoveAll("/tmp/test")
}

func TestCheckVarsFile(t *testing.T) {
	dir := t.TempDir()
	err := CheckVarsFile(dir)
	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
	_, err = os.Stat(filepath.Join(dir, "/etc/yum/vars"))
	if os.IsNotExist(err) {
		t.Errorf("CheckVarsFile() failed to create /etc/yum/vars directory")
	}
}
