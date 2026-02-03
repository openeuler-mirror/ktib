/*
   Copyright (c) 2023 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package rpm

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gitee.com/openeuler/ktib/pkg/fusion/types"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/stretchr/testify/assert"
)

// Helper process for mocking exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Verify arguments
	args := os.Args
	// args[0] is binary name
	// args[3] is command name ("rpm")
	// args[4...] are arguments
	
	// Print arguments for debugging if needed
	// fmt.Fprintf(os.Stderr, "Mock exec called with: %v\n", args)

	// Check if it's the rpm command
	cmdName := filepath.Base(args[3]) // args[3] might be full path
	if cmdName == "rpm" || cmdName == "rpm.exe" {
		// Verify critical flags
		hasDbPath := false
		for _, arg := range args {
			if arg == "--dbpath" {
				hasDbPath = true
				break
			}
		}
		if !hasDbPath {
			fmt.Fprintf(os.Stderr, "Error: --dbpath missing in rpm call\n")
			os.Exit(1)
		}
		// Success
		os.Exit(0)
	}

	os.Exit(1)
}

func TestReconstructWithRealDB(t *testing.T) {
	// 1. Check if we have a real RPM DB to test with (e.g. inside container)
	realDBPath := "/var/lib/rpm"
	if _, err := os.Stat(filepath.Join(realDBPath, "rpmdb.sqlite")); os.IsNotExist(err) {
		t.Skipf("Skipping test: no real rpmdb.sqlite found at %s", realDBPath)
	}

	// 2. Setup temp directories
	tmpDir, err := ioutil.TempDir("", "ktib-test-rpm-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "source")
	outDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(srcDir, 0755)

	// 3. Copy real DB to srcDir
	// We need to copy at least rpmdb.sqlite
	err = copyFile(filepath.Join(realDBPath, "rpmdb.sqlite"), filepath.Join(srcDir, "rpmdb.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	// 4. Read packages from DB to decide what to keep
	db, err := rpmdb.Open(filepath.Join(srcDir, "rpmdb.sqlite"))
	if err != nil {
		t.Fatalf("Failed to open copied rpmdb: %v", err)
	}
	pkgs, err := db.ListPackages()
	if err != nil {
		t.Fatalf("Failed to list packages: %v", err)
	}
	db.Close()

	if len(pkgs) < 2 {
		t.Skip("Not enough packages in system DB to test pruning")
	}

	// Keep all but the last one
	var keptPackages []string
	for i := 0; i < len(pkgs)-1; i++ {
		keptPackages = append(keptPackages, pkgs[i].Name)
	}
	droppedPackage := pkgs[len(pkgs)-1].Name

	// 5. Mock execCommand
	// Save original and restore after test
	origExec := execCommand
	defer func() { execCommand = origExec }()

	execCommand = func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	// 6. Run Reconstruct
	r := NewDefaultReconstructor(srcDir)
	plan := &types.FusionPlan{
		KeptPackages: keptPackages,
	}

	err = r.Reconstruct(srcDir, plan, outDir)
	assert.NoError(t, err)

	// 7. Verify output file exists
	assert.FileExists(t, filepath.Join(outDir, "rpmdb.sqlite"))
	
	// We can't verify content change because we mocked the pruning command,
	// but we verified the command was called successfully.
	t.Logf("Reconstruction successful, dropped package candidate: %s", droppedPackage)
}

// Simple copyFile helper for test setup (duplicates logic in main code but fine for test)
func copyFileTest(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
