/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package project

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func TestCreateTarFromDirectory(t *testing.T) {
	t.Parallel()

	type args struct {
		setupFiles func(dir string) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Basic file structure",
			args: args{
				setupFiles: func(dir string) error {
					// Create regular file
					if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1"), 0644); err != nil {
						return err
					}
					// Create subdirectory
					subDir := filepath.Join(dir, "subdir")
					if err := os.Mkdir(subDir, 0755); err != nil {
						return err
					}
					// Create file in subdirectory
					return os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644)
				},
			},
			wantErr: false,
		},
		{
			name: "Empty directory",
			args: args{
				setupFiles: func(dir string) error {
					return nil
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Setup source directory
			srcDir := t.TempDir()
			if tt.args.setupFiles != nil {
				require.NoError(t, tt.args.setupFiles(srcDir))
			}

			// Setup destination tar file
			destDir := t.TempDir()
			tarPath := filepath.Join(destDir, "output.tar")

			err := createTarFromDirectory(srcDir, tarPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("createTarFromDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify tar file exists
				require.FileExists(t, tarPath)

				// Verify tar content
				f, err := os.Open(tarPath)
				require.NoError(t, err)
				defer f.Close()

				tr := tar.NewReader(f)
				foundFiles := make(map[string]string)

				for {
					header, err := tr.Next()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)

					if header.Typeflag == tar.TypeReg {
						buf := new(strings.Builder)
						_, err := io.Copy(buf, tr)
						require.NoError(t, err)
						foundFiles[header.Name] = buf.String()
					}
				}

				// Basic verification for "Basic file structure" case
				if tt.name == "Basic file structure" {
					assert.Contains(t, foundFiles, "file1.txt")
					assert.Equal(t, "content1", foundFiles["file1.txt"])

					// path might vary slightly depending on OS separator, but we check partial match
					foundSubFile := false
					for name, content := range foundFiles {
						if strings.Contains(name, "file2.txt") {
							assert.Equal(t, "content2", content)
							foundSubFile = true
						}
					}
					assert.True(t, foundSubFile, "Should find file2.txt in tar")
				}
			}
		})
	}
}

func TestCreateDefaultDockerfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Create dockerfile in clean directory",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()

			err := createDefaultDockerfile(projectDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("createDefaultDockerfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				dockerfilePath := filepath.Join(projectDir, "dockerfile", "Dockerfile")
				assert.FileExists(t, dockerfilePath)

				content, err := os.ReadFile(dockerfilePath)
				require.NoError(t, err)

				expected := `FROM scratch
ADD rootfs.tar /
CMD ["/bin/bash"]
`
				assert.Equal(t, expected, string(content))
			}
		})
	}
}

func TestBuildImage(t *testing.T) {
	// Cannot parallelize easily due to BuildImage potentially using global state or complex dependencies

	tests := []struct {
		name           string
		setupProject   func(dir string) error
		expectedErrStr string // Substring to match in error
	}{
		{
			name: "Missing rootfs directory",
			setupProject: func(dir string) error {
				// Do not create rootfs
				return nil
			},
			expectedErrStr: "rootfs directory does not exist",
		},
		{
			name: "Rootfs exists, proceed to build (fail at buildah/store)",
			setupProject: func(dir string) error {
				// Create rootfs
				if err := os.Mkdir(filepath.Join(dir, "rootfs"), 0755); err != nil {
					return err
				}
				// Create dummy file in rootfs
				if err := os.WriteFile(filepath.Join(dir, "rootfs", "dummy"), []byte("dummy"), 0644); err != nil {
					return err
				}

				// Create files directory (optional, to test file copying logic)
				if err := os.Mkdir(filepath.Join(dir, "files"), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "files", "extra.txt"), []byte("extra"), 0644); err != nil {
					return err
				}

				return nil
			},
			// We expect it to pass validation and file creation, but fail at "failed to resolve Dockerfile" or "failed to get store"
			// depending on the environment. We just ensure it is NOT the rootfs error.
			expectedErrStr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir := t.TempDir()
			require.NoError(t, tt.setupProject(destDir))

			b := &Bootstrap{
				DestinationDir: destDir,
				BuildType:      "platform",
			}

			err := b.BuildImage("test-image", "latest")

			if tt.expectedErrStr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrStr)
			} else {
				// For the "Pass validation" case
				if err != nil {
					// It failed, but verify it wasn't the rootfs error
					assert.NotContains(t, err.Error(), "rootfs directory does not exist")
					t.Logf("Got expected downstream error: %v", err)
				}
			}
		})
	}
}

func TestStripInlineComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"无注释", "en_US.UTF-8", "en_US.UTF-8"},
		{"行内注释", "en_US.UTF-8  # comment after", "en_US.UTF-8"},
		{"注释前无空格", "en_US.UTF-8# comment", "en_US.UTF-8"},
		{"等号后的值带注释", "en_US.UTF-8   # some explanation", "en_US.UTF-8"},
		{"单引号保护 #", "'en_US.UTF-8#safe'", "'en_US.UTF-8#safe'"},
		{"双引号保护 #", "\"en_US.UTF-8#safe\"", "\"en_US.UTF-8#safe\""},
		{"值本身就含 #（无引号）", "en_US.UTF-8#bad", "en_US.UTF-8"},
		{"多空格后注释", "  en_US.UTF-8  #  comment  ", "  en_US.UTF-8"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stripInlineComment(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveLocaleFromRootfs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantLocale string
	}{
		{
			name:       "标准格式",
			content:    "LANG=en_US.UTF-8",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "等号前有空格",
			content:    "LANG = en_US.UTF-8",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "等号前有制表符",
			content:    "LANG\t=\ten_US.UTF-8",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "双引号包裹的值",
			content:    `LANG="en_US.UTF-8"`,
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "单引号包裹的值",
			content:    `LANG='en_US.UTF-8'`,
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "值带行内注释",
			content:    "LANG=en_US.UTF-8  # zh_CN.UTF-8 is default",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "值带行内注释（等号前空格）",
			content:    "LANG = en_US.UTF-8  # comment",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "文件不存在",
			content:    "__FILE_NOT_CREATED__",
			wantLocale: "",
		},
		{
			name:       "文件为空",
			content:    "",
			wantLocale: "",
		},
		{
			name:       "无 LANG 行",
			content:    "LC_ALL=en_US.UTF-8\nLC_TIME=C",
			wantLocale: "",
		},
		{
			name:       "LANG 被注释掉",
			content:    "# LANG=en_US.UTF-8",
			wantLocale: "",
		},
		{
			name:       "多行取第一个 LANG",
			content:    "LC_ALL=C\nLANG=zh_CN.UTF-8\nLANG=en_US.UTF-8",
			wantLocale: "%_install_langs zh_CN.UTF-8",
		},
		{
			name:       "带多余空行",
			content:    "\n\nLANG=en_US.UTF-8\n\n",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
		{
			name:       "引号内的 # 保留",
			content:    `LANG="en_US.UTF-8#safe"`,
			wantLocale: "%_install_langs en_US.UTF-8#safe",
		},
		{
			name:       "CRLF 行尾",
			content:    "LANG=en_US.UTF-8\r\nLC_ALL=C\r\n",
			wantLocale: "%_install_langs en_US.UTF-8",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootfsDir := t.TempDir()
			if tt.content != "__FILE_NOT_CREATED__" {
				localeConfDir := filepath.Join(rootfsDir, "etc")
				require.NoError(t, os.MkdirAll(localeConfDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(localeConfDir, "locale.conf"), []byte(tt.content), 0644))
			}

			b := &Bootstrap{Locale: ""}
			b.resolveLocaleFromRootfs(rootfsDir)

			if tt.wantLocale == "" {
				assert.Empty(t, b.Locale, "预期 Locale 为空，得到 %q", b.Locale)
			} else {
				assert.Equal(t, tt.wantLocale, b.Locale)
			}
		})
	}
}

