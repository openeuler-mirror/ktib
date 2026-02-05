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

package verify

import "testing"

func TestSummarizeLines(t *testing.T) {
	total, sample := summarizeLines("a\nb\nc\n", 2)
	if total != 3 {
		t.Fatalf("expected total=3, got %d", total)
	}
	if sample != "a | b" {
		t.Fatalf("unexpected sample=%q", sample)
	}
}

func TestSummarizeLinesEmpty(t *testing.T) {
	total, sample := summarizeLines("\n", 5)
	if total != 0 {
		t.Fatalf("expected total=0, got %d", total)
	}
	if sample != "" {
		t.Fatalf("expected empty sample, got %q", sample)
	}
}

