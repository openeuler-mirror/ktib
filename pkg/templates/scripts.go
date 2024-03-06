package templates

const Script = `#!/bin/bash -e
cat <<EOF
This is the {{.ImageName}} ktib image:
To use it, install ktib: https://gitee.com/openeuler/ktib

Sample invocation:

ktib make <source>/<rpm> {{.ImageName}} <image>

You can then run the resulting image:
docker run <image>
EOF`
