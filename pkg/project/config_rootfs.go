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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var unnecessaryFiles = []string{
	// **************locales**********************
	"/usr/lib/locale",
	"/usr/share/locale",
	"/lib/gconv",
	"/lib64/gconv",
	"/bin/localedef",
	"/sbin/build-locale-archive",
	//************docs and man pages**************
	"/usr/share/man",
	"/usr/share/doc",
	"/usr/share/info",
	"/usr/share/gnome/help",
	//**************profile.d**********************
	"/etc/profile.d/system-info.sh",
	//*****************i18n************************
	"/usr/share/i18n",
	//***************yum cache*********************
	"/var/cache/yum",
	//***************sln***************************
	"/sbin/sln",
	//*****************ldconfig********************
	"/var/cache/ldconfig",
	//**********other unnecessary files************8
	"/var/lib/dnf",
	"/run/nologin",
	"/var/log",
}

func ConfigureRootfs(target string, config Config) error {
	// 配置网络
	network := config.Network.NETWORKING
	hostname := config.Network.HOSTNAME
	networkConfig := fmt.Sprintf("NETWORKING=%s\nHOSTNAME=%s\n", network, hostname)
	networkFilePath := filepath.Join(target, "/etc/sysconfig/network")
	err := ioutil.WriteFile(networkFilePath, []byte(networkConfig), 0644)
	if err != nil {
		fmt.Printf("error writing network configuration: %v", err)
	}

	// 设置 DNF infra 变量
	infraConfig := "container"
	infraFilePath := filepath.Join(target, "/etc/dnf/vars/infra")
	// 确保目录存在
	os.MkdirAll(filepath.Dir(infraFilePath), 0755)
	err = ioutil.WriteFile(infraFilePath, []byte(infraConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing infra configuration: %v", err)
	}

	// 配置语言环境
	if config.Locale != "" {
		localeFilePath := filepath.Join(target, "/etc/rpm/macros.image-language-conf")
		// 确保目录存在
		os.MkdirAll(filepath.Dir(localeFilePath), 0755)
		err = ioutil.WriteFile(localeFilePath, []byte(config.Locale), 0644)
		if err != nil {
			return fmt.Errorf("error writing language configuration: %v", err)
		}

		// 设置系统语言环境
		localePath := filepath.Join(target, "/etc/locale.conf")
		// 从 config.Locale 中提取语言代码
		// 假设格式为 "%_install_langs en_US.UTF-8"
		localeParts := strings.Split(config.Locale, " ")
		localeValue := ""
		if len(localeParts) > 1 {
			localeValue = fmt.Sprintf("LANG=%s\n", localeParts[len(localeParts)-1])
		} else {
			localeValue = "LANG=en_US.UTF-8\n" // 默认值
		}

		// 确保目录存在
		os.MkdirAll(filepath.Dir(localePath), 0755)
		if err := ioutil.WriteFile(localePath, []byte(localeValue), 0644); err != nil {
			fmt.Printf("error writing locale.conf file: %v\n", err)
		}
	}

	// 配置时区
	if config.Timezone != "" {
		// 创建 /etc/localtime 软链接指向正确的时区文件
		timezonePath := filepath.Join("/usr/share/zoneinfo", config.Timezone)
		localtimePath := filepath.Join(target, "/etc/localtime")

		// 确保目标目录存在
		os.MkdirAll(filepath.Dir(localtimePath), 0755)

		// 创建软链接
		cmd := exec.Command("ln", "-sf", timezonePath, localtimePath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("error setting timezone: %v\n", err)
		}

		// 写入时区信息到 /etc/timezone
		timezoneFPath := filepath.Join(target, "/etc/timezone")
		if err := ioutil.WriteFile(timezoneFPath, []byte(config.Timezone), 0644); err != nil {
			fmt.Printf("error writing timezone file: %v\n", err)
		}
	}

	// force each container to have a unique machine-id
	machineId := ""
	machineIDFilePath := filepath.Join(target, "/etc/machine-id")
	err = ioutil.WriteFile(machineIDFilePath, []byte(machineId), 0644)
	if err != nil {
		return fmt.Errorf("error writing machine-id file: %v", err)
	}

	// 复制bash配置文件并设置bash历史
	if err := addCommandToScriptAndRun(target, config); err != nil {
		return fmt.Errorf("Error add command to script and run: %v\n", err)
	}
	return nil
}

func addCommandToScriptAndRun(target string, config Config) error {
	// 复制bash配置文件
	bashCmd := exec.Command("sh", "-c", fmt.Sprintf("cp /etc/skel/.bash* %s/root/", target))
	if err := bashCmd.Run(); err != nil {
		return fmt.Errorf("复制bash配置文件失败: %v", err)
	}

	// 创建空的bash历史文件
	historyPath := filepath.Join(target, "root", ".bash_history")
	if err := ioutil.WriteFile(historyPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("创建bash历史文件失败: %v", err)
	}

	return nil
}

func RemoveUnnecessaryFiles(target string) error {
	for _, i := range unnecessaryFiles {
		fmt.Println(i)
	}
	if err := removeAllFiles(target, unnecessaryFiles); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(target, "var/cache/yum"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(target, "/var/cache/ldconfig"), 0755); err != nil {
		return err
	}
	return nil
}

func removeAllFiles(target string, files []string) error {
	for _, file := range files {
		fmt.Println(filepath.Join(target, file))
		if err := os.RemoveAll(filepath.Join(target, file)); err != nil {
			return err
		}
	}
	return nil
}

// 添加以下函数来完善文件清理
func CleanupRootfsPath(target string) error {
	// 1. 清理RPM数据库历史记录
	rpmHistoryFiles, err := filepath.Glob(filepath.Join(target, "var/lib/dnf/history.*"))
	if err == nil && len(rpmHistoryFiles) > 0 {
		fmt.Println("清理RPM数据库历史记录...")
		for _, file := range rpmHistoryFiles {
			os.Remove(file)
		}
	}

	// 2. 清理临时文件和日志文件
	fmt.Println("清理临时文件和日志文件...")

	logDir := filepath.Join(target, "var/log")
	if _, err := os.Stat(logDir); !os.IsNotExist(err) {
		fmt.Printf("清空目录: %s\n", logDir)
		os.RemoveAll(logDir)
		os.MkdirAll(logDir, 0755)
	}

	tmpDir := filepath.Join(target, "tmp")
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		fmt.Printf("清空目录: %s\n", tmpDir)
		os.RemoveAll(tmpDir)
		os.MkdirAll(tmpDir, 0755)
	}

	// 3. 删除nologin文件
	nologinFile := filepath.Join(target, "run/nologin")
	if _, err := os.Stat(nologinFile); !os.IsNotExist(err) {
		fmt.Printf("删除文件: %s\n", nologinFile)
		os.Remove(nologinFile)
	}

	// 4. 清理bash历史
	bashHistoryPath := filepath.Join(target, "root/.bash_history")
	if _, err := os.Stat(bashHistoryPath); !os.IsNotExist(err) {
		fmt.Printf("清空文件: %s\n", bashHistoryPath)
		ioutil.WriteFile(bashHistoryPath, []byte(""), 0644)
	}

	return nil
}

