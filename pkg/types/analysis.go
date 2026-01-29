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
	Ref     string    `json:"ref"`
	Size    int64     `json:"size"`
	OS      string    `json:"os"`
	Created      time.Time `json:"created"`
	Architecture string    `json:"architecture,omitempty"`
}

// AnalysisData aggregates all detailed analysis results
type AnalysisData struct {
	Layers         []LayerInfo    `json:"layers"`
	Packages       PackageInfo    `json:"packages"`
	Filesystem     FilesystemInfo `json:"filesystem"`
	WasteDetection WasteDetection `json:"waste_detection"`
}

// LayerInfo describes a single image layer
type LayerInfo struct {
	Index          int    `json:"index"`
	Digest         string `json:"digest"`
	Size           int64  `json:"size"`
	Command          string `json:"command"`
	AddedFileCount   int    `json:"added_file_count"`
	DeletedFileCount int    `json:"deleted_file_count"`
	TopFiles         []File `json:"top_files"`
}

// File represents a file entry with path and size
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// PackageInfo contains lists of installed packages
type PackageInfo struct {
	RPM    []Package `json:"rpm"`
	Python []Package `json:"python"`
}

// Package describes a single software package
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Size    int64  `json:"size"`
	License string `json:"license,omitempty"`
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
	Level   string `json:"level"` // e.g., WARN, INFO
	Code    string `json:"code"`  // e.g., RM_CACHE
	Message string `json:"message"`
	Saving  string `json:"saving"`
}
