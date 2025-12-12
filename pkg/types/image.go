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

import (
	"fmt"
	"github.com/opencontainers/go-digest"
	"strings"
	"time"
)

type JsonImage struct {
	Name    []string      `json:"name"`
	Digest  digest.Digest `json:"digest"`
	ImageID string        `json:"images ID"`
	Created time.Time     `json:"created"`
	Size    int64         `json:"size"`
}

type TableImage struct {
	Repository string
	Tag        string
	ImageID    string
	Created    string
	Size       string
	Digest     string
}

type JsonBuilder struct {
	ID      string    `json:"id"`
	Names   []string  `json:"names,omitempty"`
	ImageID string    `json:"imageID"`
	Created time.Time `json:"created,omitempty"`
	Mount   string    `json:"mount,omitempty"`
}

func MergeAnnotations(preferred map[string]string, aux []string) (map[string]string, error) {
	// If the aux list is not empty, process auxiliary annotations
	if len(aux) != 0 {
		// Create a new map to store auxiliary annotations
		auxAnnotations := make(map[string]string)
		for _, annotationSpec := range aux {
			// Split the annotation string into key-value pairs
			key, val, hasVal := strings.Cut(annotationSpec, "=")
			if !hasVal {
				return nil, fmt.Errorf("no value given for annotation %q", key)
			}
			auxAnnotations[key] = val
		}

		// If preferred is nil, initialize it as a new map
		if preferred == nil {
			preferred = make(map[string]string)
		}

		// Merge auxAnnotations and preferred map
		// Clone the preferred map to avoid modifying the original map
		merged := make(map[string]string, len(preferred)+len(auxAnnotations))
		for k, v := range preferred {
			merged[k] = v
		}
		for k, v := range auxAnnotations {
			merged[k] = v
		}

		preferred = merged
	}

	return preferred, nil
}
