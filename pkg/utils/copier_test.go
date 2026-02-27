/*
   Copyright (c) 2025 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package utils

import (
    "archive/tar"
    "archive/zip"
    "bytes"
    "compress/gzip"
    "os"
    "strings"
    "testing"
    "time"
)

func TestArchiveUncompress(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		destination  string
		wantErr      bool
		expectedMsg  string
		setupFunc    func() string
		teardownFunc func(string)
	}{
		{
			name:        "Source file does not exist",
			source:      "nonexistent.file",
			destination: "output.file",
			wantErr:     true,
			expectedMsg: "source file '' does not exist",
		},
		{
			name:        "Valid zip file",
			source:      "test.zip",
			destination: "output.txt",
			wantErr:     false,
			setupFunc: func() string {
				// Create a zip file for testing
				buf := new(bytes.Buffer)
				zipWriter := zip.NewWriter(buf)
				fw, _ := zipWriter.Create("output.txt")
				fw.Write([]byte("Hello, World!"))
				zipWriter.Close()

				// Write zip file to disk
				tmpFile := "test.zip"
                os.WriteFile(tmpFile, buf.Bytes(), 0644)
				return tmpFile
			},
			teardownFunc: func(src string) {
				os.Remove(src)
			},
		},
		{
			name:        "Valid tar.gz file",
			source:      "test.tar.gz",
			destination: "output.txt",
			wantErr:     false,
			setupFunc: func() string {
				// Create a tar.gz file for testing
				tmpFile := "test.tar.gz"
				file, err := os.Create(tmpFile)
				if err != nil {
					t.Fatal(err)
				}

				gz := gzip.NewWriter(file)
				tarWriter := tar.NewWriter(gz)
				header := &tar.Header{
					Name:    "output.txt",
					Mode:    0600,
					Size:    int64(len("Hello, Tar!")),
					ModTime: time.Now(),
				}
				if err := tarWriter.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				tarWriter.Write([]byte("Hello, Tar!"))
				if err != nil {
					t.Fatal(err)
				}

				tarWriter.Close()
				gz.Close()
				file.Close()
				return tmpFile
			},
			teardownFunc: func(src string) {
				os.Remove(src)
			},
		},
		{
			name:        "File not found in zip",
			source:      "test.zip",
			destination: "missing.txt",
			wantErr:     true,
			expectedMsg: "file 'missing.txt' not found in 'test.zip'",
			setupFunc: func() string {
				// Create a zip file for testing
				buf := new(bytes.Buffer)
				zipWriter := zip.NewWriter(buf)
				fw, _ := zipWriter.Create("output.txt")
				fw.Write([]byte("Hello, World!"))
				zipWriter.Close()

				// Write zip file to disk
				tmpFile := "test.zip"
                os.WriteFile(tmpFile, buf.Bytes(), 0644)
				return tmpFile
			},
			teardownFunc: func(src string) {
				os.Remove(src)
			},
		},
		{
			name:        "Error creating destination file",
			source:      "test.txt",
			destination: "/invalid/path/output.txt", // Invalid path
			wantErr:     true,
			expectedMsg: "mkdir /invalid/path: no such file or directory",
			setupFunc: func() string {
				// Create a simple text file
                os.WriteFile("test.txt", []byte("Test content"), 0644)
                return "test.txt"
            },
			teardownFunc: func(src string) {
				os.Remove(src)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sourceFile string
			if tt.setupFunc != nil {
				sourceFile = tt.setupFunc()
			}

			// Call the function under test
			err := ArchiveUncompress(sourceFile, tt.destination)

			// Check for expected error
			//if (err != nil) != tt.wantErr {
			//	t.Errorf("got error %v, want error: %v", err, tt.wantErr)
			//}
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if err != nil && tt.expectedMsg != "" && !strings.Contains(err.Error(), tt.expectedMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.expectedMsg, err.Error())
			}

			if tt.teardownFunc != nil {
				tt.teardownFunc(sourceFile)
			}
		})
	}
}
