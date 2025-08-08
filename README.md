
# 初始化 工程

go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

operator-sdk init --domain dac.io --repo github.com/James-Dao/execution-engine


# 初始化 资源对象和控制器

operator-sdk create api --group dac --version v1alpha1 --kind DataDescriptor --resource --controller

operator-sdk create api --group dac --version v1alpha1 --kind DataAgentContainer --resource --controller



# 本地测试：

make run



# build

amd64:

export GOOS=linux
export GOARCH=amd64
make docker-build docker-push IMG="registry.cn-shanghai.aliyuncs.com/jamesxiong/execution-engine:v0.0.1-amd64"



arm64:

export GOOS=linux
export GOARCH=arm64
make docker-build docker-push IMG="registry.cn-shanghai.aliyuncs.com/jamesxiong/execution-engine:v0.0.1-arm64"




# deploy

## 配置kubeconfig

将kubeconfig文件拿到之后，放到当前用户的.kube文件夹下，命名成config

~/.kube/config




## 下发 controller 到 k8s的execution-engine-system namespace下

make deploy IMG="registry.cn-shanghai.aliyuncs.com/jamesxiong/execution-engine:v0.0.1-amd64"


make deploy IMG="registry.cn-shanghai.aliyuncs.com/jamesxiong/execution-engine:v0.0.1-arm64"