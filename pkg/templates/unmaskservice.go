/*
Copyright (c) 2025 KylinSoft Co., Ltd.
Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
You can use this software according to the terms and conditions of the Mulan PSL v2.
You may obtain a copy of Mulan PSL v2 at:

	http://license.coscl.org.cn/MulanPSL2

THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
See the Mulan PSL v2 for more details.
*/
package templates

const UnmaskService = `# # Disable systemd-networkd.service
rm -rf /etc/systemd/system/multi-user.target.wants/systemd-networkd.service
rm -rf /etc/systemd/system/dbus-org.freedesktop.network1.service
rm -rf /etc/systemd/system/sockets.target.wants/systemd-networkd.socket
rm -rf /etc/systemd/system/network-online.target.wants/systemd-networkd-wait-online.service

# Disable crond.service
rm -rf  /etc/systemd/system/multi-user.target.wants/crond.service
rm -rf  /etc/systemd/system/cron.service

# Disable systemd-timesyncd.service
rm -rf /etc/systemd/system/dbus-org.freedesktop.timesync1.service
rm -rf /etc/systemd/system/sysinit.target.wants/systemd-timesyncd.service

# Disable getty@tty1.service
rm -rf /etc/systemd/system/getty.target.wants/getty@tty1.service


# Disable service (guarded when systemctl is available)
if command -v systemctl >/dev/null 2>&1; then
  systemctl disable system-getty.slice || true
  systemctl disable systemd-networkd.socket systemd-networkd || true
  systemctl disable systemd-hostnamed.service || true
else
  echo "systemctl not found, skipping systemd service disabling step"
fi

# rm systemd drop file
rm -f /lib/systemd/system/multi-user.target.wants/*
rm -f /etc/systemd/system/*.wants/*
rm -f /lib/systemd/system/local-fs.target.wants/*
rm -f /lib/systemd/system/sockets.target.wants/*udev*
rm -f /lib/systemd/system/sockets.target.wants/*initctl*
rm -f /lib/systemd/system/basic.target.wants/*
rm -f /lib/systemd/system/anaconda.target.wants/*
`
