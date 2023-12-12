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

package project

type Bootstrap struct {
	Type           string
	DestinationDir string
}

func NewBootstrap(types, dst string) *Bootstrap {
	return &Bootstrap{Type: types, DestinationDir: dst}
}

func (b *Bootstrap) AddDockerfile() {

}

func (b *Bootstrap) AddTestcase() {

}

func (b *Bootstrap) AddScript() {

}

func (b *Bootstrap) AddChangeInfo() {

}
