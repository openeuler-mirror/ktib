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

package types

import "time"

// AnalysisReport is the top-level structure for the analysis report
type AnalysisReport struct {
	ImageInfo       ImageInfo        `json:"image_info"`
	Analysis        AnalysisData     `json:"analysis"`
	Recommendations []Recommendation `json:"recommendations"`
}

// ImageInfo contains basic metadata about the analyzed image
type ImageInfo struct {
	Ref          string      `json:"ref"`
	Size         int64       `json:"size"`
	OS           string      `json:"os"`
	Created      time.Time   `json:"created"`
	Architecture string      `json:"architecture,omitempty"`
	Config       ImageConfig `json:"config,omitempty"`
}

// ImageConfig contains configuration from the image
type ImageConfig struct {
	Cmd        []string `json:"cmd"`
	Entrypoint []string `json:"entrypoint"`
	Env        []string `json:"env"`
	WorkingDir string   `json:"working_dir"`
}

// AnalysisData aggregates all detailed analysis results
type AnalysisData struct {
	Layers         []LayerInfo    `json:"layers"`
	Packages       PackageInfo    `json:"packages"`
	Filesystem     FilesystemInfo `json:"filesystem"`
	WasteDetection WasteDetection `json:"waste_detection"`
	ELFMetadata    ELFMetadata    `json:"elf_metadata,omitempty"`
}

// ELFMetadata contains dependency information for ELF files
type ELFMetadata struct {
	// Dependencies maps a file path (inside container) to a list of resolved library paths (inside container)
	Dependencies map[string][]string `json:"dependencies"`
	// Libs contains all shared libraries found in the image with their sizes
	Libs []File `json:"libs,omitempty"`
}

// LayerInfo describes a single image layer
type LayerInfo struct {
	Index            int    `json:"index"`
	Digest           string `json:"digest"`
	Size             int64  `json:"size"`
	Command          string `json:"command"`
	AddedFileCount   int    `json:"added_file_count"`
	DeletedFileCount int    `json:"deleted_file_count"`
	TopFiles         []File `json:"top_files"`
}

// File represents a file entry with path and size
type File struct {
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	LinkTarget string `json:"link_target,omitempty"`
}

// PackageInfo contains lists of installed packages
type PackageInfo struct {
	RPM    []Package `json:"rpm"`
	Python []Package `json:"python"`
}

// Package describes a single software package
type Package struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Size     int64    `json:"size"`
	License  string   `json:"license,omitempty"`
	Digest   string   `json:"digest,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Provides []string `json:"provides,omitempty"`
	Files    []string `json:"files,omitempty"` // Files installed by the package
}

// FilesystemInfo contains filesystem statistics
type FilesystemInfo struct {
	TopDirectories []TopDirectory `json:"top_directories"`
	FileTypes      []FileTypeInfo `json:"file_types"`
}

// TopDirectory represents a directory consuming significant space
type TopDirectory struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// FileTypeInfo aggregates statistics by file type
type FileTypeInfo struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Size  int64  `json:"size"`
}

// WasteDetection contains information about potential waste
type WasteDetection struct {
	Duplicates []DuplicateFile `json:"duplicates"`
	Caches     []File          `json:"caches"`
}

// DuplicateFile represents a file that exists in multiple layers or locations
type DuplicateFile struct {
	Path        string   `json:"path"`
	Size        int64    `json:"size"`
	LayerDigest []string `json:"layer_digest,omitempty"` // Layers where this file appears
}

// Recommendation represents an actionable optimization suggestion
type Recommendation struct {
	Level        string   `json:"level"` // e.g., WARN, INFO
	Code         string   `json:"code"`  // e.g., RM_CACHE
	Message      string   `json:"message"`
	Saving       string   `json:"saving"`
	Command      string   `json:"command,omitempty"`
	MatchedItems []string `json:"matched_items,omitempty"`
}
