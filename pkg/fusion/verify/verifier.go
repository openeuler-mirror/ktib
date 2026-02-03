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

import (
	"github.com/sirupsen/logrus"
)

// DefaultVerifier is a stub implementation of Verifier
type DefaultVerifier struct{}

// NewDefaultVerifier creates a new DefaultVerifier
func NewDefaultVerifier() *DefaultVerifier {
	return &DefaultVerifier{}
}

// Verify checks the integrity and usability of the fused image
func (v *DefaultVerifier) Verify(rootfsPath string) error {
	logrus.Infof("Verifying rootfs at %s", rootfsPath)
	// TODO: Implement chroot/container verification
	return nil
}
