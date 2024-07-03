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

package utils

import (
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"syscall"
)

func ReexecInit() bool {
	return reexec.Init()
}

func check() {
	oldUmask := syscall.Umask(0o022) //nolint
	if (oldUmask & ^0o022) != 0 {
		logrus.Debugf("umask value too restrictive.  Forcing it to 022")
	}
}

func GetStore(c *cobra.Command) (storage.Store, error) {
	// 下面为获取option默认方法，注意需考虑options其他属性是否是必须的，在下面进行展开
	options, err := storage.DefaultStoreOptions(unshare.GetRootlessUID() > 0, unshare.GetRootlessUID())
	// TODO 判断参数common-builders 例如：storage-dirver storage-opt root 后续支持

	// umask check force on 022
	check()
	// get store object
	store, err := storage.GetStore(options)
	return store, err
}
