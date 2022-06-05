## redis server启动
### 1. 简介
本小节主要介绍内容为介绍redis server启动时的一些动作，主要包含如下三方面
- redis server启动之后的初始化操作
- redis server初始化哪些关键配置项
- redis server如何开始处理客户端请求

### 2. 详细解析
通过查看redis server入口main()函数，初始化操作分为如下几大部分
- 基本参数初始化
    - 设置时区setlocale(LC_COLLATE,"");
    - 设置哈希函数的随机种子

- 检查哨兵模式，RDB或者AOF模式
- 运行参数解析
    - 设置端口
    - 主从复制相关参数

- 加载配置文件，覆盖默认参数（loadServerConfig, loadServiceConfigString)
- 初始化server(initServer)
    - 初始化DB
    - 初始化LRU缓冲池
    - 注册事件驱动框架函数
- 执行事件驱动框架(aemain)
    - 处理请求
    - 处理定时任务

