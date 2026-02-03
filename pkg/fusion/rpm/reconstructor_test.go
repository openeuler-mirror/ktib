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
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"gitee.com/openeuler/ktib/pkg/fusion/types"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestReconstructSQLite(t *testing.T) {
	// 1. Setup temporary source and output directories
	tmpDir, err := os.MkdirTemp("", "ktib-test-rpm-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "source")
	outDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 2. Create a mock RPM SQLite DB in source
	dbPath := filepath.Join(srcDir, "rpmdb.sqlite")
	
	// Check if sqlite3 driver is available
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Logf("Skipping SQLite test: %v", err)
		return
	}
	// If driver is not registered, sql.Open might return error or db.Ping will fail?
	// sql.Open usually doesn't error on unknown driver until used, or it returns "sql: unknown driver: sqlite3"
	if err := db.Ping(); err != nil {
		t.Logf("Skipping SQLite test (driver not working): %v", err)
		return
	}
	
	// Create minimal schema for testing
	_, err = db.Exec(`
		CREATE TABLE Packages (pkgKey INTEGER PRIMARY KEY, blob BLOB);
		CREATE TABLE Name (name TEXT, pkgKey INTEGER);
		CREATE TABLE Basenames (name TEXT, pkgKey INTEGER);
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert test data
	// Package A (Key 1) - KEEP
	// Package B (Key 2) - DROP
	// Package C (Key 3) - KEEP
	_, err = db.Exec(`
		INSERT INTO Packages (pkgKey, blob) VALUES (1, 'dataA'), (2, 'dataB'), (3, 'dataC');
		INSERT INTO Name (name, pkgKey) VALUES ('pkgA', 1), ('pkgB', 2), ('pkgC', 3);
		INSERT INTO Basenames (name, pkgKey) VALUES ('/bin/a', 1), ('/bin/b', 2), ('/bin/c', 3);
	`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// 3. Initialize Reconstructor
	r := NewDefaultReconstructor(srcDir)
	plan := &types.FusionPlan{
		KeptPackages: []string{"pkgA", "pkgC"},
	}

	// 4. Run Reconstruct
	err = r.Reconstruct(plan, outDir)
	assert.NoError(t, err)

	// 5. Verify Output DB
	targetDBPath := filepath.Join(outDir, "rpmdb.sqlite")
	assert.FileExists(t, targetDBPath)

	targetDB, err := sql.Open("sqlite3", targetDBPath)
	assert.NoError(t, err)
	defer targetDB.Close()

	// Check Packages count
	var count int
	err = targetDB.QueryRow("SELECT COUNT(*) FROM Packages").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // Should remain A and C

	// Check Name table
	rows, err := targetDB.Query("SELECT name FROM Name ORDER BY name")
	assert.NoError(t, err)
	var names []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		names = append(names, name)
	}
	rows.Close()
	assert.Equal(t, []string{"pkgA", "pkgC"}, names)

	// Check Basenames table (Cascade delete check)
	err = targetDB.QueryRow("SELECT COUNT(*) FROM Basenames WHERE pkgKey=2").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count) // pkgB's files should be gone
}

func TestReconstructUnsupportedFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ktib-test-rpm-bdb-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy BDB file
	f, err := os.Create(filepath.Join(tmpDir, "Packages"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	r := NewDefaultReconstructor(tmpDir)
	plan := &types.FusionPlan{}

	err = r.Reconstruct(plan, filepath.Join(tmpDir, "output"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BerkeleyDB (BDB) format is not supported")
}
