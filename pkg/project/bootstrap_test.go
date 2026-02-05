package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// fakeExecCommandBootstrap mocks exec.Command for testing
func fakeExecCommandBootstrap(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcessBootstrap", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcessBootstrap is the helper process for fakeExecCommand
func TestHelperProcessBootstrap(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Mock success exit code
	os.Exit(0)
}

func TestNewBootstrap(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want *Bootstrap
	}{
		{
			name: "Create new bootstrap",
			dir:  "/tmp/test-project",
			want: &Bootstrap{
				DestinationDir: "/tmp/test-project",
				BuildType:      "platform",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBootstrap(tt.dir)
			if got.DestinationDir != tt.want.DestinationDir {
				t.Errorf("NewBootstrap().DestinationDir = %v, want %v", got.DestinationDir, tt.want.DestinationDir)
			}
			if got.BuildType != tt.want.BuildType {
				t.Errorf("NewBootstrap().BuildType = %v, want %v", got.BuildType, tt.want.BuildType)
			}
		})
	}
}

func TestBootstrap_InitProjectStructure(t *testing.T) {
	type fields struct {
		BuildType string
	}
	tests := []struct {
		name    string
		fields  fields
		setup   func(string) error // Setup function to create conflicting files
		wantErr bool
	}{
		{
			name:   "Initialize clean directory",
			fields: fields{BuildType: "platform"},
			setup:  nil,
		},
		{
			name:   "Handle existing file conflict with rename",
			fields: fields{BuildType: "platform"},
			setup: func(dir string) error {
				// Create a file named 'rootfs' where a directory should be
				return os.WriteFile(filepath.Join(dir, "rootfs"), []byte("conflict"), 0644)
			},
		},
		{
			name:   "Idempotent execution",
			fields: fields{BuildType: "platform"},
			setup: func(dir string) error {
				// Create the directory structure beforehand
				return os.MkdirAll(filepath.Join(dir, "rootfs"), 0755)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for each test
			tmpDir := t.TempDir()
			b := &Bootstrap{
				DestinationDir: tmpDir,
				BuildType:      tt.fields.BuildType,
			}

			if tt.setup != nil {
				if err := tt.setup(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			if err := b.InitProjectStructure(); (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap.InitProjectStructure() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify directory structure
			dirs := []string{"dockerfile", "rootfs", "files", "tests"}
			for _, dir := range dirs {
				path := filepath.Join(tmpDir, dir)
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Directory %s was not created", dir)
					continue
				}
				if !info.IsDir() {
					t.Errorf("%s is not a directory", dir)
				}
			}

			// Verify key files exist
			files := []string{
				"dockerfile/Dockerfile",
				"files/removeminimallist",
				"files/unmaskService",
				"README.md",
			}
			for _, file := range files {
				path := filepath.Join(tmpDir, file)
				if _, err := os.Stat(path); err != nil {
					t.Errorf("File %s was not created", file)
				}
			}

			// Verify backup creation if conflict existed
			if tt.name == "Handle existing file conflict with rename" {
				bakPath := filepath.Join(tmpDir, "rootfs.bak")
				if _, err := os.Stat(bakPath); err != nil {
					t.Errorf("Backup file rootfs.bak was not created")
				}
			}
		})
	}
}

func TestBootstrap_InitWorkDir(t *testing.T) {
	tests := []struct {
		name      string
		types     string
		wantDir   string
	}{
		{
			name:      "Init baseimage workdir",
			types:     "baseimage",
			wantDir:   "init/baseimage",
		},
		{
			name:      "Init appimage workdir",
			types:     "appimage",
			wantDir:   "init/appimage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			b := &Bootstrap{
				DestinationDir: tmpDir,
			}

			b.InitWorkDir(tt.types, "")

			path := filepath.Join(tmpDir, tt.wantDir)
			info, err := os.Stat(path)
			if err != nil {
				t.Errorf("Directory %s was not created", tt.wantDir)
				return
			}
			if !info.IsDir() {
				t.Errorf("%s is not a directory", tt.wantDir)
			}
		})
	}
}

