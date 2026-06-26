**erc20_go_Demo 配合练习与扩展任务**

```shell
npx hardhat ignition deploy ignition/modules/MyERC20.ts --network localhost
```

# 1.完成以下练习：

1. **练习 1：实现简易块高监控**
   - 每隔 10 秒打印当前最新区块号
   - 当块高长时间不变时给出告警日志

2. **练习 2：实现指定地址交易监控**
   - 订阅新区块
   - 对每个区块中的交易，筛选 `from` 或 `to` 为指定地址的交易并打印

3. **练习 3：扩展 ERC-20 监听**
   - 在日志订阅基础上，支持监听多个 ERC-20 合约地址
   - 增加一个简单的内存索引：按地址维护最近收到/发出的转账列表

```
$env:ETH_WS_URL="ws://127.0.0.1:8545"
$env:ERC20_CONTRACTS="0x5FbDB2315678afecb367f032d93F642f64180aa3,0xa513E6E4b8f2a923D98304ec87F64353C4D5C853"

# 运行命令
go run main.go 01-block-height-monitor.go 02-address-tx-monitor.go 03-multi-erc20-monitor.go --mode 03
```

# 2.实战项目——迷你区块浏览器与 ERC-20 监听服务

本讲义聚焦项目实战部分，帮助你将前面学到的 Go-Eth Client 能力整合到一个可运行的小项目中。

## 1. 项目目标

实现一个基于 Go-Eth Client 的**迷你区块浏览器/监听服务**，具备以下能力：

- 查询**区块 / 交易 / 地址**的基础信息
- 实时监听指定 ERC-20 合约的 `Transfer` 事件
- 提供简单的 HTTP API 或 CLI 接口对外查询
- 为后续扩展（多合约、多网络、持久化存储）预留空间

## 2. 功能范围设计

### 2.1 必做功能（MVP）

1. **区块查询**
   - 根据区块号或哈希，返回：
     - 区块号、哈希、父哈希、时间戳
     - 交易数量、Gas 使用情况等简要信息

2. **交易查询**
   - 根据交易哈希，返回：
     - from / to / value / gas / input data
     - 回执中的 status / gasUsed / logs 数量等

3. **ERC-20 转账监听**
   - 背景协程订阅指定 ERC-20 合约的 `Transfer` 事件
   - 将最近 N 条事件保存在内存中（如环形队列）
   - 提供接口查看最近事件列表

### 2.2 选做功能（进阶）

- 地址视图：
  - 查看某地址最近参与的交易列表
  - 查看地址当前 ETH 余额（后续可扩展到代币余额）
- 历史扫描：
  - 程序启动时，从指定高度回放扫描一定区间的历史区块/事件
- 持久化存储：
  - 将事件写入 SQLite / PostgreSQL，并支持分页查询

## 3. 推荐项目结构

在 `lesson-04/examples/09-project` 中，我们提供了一个最小可行版本（MVP）的实现：

```
exercist/02-project/
├── go.mod
└── main.go                # 单文件实现，包含所有功能
```

## 7. 练习与扩展任务

你可以在完成基础功能后继续练习：

1. **练习 1：为地址增加过滤参数**
   - 在 `/api/events` 增加 `address` 参数，只返回涉及该地址的事件

2. **练习 2：增加分页能力**
   - 为事件查询添加 `page` 与 `page_size` 参数
   - 修改 `EventStore.List()` 方法或 HTTP handler 支持分页返回

3. **练习 3：增加简单的 Web 页面**
   - 使用任意前端框架或纯 HTML，展示最近区块与事件列表
   - 可以添加一个 `GET /` 路由返回 HTML 页面，或使用模板引擎渲染
4. **练习 4：实现重连机制**
   - 参考 `examples/07-reconnect-strategy/main.go`，为事件订阅添加自动重连功能
   - 当订阅失败时，等待一段时间后重新建立连接和订阅
5. **练习 5：增加区块和交易查询接口**
   - 添加 `GET /api/block/:number` 和 `GET /api/tx/:hash` 接口
   - 复用前面示例中的区块和交易查询逻辑

# 3.作业——homework05

目录下 03-task/homework05.md

## 任务 1：区块链读写 任务目标

## 任务 2：合约代码生成 任务目标
