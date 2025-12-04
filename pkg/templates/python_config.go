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