func TestBootstrap_AddMethods(t *testing.T) {
	tmpDir := t.TempDir()
	b := &Bootstrap{
		DestinationDir: tmpDir,
		BuildType:      "platform",
	}

	// Helper to verify file creation
	verifyFile := func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("File %s not created", path)
			return
		}
		if info.Size() == 0 {
			t.Errorf("File %s is empty", path)
		}
	}

	// Prepare directories
	os.MkdirAll(filepath.Join(tmpDir, "dockerfile"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "files"), 0755)

	t.Run("AddDockerfile", func(t *testing.T) {
		b.AddDockerfile()
		verifyFile(filepath.Join(tmpDir, "dockerfile", "Dockerfile"))
	})

	t.Run("AddRemoveMinimalList", func(t *testing.T) {
		b.AddRemoveMinimalList()
		verifyFile(filepath.Join(tmpDir, "files", "removeminimallist"))
	})

	t.Run("AddUnmaskService", func(t *testing.T) {
		b.AddUnmaskService()
		verifyFile(filepath.Join(tmpDir, "files", "unmaskService"))
	})

	t.Run("AddChangeInfo", func(t *testing.T) {
		b.AddChangeInfo()
		verifyFile(filepath.Join(tmpDir, "README.md"))
	})
}

func TestBootstrap_CleanRootfs(t *testing.T) {
	// Mock execCommand
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()
	execCommand = fakeExecCommandBootstrap

	tests := []struct {
		name      string
		buildType string
		setup     func(string) error
		wantErr   bool
	}{
		{
			name:      "Rootfs missing",
			buildType: "minimal",
			setup:     func(dir string) error { return nil }, // No rootfs
			wantErr:   true,
		},
		{
			name:      "Clean successful (Mock)",
			buildType: "minimal",
			setup:     func(dir string) error {
				// Setup rootfs
				rootfs := filepath.Join(dir, "rootfs")
				if err := os.MkdirAll(rootfs, 0755); err != nil {
					return err
				}
				// Setup files dir and config files required by CleanRootfs
				filesDir := filepath.Join(dir, "files")
				if err := os.MkdirAll(filesDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(filesDir, "removeminimallist"), []byte("pkg1\npkg2"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(filesDir, "unmaskService"), []byte("service1"), 0644); err != nil {
					return err
				}
				// Setup directories to be cleaned
				if err := os.MkdirAll(filepath.Join(rootfs, "var/log"), 0755); err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Join(rootfs, "tmp"), 0755); err != nil {
					return err
				}
				return nil
			},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip on Windows if running actual privileged operations, but here we mock execCommand.
			// However, CleanRootfs calls os.Geteuid() which returns -1 on Windows.
			// The check 'if os.Geteuid() != 0' will pass (fail the check) on Windows.
			// We can't mock os.Geteuid easily.
			if os.Geteuid() != 0 && tt.name != "Rootfs missing" {
				t.Skip("Skipping test requiring root privileges (os.Geteuid() != 0)")
			}

			tmpDir := t.TempDir()
			b := &Bootstrap{
				DestinationDir: tmpDir,
				BuildType:      tt.buildType,
			}

			if tt.setup != nil {
				if err := tt.setup(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			err := b.CleanRootfs()
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap.CleanRootfs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBootstrap_initialize_Failures(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		setup    func(dir string) error
		cleanup  func(dir string)
		check    func(t *testing.T, dir, fileName string)
	}{
		{
			name:     "File already exists",
			fileName: "exists.txt",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("original"), 0644)
			},
			check: func(t *testing.T, dir, fileName string) {
				content, err := os.ReadFile(filepath.Join(dir, fileName))
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(content) != "original" {
					t.Errorf("File content changed, want 'original', got %q", string(content))
				}
			},
		},
		{
			name:     "Create failure (directory exists)",
			fileName: "fail_dir",
			setup: func(dir string) error {
				// Create a directory with the same name to cause creation failure
				return os.Mkdir(filepath.Join(dir, "fail_dir"), 0755)
			},
			check: func(t *testing.T, dir, fileName string) {
				path := filepath.Join(dir, fileName)
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Path %s should exist", path)
				}
				if !info.IsDir() {
					t.Errorf("Path %s should be a directory", path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setup != nil {
				if err := tt.setup(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			if tt.cleanup != nil {
				defer tt.cleanup(tmpDir)
			}

			b := &Bootstrap{
				DestinationDir: tmpDir,
				BuildType:      "platform",
			}

			// Try to initialize a file with "new content"
			b.initialize("new content", tt.fileName, 0644)

			if tt.check != nil {
				tt.check(t, tmpDir, tt.fileName)
			}
		})
	}
}
