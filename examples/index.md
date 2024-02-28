**# 使用方法示例**

### **KTIB**
KTIB 是一个为 KCR 提供可信访问控制的工具。
请在 https://gitee.com/openeuler/ktib/issues 提供反馈意见。

### **示例用法:**
1. 初始化阶段:

初始化一个空项目:
    
    builder-machine# git pull http://gitlab.com/test/test.git
    builder-machine# cd test
    builder-machine# ktib init --buildType=source

2. 扫描阶段:

扫描项目:(扫描还未实现)

    builder-machine# ktib scan --check-test=true 

3. 构建阶段:

构建项目:

（1） 一步构建：

    builder-machine# ktib make

（2）分层构建：

    builder-machine# ktib builders [command]

可用命令:

* add: 添加一个构建器到配置中。

        ktib builders add [builder-name]
* build: 从 Dockerfile 构建一个镜像。

        ktib builders build [Dockerfile-path]
* commit: 提交一个容器的更改。

        ktib builders commit [container-id]

* copy: 将文件从本地文件系统复制到容器中。

        ktib builders copy [source-path] [destination-path]

* from: 设置当前工作构建器。

        ktib builders from [builder-name]

* label: 根据容器镜像标签执行命令。

        ktib builders label [label-name] [command]

* list: 列出当前可用的工作构建器及其基础镜像。

        ktib builders list 

* mount: 将容器挂载到本地文件系统。

        ktib builders mount [container-id] [mount-path]

* rm: 删除一个容器。

        ktib builders rm [container-id]

* run: 在容器中运行一个命令。

        ktib builders run [container-id] [command]

* umount: 取消挂载容器。

        ktib builders umount [container-id]

4.操作镜像：

操作本地或远程镜像：

    builder-machine# ktib images [command]

