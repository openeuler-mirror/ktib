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

const PythonConfigureScript = `#!/bin/bash
set -e
if command -v python3 >/dev/null 2>&1; then
  python3 -m ensurepip --upgrade || true
  rpm -e --nodeps python-setuptools python-pip-wheel || true
  ln -sf /usr/local/bin/pip3 /usr/bin/pip || true
  python3 -c 'import pathlib,shutil;[shutil.rmtree(p) for p in pathlib.Path("/").rglob("__pycache__")]' || true
fi
`
