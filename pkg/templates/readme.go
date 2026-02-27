/*
   Copyright (c) 2024 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/
package templates

const README = `# Create a basic image using ktib

## Getting started

#### Install ktib
If you haven't already installed ktib, please refer to the official documentation for installation instructions.

#### Steps for using ktib

##### Create a Project
Use the ktib command to create a new project. You can do this with the following command:
ktib project init /path/to/project

##### Configure the Project
Create a configuration file for your project. You can generate a default configuration with:
ktib project default_config > config.yml

Then edit the configuration file to specify the packages and settings for your image.

##### Build RootFS
Build the root filesystem for your image:
ktib project build-rootfs --config config.yml /path/to/project

##### Clean RootFS
Clean unnecessary files and packages from the rootfs to optimize image size:
ktib project clean-rootfs --type minimal /path/to/project

##### Build the Container Image
Build the final container image from the rootfs:
ktib project build --name my-image --tag latest /path/to/project

##### Verify the Image
After the build process completes, use container commands to verify your image. For example:
ktib images list
docker images
`
