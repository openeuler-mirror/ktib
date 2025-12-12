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
package templates

// Dockerfile is a default Dockerfile laid down by s2i create
const Dockerfile = `# {{.ImageName}}
FROM BaseImage

# TODO: Put the maintainer name in the image metadata
# LABEL maintainer="Your Name <your@email.com>"

# TODO: Rename the builder environment variable to inform users about application you provide them
# ENV BUILDER_VERSION 1.0

# TODO: Set labels to describe the builder image
#LABEL io.k8s.description="Platform for building xyz" \
#      io.k8s.display-name="builder x.y.z" 

# TODO: Install required packages here:
# RUN yum install -y ... && yum clean all -y
RUN yum install -y rubygems && yum clean all -y
RUN gem install asdf

# TODO (optional): Copy the builder files into /opt/app-root
# COPY ./<builder_folder>/ /opt/app-root/
COPY ./init/bin/xxx /usr/bin/xxx

# This default user is created in the base image
USER 1001

# TODO: Set the default port for applications built using this image
# EXPOSE 8080

# TODO: Set the default CMD for the image
# CMD ["/usr/bin/xxx"]
`

// BaseImageDockerfile is the Dockerfile template used to build base images
const BaseImageDockerfile = `# {{.ImageName}} Base Image
FROM scratch
ADD rootfs.tar /
CMD ["/bin/bash"]
`

// InitImageDockerfile is the Dockerfile template used to build init images
const InitImageDockerfile = `# {{.ImageName}} Init Image
FROM scratch
ADD rootfs.tar /
CMD ["/sbin/init"]
`
