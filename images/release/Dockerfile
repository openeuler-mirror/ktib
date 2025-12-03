# 使用openeuler/go:1.21.1-oe2203lts作为基础镜像
FROM openeuler/go:1.21.1-oe2203lts

# 设置工作目录
WORKDIR /workspace

# 安装构建和运行时依赖
RUN yum update -y && \
    yum install -y make libseccomp-devel gpgme gpgme-devel device-mapper-devel btrfs-progs-devel runc containers-common && \
    yum clean all

# 复制整个项目到容器中
COPY . .

# 使用make build来构建项目
RUN make build

# 设置容器启动命令为/bin/bash
CMD ["/bin/bash"]