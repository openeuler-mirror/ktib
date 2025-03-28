#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package utils

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"strings"
	"testing"
)

func TestGetStore(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "test cmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			exec.Command("echo", "hello")
			return nil
		},
	}
	_, err := GetStore(cmd)
	if err != nil {
		if strings.Contains(err.Error(), "is not supported over overlayfs") {
			t.Skip("Skipping test due to unsupported overlay error.")
		}
		t.Fatalf("Error during Commit: %v", err)
	}
	assert.NoError(t, err)
}
