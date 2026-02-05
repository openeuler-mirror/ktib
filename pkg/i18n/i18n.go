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

package i18n

import "sync"

var (
	currentLang string = "en"
	mu          sync.RWMutex
)

// SetLanguage sets the global language
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	currentLang = lang
}

// T translates a key to the current language
func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if currentLang == "zh" || currentLang == "zh_cn" || currentLang == "zh_CN" {
		if val, ok := zhCN[key]; ok {
			return val
		}
	}
	return key
}

var zhCN = map[string]string{
	// Analyze Report Headers
	"IMAGE ANALYSIS SUMMARY": "镜像分析摘要",
	"Image Ref:":             "镜像引用:",
	"Architecture:":          "架构:",
	"OS:":                    "操作系统:",
	"Total Size:":            "总大小:",
	"CONTENT STATS":          "内容统计",
	"Layers:":                "层数:",
	"RPM Packages:":          "RPM 包数量:",
	"Python Packages:":       "Python 包数量:",
	"Potential Waste:":       "潜在浪费:",
	"Image Efficiency:":      "镜像效率:",
	"RECOMMENDATIONS":        "优化建议",
	"LEVEL":                  "级别",
	"ID":                     "ID",
	"SAVINGS":                "节省空间",
	"DESCRIPTION":            "描述",
	"COMMAND":                "建议操作",
	"Tip: Use '-o json' or '-f <file>' for detailed report.": "提示: 使用 '-o json' 或 '-f <file>' 获取详细报告。",

	// Fusion Phases
	"Phase 1: Solving dependencies":        "阶段 1: 解析依赖",
	"Phase 2: Synthesizing Filesystem":     "阶段 2: 合成文件系统",
	"Phase 3: Reconstructing RPM Database": "阶段 3: 重建 RPM 数据库",
	"Phase 4: Verifying result":            "阶段 4: 验证结果",
	"Phase 5: Committing to new image":     "阶段 5: 提交新镜像",
}
