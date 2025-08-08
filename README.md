
# 初始化 工程

go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

operator-sdk init --domain dac.io --repo github.com/James-Dao/execution-engine


# 初始化 资源对象和控制器

operator-sdk create api --group dac --version v1alpha1 --kind DataDescriptor --resource --controller

operator-sdk create api --group dac --version v1alpha1 --kind DataAgentContainer --resource --controller
