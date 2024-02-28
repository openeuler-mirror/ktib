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

4.操作镜像：

操作本地或远程镜像：

    builder-machine# ktib images [command]

