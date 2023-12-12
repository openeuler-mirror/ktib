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

package app

import "gitee.com/openeuler/ktib/cmd/ktib/app/utils"

func Run() error {
	//A library subroutine needed to run a subprocess.
	//So reexec.Init() should be called in main()
	if utils.ReexecInit() {
		return nil
	}
	cmd := NewCommand()
	cmd.CompletionOptions.DisableDefaultCmd = true
	return cmd.Execute()
}
