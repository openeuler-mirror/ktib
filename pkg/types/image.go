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
	"github.com/opencontainers/go-digest"
	"time"
)

type JsonImage struct {
	Name    []string  `json:"name"`
	Digest  digest.Digest    `json:"digest"`
	ImageID string    `json:"images ID"`
	Created time.Time `json:"created"`
	Size    int64     `json:"size"`
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
