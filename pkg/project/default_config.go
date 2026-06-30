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

import (
	"fmt"
	"os"
)

// WriteDefaultConfig writes a default config file for project workflows.
func WriteDefaultConfig(outputFileName, timezone, locale, imageType string) error {
	effectiveImageType, err := normalizeImageType(imageType)
	if err != nil {
		return err
	}
	if timezone == "" {
		timezone = DefaultTimezone
	}
	if locale == "" {
		locale = DefaultLocale
	}

	yamlContent := fmt.Sprintf(`packages:
  install_pkgs:
%s
network: 
    networking: yes
    hostname: localhost.localdomain
locale: "%%_install_langs %s"
timezone: "%s"
`, getPackagesByType(effectiveImageType), locale, timezone)
	if err := os.WriteFile(outputFileName, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %v", outputFileName, err)
	}
	return nil
}

// getPackagesByType returns the corresponding package list based on the type (in YAML content format)
func getPackagesByType(imageType string) string {
	var packages []string

	switch imageType {
	case "init":
		packages = []string{
			"yum",
			"vim-minimal",
			"dbus-daemon",
			"kbd",
			"util-linux",
		}
	case "platform":
		packages = []string{
			"yum",
			"vim-minimal",
			"shadow",
		}
	case "minimal":
		packages = []string{
			"microdnf",
			"vim-minimal",
		}
	case "micro":
		packages = []string{
			"coreutils",
		}
	default:
		packages = []string{
			"yum",
			"vim-minimal",
			"shadow",
		}
	}

	var packagesYAML string
	for _, pkg := range packages {
		packagesYAML += fmt.Sprintf("    - %s\n", pkg)
	}
	packagesYAML += "    # You can add more packages\n    # - package1\n    # - package2"

	return packagesYAML
}
