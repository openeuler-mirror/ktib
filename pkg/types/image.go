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
	// 如果 aux 列表不为空，处理附加注释
	if len(aux) != 0 {
		// 创建一个新的映射来存储附加注释
		auxAnnotations := make(map[string]string)
		for _, annotationSpec := range aux {
			// 分割注释字符串为键值对
			key, val, hasVal := strings.Cut(annotationSpec, "=")
			if !hasVal {
				return nil, fmt.Errorf("no value given for annotation %q", key)
			}
			auxAnnotations[key] = val
		}

		// 如果 preferred 为空，初始化为一个新的映射
		if preferred == nil {
			preferred = make(map[string]string)
		}

		// 合并 auxAnnotations 和 preferred 映射
		// 克隆 preferred 映射，以免修改原始映射
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