// 添加以下函数来移除不必要的包
// 修改函数，接受文件路径参数
func RemoveUnnecessaryPackages(target string, imageType string, removeListPath, removeMinimalListPath string) error {
	var packagesToRemove []string
	var err error
	var data []byte

	// 检查是否有root权限
	if os.Geteuid() != 0 {
		return fmt.Errorf("需要root权限执行chroot命令")
	}

	// 根据镜像类型选择要移除的包列表
	if imageType == "minimal" {
		// 读取 removeminimallist 文件
		data, err = ioutil.ReadFile(removeMinimalListPath)
		if err != nil {
			return fmt.Errorf("无法读取 removeminimallist 文件: %v", err)
		}
	} else if imageType != "micro" {
		// 读取 removelist 文件
		data, err = ioutil.ReadFile(removeListPath)
		if err != nil {
			return fmt.Errorf("无法读取 removelist 文件: %v", err)
		}
	} else {
		// micro 类型不需要移除包
		return nil
	}

	packagesToRemove = strings.Split(string(data), "\n")

	// 检查是否有包需要移除
	hasPackagesToRemove := false
	for _, pkg := range packagesToRemove {
		pkg = strings.TrimSpace(pkg)
		if pkg != "" && !strings.HasPrefix(pkg, "#") {
			hasPackagesToRemove = true
			break
		}
	}

	if !hasPackagesToRemove {
		fmt.Println("没有需要移除的软件包")
		return nil
	}

	// 创建移除包的脚本
	scriptContent := "#!/bin/bash\n"
	scriptContent += "set -e\n" // 遇到错误立即退出
	scriptContent += "echo '开始移除不必要的软件包...'\n"

	for _, pkg := range packagesToRemove {
		pkg = strings.TrimSpace(pkg)
		if pkg != "" && !strings.HasPrefix(pkg, "#") {
			// 先检查包是否已安装
			scriptContent += fmt.Sprintf("if rpm -q %s &>/dev/null; then\n", pkg)
			scriptContent += fmt.Sprintf("  echo '移除软件包: %s'\n", pkg)
			scriptContent += fmt.Sprintf("  rpm -e --nodeps %s || echo '警告: 无法移除 %s'\n", pkg, pkg)
			scriptContent += "fi\n"
		}
	}

	scriptContent += "echo '软件包移除完成'\n"

	// 使用绝对路径
	scriptPath := filepath.Join(target, "remove_packages.sh")
	if err := ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("无法创建移除软件包脚本: %v", err)
	}

	fmt.Println("执行软件包移除脚本...")

	// 执行脚本
	cmd := exec.Command("chroot", target, "/remove_packages.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	// 清理脚本
	os.Remove(scriptPath)

	if err != nil {
		return fmt.Errorf("执行移除软件包脚本失败: %v", err)
	}

	return nil
}

// 修改函数，接受文件路径参数
func UnmaskServices(target string, unmaskServicePath string) error {
	// 检查是否有root权限
	if os.Geteuid() != 0 {
		return fmt.Errorf("需要root权限执行chroot命令")
	}

	// 读取 unmaskService 文件
	data, err := ioutil.ReadFile(unmaskServicePath)
	if err != nil {
		return fmt.Errorf("无法读取 unmaskService 文件: %v", err)
	}

	// 检查文件内容是否为空
	if len(strings.TrimSpace(string(data))) == 0 {
		fmt.Println("unmaskService文件为空，跳过解除服务屏蔽")
		return nil
	}

	// 创建解除屏蔽服务的脚本
	scriptPath := filepath.Join(target, "unmask_services.sh")

	// 添加脚本头和错误处理
	scriptContent := "#!/bin/bash\n"
	scriptContent += "set -e\n" // 遇到错误立即退出
	scriptContent += "echo '开始解除服务屏蔽...'\n"
	scriptContent += string(data)
	scriptContent += "\necho '服务屏蔽解除完成'\n"

	if err := ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("无法创建解除服务屏蔽脚本: %v", err)
	}

	fmt.Println("执行解除服务屏蔽脚本...")

	// 执行脚本
	cmd := exec.Command("chroot", target, "/unmask_services.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	// 清理脚本
	os.Remove(scriptPath)

	if err != nil {
		return fmt.Errorf("执行解除服务屏蔽脚本失败: %v", err)
	}

	return nil
}
