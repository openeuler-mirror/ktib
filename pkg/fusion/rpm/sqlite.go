//go:build cgo

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
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// Common RPM SQLite tables that use pkgKey as foreign key
var rpmTables = []string{
	"Packages",
	"Name",
	"Basenames",
	"Group",
	"Requirename",
	"Providename",
	"Conflictname",
	"Obsoletename",
	"Triggername",
	"Dirnames",
	"Filemd5s",
	"Sha1header",
	"Sigmd5",
	"Installtid",
	"Pubkeys", // Sometimes exists
}

// pruneSQLiteDB removes packages not in the keep list from the SQLite database
func pruneSQLiteDB(dbPath string, keptPackages []string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite db: %w", err)
	}
	defer db.Close()

	// 1. Get all package names and their keys
	// The 'Name' table maps pkgKey -> Name (or Name -> pkgKey depending on schema view, but usually it has both columns)
	// Schema: CREATE TABLE "Name" ( "name" TEXT, "pkgKey" INTEGER );
	rows, err := db.Query("SELECT name, pkgKey FROM Name")
	if err != nil {
		return fmt.Errorf("failed to query Name table: %w", err)
	}
	defer rows.Close()

	keptSet := make(map[string]bool)
	for _, p := range keptPackages {
		keptSet[p] = true
	}

	var keysToDelete []int
	var namesToDelete []string

	for rows.Next() {
		var name string
		var key int
		if err := rows.Scan(&name, &key); err != nil {
			return fmt.Errorf("failed to scan Name row: %w", err)
		}
		if !keptSet[name] {
			keysToDelete = append(keysToDelete, key)
			namesToDelete = append(namesToDelete, name)
		}
	}
	rows.Close()

	if len(keysToDelete) == 0 {
		logrus.Info("No packages to prune.")
		return nil
	}

	logrus.Infof("Pruning %d packages from RPM DB...", len(keysToDelete))
	logrus.Debugf("Packages to delete: %v", namesToDelete)

	// 2. Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 3. Delete from all tables
	// We do this in batches to avoid "too many SQL variables" error
	batchSize := 500
	for i := 0; i < len(keysToDelete); i += batchSize {
		end := i + batchSize
		if end > len(keysToDelete) {
			end = len(keysToDelete)
		}
		batch := keysToDelete[i:end]

		placeholders := make([]string, len(batch))
		args := make([]interface{}, len(batch))
		for j, k := range batch {
			placeholders[j] = "?"
			args[j] = k
		}
		queryIn := fmt.Sprintf("(%s)", strings.Join(placeholders, ","))

		for _, table := range rpmTables {
			// Check if table exists first to avoid errors on different RPM versions
			if !tableExists(tx, table) {
				continue
			}

			// Special case: Packages table usually has 'pkgKey' as PRIMARY KEY, but sometimes it's implied rowid.
			// In standard RPM SQLite: CREATE TABLE "Packages" ( "blob" BLOB, "pkgKey" INTEGER PRIMARY KEY );
			// Other tables have "pkgKey" column.
			
			q := fmt.Sprintf("DELETE FROM %s WHERE pkgKey IN %s", table, queryIn)
			if _, err := tx.Exec(q, args...); err != nil {
				return fmt.Errorf("failed to delete from %s: %w", table, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// 4. Vacuum to reclaim space
	logrus.Info("Vacuuming RPM DB...")
	if _, err := db.Exec("VACUUM"); err != nil {
		logrus.Warnf("Failed to vacuum DB: %v", err)
	}

	return nil
}

func tableExists(tx *sql.Tx, tableName string) bool {
	var name string
	err := tx.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	return err == nil
}
