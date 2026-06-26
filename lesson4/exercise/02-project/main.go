// ==============================================
// 包声明：main包是Go程序的入口包，编译后生成可执行文件
// ==============================================
package main

// ==============================================
// 依赖导入：所有需要用到的标准库和第三方库
// ==============================================
import (
	// 标准库：上下文管理，用于控制请求超时、取消、跨协程传递取消信号
	"context"
	// 标准库：JSON编解码，用于API响应和请求处理
	"encoding/json"
	// 标准库：格式化输出，用于字符串拼接和错误信息
	"fmt"
	// 标准库：日志输出，带时间戳的打印
	"log"
	// 标准库：数学函数，这里用Min计算重连等待时间的最大值
	"math"
	// 标准库：大整数处理，以太坊的金额、区块号都是大整数，不能用普通int
	"math/big"
	// 标准库：HTTP服务和客户端，用于提供API接口
	"net/http"
	// 标准库：操作系统相关，用于读取环境变量、处理信号
	"os"
	// 标准库：系统信号处理，用于捕获Ctrl+C实现优雅退出
	"os/signal"
	// 标准库：路径处理，用于从URL路径中提取参数
	"path"
	// 标准库：字符串转数字，用于解析分页参数、区块号等
	"strconv"
	// 标准库：字符串处理，用于地址大小写转换、前缀判断等
	"strings"
	// 标准库：并发同步，这里用读写锁保证事件存储的并发安全
	"sync"
	// 标准库：系统调用，用于指定捕获的信号类型
	"syscall"
	// 标准库：时间处理，用于超时、等待、时间戳转换
	"time"

	// 第三方库：go-ethereum的核心接口，定义了FilterQuery等通用类型
	"github.com/ethereum/go-ethereum"
	// 第三方库：ABI解析，用于合约方法编码、事件解码
	"github.com/ethereum/go-ethereum/accounts/abi"
	// 第三方库：通用类型，地址、哈希等基础类型定义
	"github.com/ethereum/go-ethereum/common"
	// 第三方库：十六进制工具，用于把字节数组转成带0x前缀的十六进制字符串
	"github.com/ethereum/go-ethereum/common/hexutil"
	// 第三方库：核心类型，区块、交易、日志等以太坊核心数据结构
	"github.com/ethereum/go-ethereum/core/types"
	// 第三方库：以太坊客户端，封装了所有RPC调用的方法
	"github.com/ethereum/go-ethereum/ethclient"
)

// ==============================================
// 常量与配置：所有硬编码的配置都集中在这里，方便统一修改
// ==============================================

// ERC20 标准 ABI（Application Binary Interface，应用二进制接口）
// 作用：定义了和合约交互的接口规范，相当于合约的"API文档"
// 包含3个标准ERC20接口：
// 1. Transfer事件：转账时触发，from(转出)、to(转入)、value(金额)
// 2. balanceOf方法：查询指定地址的代币余额，只读方法
// 3. decimals方法：查询代币精度（比如18位就是最小单位是1e-18）
const erc20ABIJSON = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "name": "from", "type": "address"},
      {"indexed": true, "name": "to", "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ],
    "name": "Transfer",
    "type": "event"
  },
  {
    "constant": true,
    "inputs": [{"name": "_owner", "type": "address"}],
    "name": "balanceOf",
    "outputs": [{"name": "balance", "type": "uint256"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "decimals",
    "outputs": [{"name": "", "type": "uint8"}],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  }
]`

// 默认配置常量：集中管理所有默认值，避免魔法数字
const (
	defaultEventLimit    = 100     // 默认最多存储100条事件（环形缓冲大小）
	defaultPageSize      = 20      // 默认分页大小，每页返回20条
	maxPageSize          = 100     // 最大分页大小，防止一次请求太多数据压垮服务
	defaultHTTPAddr      = ":8080" // 默认HTTP服务端口，监听所有网卡的8080端口
	maxReconnectWait     = 60      // 重连最大等待时间（秒），避免无限等待
	initialReconnectWait = 1       // 重连初始等待时间（秒），指数退避的起始值
)

// ==============================================
// 数据模型：所有结构体定义，对应业务数据和API响应格式
// ==============================================

// TransferEvent ERC20转账事件结构体
// 作用：统一存储解析后的转账事件，方便后续查询和展示
// json tag：用于JSON序列化时指定字段名，API返回时自动转成下划线格式
type TransferEvent struct {
	BlockNumber uint64    `json:"block_number"` // 事件所在的区块号
	TxHash      string    `json:"tx_hash"`      // 触发事件的交易哈希
	From        string    `json:"from"`         // 转出地址
	To          string    `json:"to"`           // 转入地址
	Value       string    `json:"value"`        // 转账金额（原始uint256字符串，单位是最小单位）
	Timestamp   time.Time `json:"timestamp"`    // 事件处理时间（可优化为区块时间）
}

// EventStore 并发安全的事件存储（环形缓冲）
// 作用：在内存中存储最近的N条事件，支持并发读写
// 设计要点：
// 1. 用sync.RWMutex读写锁保证并发安全（读多写少场景性能好）
// 2. 环形缓冲：超过最大数量时丢弃最旧的事件，避免内存无限增长
type EventStore struct {
	mu     sync.RWMutex    // 读写锁：读共享、写独占，适合读多写少的场景
	events []TransferEvent // 事件切片，存储所有事件
	limit  int             // 最大存储数量，环形缓冲的大小
}

// BlockInfo 区块信息响应结构
// 作用：统一区块查询接口的返回格式，只返回前端需要的字段
type BlockInfo struct {
	Number     uint64 `json:"number"`      // 区块号
	Hash       string `json:"hash"`        // 区块哈希
	ParentHash string `json:"parent_hash"` // 父区块哈希（上一个区块的哈希）
	Timestamp  uint64 `json:"timestamp"`   // 区块时间戳（Unix时间，秒）
	TimeStr    string `json:"time_str"`    // 格式化后的时间字符串，方便人类阅读
	TxCount    int    `json:"tx_count"`    // 区块内的交易数量
	GasUsed    uint64 `json:"gas_used"`    // 区块实际消耗的Gas总量
	GasLimit   uint64 `json:"gas_limit"`   // 区块Gas上限（每个区块最多能容纳的Gas）
	Miner      string `json:"miner"`       // 矿工地址（打包这个区块的节点）
	Size       uint64 `json:"size"`        // 区块大小（字节）
}

// TxInfo 交易信息响应结构
// 作用：统一交易查询接口的返回格式，包含交易本身和回执信息
type TxInfo struct {
	Hash        string `json:"hash"`         // 交易哈希
	From        string `json:"from"`         // 发送方地址
	To          string `json:"to"`           // 接收方地址（合约交易就是合约地址）
	Value       string `json:"value"`        // 转账金额（wei字符串，ETH的最小单位）
	ValueEth    string `json:"value_eth"`    // 转成ETH单位的金额，方便人类阅读
	Nonce       uint64 `json:"nonce"`        // 交易序号，防止重放攻击
	Gas         uint64 `json:"gas"`          // 用户设置的Gas上限
	GasPrice    string `json:"gas_price"`    // Gas价格（wei/Gas）
	Input       string `json:"input"`        // 交易输入数据，合约调用的参数编码
	BlockNumber uint64 `json:"block_number"` // 交易所在的区块号（pending交易为0）

	// 交易回执信息：交易上链后才会有，包含执行结果
	Status    uint64 `json:"status"`     // 交易状态：1=成功，0=失败
	GasUsed   uint64 `json:"gas_used"`   // 交易实际消耗的Gas
	LogsCount int    `json:"logs_count"` // 交易产生的日志数量
	TxIndex   uint   `json:"tx_index"`   // 交易在区块中的序号
}

// ApiResponse 统一API响应格式
// 作用：所有接口都返回统一格式，前端对接更方便，错误处理更规范
// 设计要点：
// 1. code=0表示成功，非0表示错误
// 2. 分页接口额外返回total、page、page_size、pages字段
type ApiResponse struct {
	Code     int         `json:"code"`                // 状态码：0成功，其他为错误码
	Message  string      `json:"message"`             // 状态描述：success或错误信息
	Data     interface{} `json:"data,omitempty"`      // 响应数据，成功时返回
	Total    int         `json:"total,omitempty"`     // 总条数，分页接口用
	Page     int         `json:"page,omitempty"`      // 当前页码，分页接口用
	PageSize int         `json:"page_size,omitempty"` // 每页条数，分页接口用
	Pages    int         `json:"pages,omitempty"`     // 总页数，分页接口用
}

// ==============================================
// EventStore 方法：事件存储的所有操作方法
// ==============================================

// NewEventStore 创建事件存储实例
// 参数：limit - 最大存储事件数量
// 返回：EventStore指针
// 原理：预分配切片容量，避免后续频繁扩容
func NewEventStore(limit int) *EventStore {
	return &EventStore{
		// 预分配容量为limit的切片，避免append时频繁扩容影响性能
		events: make([]TransferEvent, 0, limit),
		limit:  limit,
	}
}

// Add 添加新事件到存储
// 参数：e - 要添加的转账事件
// 原理：加写锁，超过限制时丢弃最旧的一条（环形缓冲）
// 并发安全：写锁保护，同一时间只能有一个协程写入
func (s *EventStore) Add(e TransferEvent) {
	// 加写锁：独占锁，其他读写都要等锁释放
	s.mu.Lock()
	// 函数退出时自动释放锁，避免忘记释放导致死锁
	defer s.mu.Unlock()

	// 如果当前事件数已经达到上限，丢弃最旧的一条（切片第一个元素）
	// 这就是环形缓冲的核心逻辑：先进先出，满了就丢最旧的
	if len(s.events) >= s.limit {
		s.events = s.events[1:]
	}

	// 添加新事件到切片末尾
	s.events = append(s.events, e)
}

// List 查询事件列表，支持地址过滤和分页
// 参数：
//
//	address - 过滤地址，为空则返回所有事件
//	page - 页码，从1开始
//	pageSize - 每页条数
//
// 返回：
//
//	int - 过滤后的总事件数
//	[]TransferEvent - 当前页的事件列表（倒序，最新的在最前面）
//
// 并发安全：读锁保护，多个协程可以同时读
func (s *EventStore) List(address string, page int, pageSize int) (int, []TransferEvent) {
	// 加读锁：共享锁，多个协程可以同时读，不阻塞读，只阻塞写
	s.mu.RLock()
	// 函数退出时自动释放读锁
	defer s.mu.RUnlock()

	// ========== 第一步：按地址过滤 ==========
	var filtered []TransferEvent
	if address == "" {
		// 没有传过滤地址，直接用所有事件
		filtered = s.events
	} else {
		// 转小写比较，避免大小写不一致导致匹配失败
		// 坑点：以太坊地址大小写不敏感，必须统一大小写再比较
		addrLower := strings.ToLower(address)
		for _, evt := range s.events {
			// 匹配from或to任一即可，只要涉及该地址的事件都返回
			if strings.ToLower(evt.From) == addrLower || strings.ToLower(evt.To) == addrLower {
				filtered = append(filtered, evt)
			}
		}
	}

	total := len(filtered)
	if total == 0 {
		return 0, []TransferEvent{}
	}

	// ========== 第二步：分页参数校验 ==========
	// 页码小于1的话默认第1页
	if page < 1 {
		page = 1
	}
	// 每页条数非法的话用默认值，且不能超过最大值
	if pageSize <= 0 || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}

	// ========== 第三步：计算分页范围（倒序，最新的在前面） ==========
	// 原理：事件是按时间顺序存在切片里的（旧的在前，新的在后）
	// 倒序返回就是从后往前取，最新的在最前面，符合浏览习惯
	// start：当前页第一条在切片中的索引（包含）
	// end：当前页最后一条在切片中的索引（不包含）
	start := total - (page * pageSize)
	end := total - ((page - 1) * pageSize)

	// 边界处理：start不能小于0
	if start < 0 {
		start = 0
	}
	// 边界处理：end不能超过总长度
	if end > total {
		end = total
	}
	// 边界处理：如果start >= end，说明没有数据了
	if start >= end {
		return total, []TransferEvent{}
	}

	// ========== 第四步：倒序组装分页数据 ==========
	pageData := make([]TransferEvent, end-start)
	// 从后往前遍历，把最新的放在最前面
	for i := 0; i < end-start; i++ {
		// end-1-i：从最后一条往前数第i条
		pageData[i] = filtered[end-1-i]
	}

	return total, pageData
}

// GetLastBlockNumber 获取最后一条事件的区块号，用于断点续传
// 返回：最后一条事件的区块号，如果没有事件返回0
// 作用：断线重连时，从这个区块号+1开始补历史事件，保证不丢数据
func (s *EventStore) GetLastBlockNumber() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 没有事件的话返回0，表示需要从头开始扫
	if len(s.events) == 0 {
		return 0
	}
	// 返回最后一条（最新的）事件的区块号
	return s.events[len(s.events)-1].BlockNumber
}

// ==============================================
// 历史扫描与事件解析工具函数
// ==============================================

// parseLog 解析单个日志（types.Log）为TransferEvent
// 参数：
//
//	parsedABI - 解析后的ABI对象
//	vLog - 原始日志数据
//
// 返回：
//
//	TransferEvent - 解析后的转账事件
//	error - 解析失败时返回错误
//
// 原理：
// 1. Indexed参数存在Topics数组里，需要自己从Topics中提取
// 2. 非Indexed参数存在Data字段里，用ABI.Unpack解码
// 坑点：Topics[0]是事件签名哈希，Topics[1]开始才是第一个Indexed参数
func parseLog(parsedABI abi.ABI, vLog types.Log) (TransferEvent, error) {
	// Transfer事件有2个Indexed参数（from和to），所以Topics至少要有3个元素
	// Topics[0] = 事件签名哈希，Topics[1] = from，Topics[2] = to
	if len(vLog.Topics) < 3 {
		return TransferEvent{}, fmt.Errorf("invalid log topics length")
	}

	// ========== 解析非Indexed参数（value） ==========
	// 非Indexed参数都编码在Data字段里，用ABI的Unpack方法解码
	// 定义一个临时结构体接收解码结果，字段名和类型要和ABI里的一致
	var event struct {
		Value *big.Int // 转账金额，uint256对应Go的*big.Int
	}
	// UnpackIntoInterface：把Data解码到结构体里，第二个参数是事件名
	if err := parsedABI.UnpackIntoInterface(&event, "Transfer", vLog.Data); err != nil {
		return TransferEvent{}, fmt.Errorf("unpack log data failed: %v", err)
	}

	// ========== 解析Indexed参数（from、to） ==========
	// Indexed的address类型参数，存在Topics里，是32字节的哈希
	// 地址是20字节，所以32字节里前12字节是0填充，后20字节是真实地址
	// 用common.BytesToAddress把32字节转成地址类型，自动处理前导0
	from := common.BytesToAddress(vLog.Topics[1].Bytes())
	to := common.BytesToAddress(vLog.Topics[2].Bytes())

	// 组装成TransferEvent返回
	return TransferEvent{
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash.Hex(),
		From:        from.Hex(),
		To:          to.Hex(),
		Value:       event.Value.String(),
		// 注意：这里用的是当前时间，不是区块时间
		// 优化点：可以根据区块号查询区块时间，更准确
		Timestamp: time.Now(),
	}, nil
}

// scanHistoricalEvents 扫描指定区块范围的历史Transfer事件，存入存储
// 参数：
//
//	ctx - 上下文，用于取消扫描
//	client - 以太坊客户端
//	parsedABI - 解析后的ABI
//	contract - 合约地址
//	store - 事件存储
//	fromBlock - 起始区块号（包含）
//	toBlock - 结束区块号（包含）
//
// 返回：
//
//	int - 扫描到的事件数量
//	error - 扫描失败时返回错误
//
// 原理：用eth_getLogs接口批量查询历史日志，然后逐个解析存入存储
func scanHistoricalEvents(ctx context.Context, client *ethclient.Client,
	parsedABI abi.ABI, contract common.Address, store *EventStore,
	fromBlock uint64, toBlock uint64) (int, error) {

	// 起始区块大于结束区块，直接返回，非法范围
	if fromBlock > toBlock {
		return 0, nil
	}

	// 获取Transfer事件的签名哈希，用于过滤事件
	// 原理：每个事件的签名（比如Transfer(address,address,uint256)）做Keccak256哈希，就是Topics[0]
	// 加这个过滤可以只返回Transfer事件，减少无效数据
	transferEventID := parsedABI.Events["Transfer"].ID

	// 构建过滤查询条件
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),       // 起始区块
		ToBlock:   big.NewInt(int64(toBlock)),         // 结束区块
		Addresses: []common.Address{contract},         // 合约地址列表，只查这些合约的日志
		Topics:    [][]common.Hash{{transferEventID}}, // 主题过滤，只查Transfer事件
	}

	// 调用FilterLogs接口查询历史日志
	// 对应RPC方法：eth_getLogs
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("filter logs failed: %v", err)
	}

	// 遍历所有日志，逐个解析存入存储
	count := 0
	for _, vLog := range logs {
		evt, err := parseLog(parsedABI, vLog)
		if err != nil {
			log.Printf("[历史扫描] 解析事件失败: %v", err)
			// 单个事件解析失败不影响整体，跳过继续
			continue
		}
		store.Add(evt)
		count++
	}

	return count, nil
}

// ==============================================
// 事件订阅（带断点续传自动重连）
// ==============================================

// subscribeTransferEvents 订阅ERC20 Transfer事件，带断点续传自动重连
// 参数：
//
//	ctx - 上下文，用于取消订阅
//	client - 以太坊客户端
//	parsedABI - 解析后的ABI
//	contract - 合约地址
//	store - 事件存储
//
// 原理：
// 1. 外层死循环负责重连，内层循环负责处理实时事件
// 2. 每次重连前先补全断线期间的历史事件（断点续传）
// 3. 指数退避算法：每次重连失败等待时间翻倍，避免频繁重试打满节点
// 注意：这个函数应该在goroutine中运行，不会阻塞主流程
func subscribeTransferEvents(ctx context.Context, client *ethclient.Client,
	parsedABI abi.ABI, contract common.Address, store *EventStore) {

	// 重连等待时间，初始为1秒
	reconnectWait := initialReconnectWait
	// Transfer事件签名哈希，提前算好复用
	transferEventID := parsedABI.Events["Transfer"].ID

	// 外层死循环：负责重连，订阅失败就退到这里重新来
	for {
		// ========== 断点续传：重连前先补全缺失的历史事件 ==========
		// 原理：断线期间可能产生了新的事件，订阅不会自动补，需要自己扫历史补上
		// 步骤：
		// 1. 获取当前最新区块号
		// 2. 获取最后处理的区块号
		// 3. 如果有缺失，扫描中间的区块补全事件

		// 1. 获取当前最新区块号
		latestBlock, err := client.BlockNumber(ctx)
		if err != nil {
			log.Printf("[订阅] 获取最新区块号失败: %v，%d秒后重试...", err, reconnectWait)
			// 等待一段时间后重试，支持上下文取消
			if !sleepWithContext(ctx, time.Duration(reconnectWait)*time.Second) {
				log.Println("订阅协程退出：上下文取消")
				return
			}
			// 指数退避：等待时间翻倍，不超过最大值
			reconnectWait = int(math.Min(float64(reconnectWait*2), float64(maxReconnectWait)))
			continue
		}

		// 2. 获取最后处理的区块号
		lastProcessedBlock := store.GetLastBlockNumber()
		// 3. 如果有缺失的区块，先扫描补全
		if lastProcessedBlock > 0 && lastProcessedBlock < latestBlock {
			// 从最后处理的区块+1开始，到最新区块结束
			startBlock := lastProcessedBlock + 1
			log.Printf("[断点续传] 补全区块 %d ~ %d 的历史事件...", startBlock, latestBlock)
			count, err := scanHistoricalEvents(ctx, client, parsedABI, contract, store, startBlock, latestBlock)
			if err != nil {
				log.Printf("[断点续传] 扫描失败: %v", err)
			} else {
				log.Printf("[断点续传] 补全完成，新增 %d 条事件", count)
			}
		}

		// ========== 开始实时订阅 ==========
		// 构建订阅查询条件，和历史扫描的条件一致
		query := ethereum.FilterQuery{
			Addresses: []common.Address{contract},
			Topics:    [][]common.Hash{{transferEventID}},
		}

		// 创建日志通道，缓冲100条，防止处理不过来阻塞订阅
		logsCh := make(chan types.Log, 100)
		// 订阅日志，对应RPC方法：eth_subscribe，参数是logs
		sub, err := client.SubscribeFilterLogs(ctx, query, logsCh)
		if err != nil {
			log.Printf("[订阅失败] %v，%d秒后重试...", err, reconnectWait)
			// 订阅失败，等待后重试
			if !sleepWithContext(ctx, time.Duration(reconnectWait)*time.Second) {
				log.Println("订阅协程退出：上下文取消")
				return
			}
			reconnectWait = int(math.Min(float64(reconnectWait*2), float64(maxReconnectWait)))
			continue
		}

		// 订阅成功，重置重连等待时间为初始值
		// 原理：连接成功后说明节点正常，下次重连从1秒开始等，不用等很久
		reconnectWait = initialReconnectWait
		log.Printf("[订阅成功] 监听合约 %s 的 Transfer 事件", contract.Hex())

		// ========== 事件处理循环（内层循环） ==========
		// 用标签跳出多层循环，Go语言的标准用法
	subscribeLoop:
		for {
			// select多路复用：同时监听多个通道
			select {
			// 收到新的日志事件
			case vLog := <-logsCh:
				// 解析事件
				evt, err := parseLog(parsedABI, vLog)
				if err != nil {
					log.Printf("[事件解析失败] %v", err)
					continue
				}
				// 存入存储
				store.Add(evt)

			// 订阅出错（比如断线、节点关闭连接）
			case err := <-sub.Err():
				log.Printf("[订阅出错] %v，准备重连...", err)
				// 跳出内层循环，回到外层的重连逻辑
				break subscribeLoop

			// 上下文被取消（比如程序退出）
			case <-ctx.Done():
				log.Println("订阅协程退出：上下文取消")
				return
			}
		}

		// 重连前等待，指数退避
		if !sleepWithContext(ctx, time.Duration(reconnectWait)*time.Second) {
			return
		}
		reconnectWait = int(math.Min(float64(reconnectWait*2), float64(maxReconnectWait)))
	}
}

// sleepWithContext 带上下文取消的睡眠
// 参数：
//
//	ctx - 上下文
//	d - 睡眠时长
//
// 返回：
//
//	bool - true表示正常睡完了，false表示被上下文取消了
//
// 原理：用Timer+select实现，而不是time.Sleep
// 为什么不用time.Sleep？
// 因为time.Sleep无法被中断，程序退出时还要等Sleep完才能退，体验差
// 用Timer+select可以监听ctx取消，提前退出睡眠
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	// 创建定时器，d时间后触发
	timer := time.NewTimer(d)
	// 函数退出时停止定时器，避免资源泄漏
	defer timer.Stop()

	// 多路复用：等定时器触发，还是等上下文取消
	select {
	case <-timer.C:
		// 定时器触发，正常睡完了
		return true
	case <-ctx.Done():
		// 上下文取消，提前退出
		return false
	}
}

// ==============================================
// HTTP Handler 工具函数：统一响应格式
// ==============================================

// sendJSON 发送JSON响应
// 参数：
//
//	w - HTTP响应写入器
//	code - HTTP状态码
//	resp - 响应结构体
//
// 作用：统一设置Content-Type，统一编码JSON，避免每个handler重复写
func sendJSON(w http.ResponseWriter, code int, resp ApiResponse) {
	// 设置响应头：内容类型是JSON，编码是utf-8
	w.Header().Set("Content-Type", "application/json")
	// 设置HTTP状态码
	w.WriteHeader(code)
	// 编码JSON并写入响应
	// 忽略错误，因为写入失败客户端会自己感知
	_ = json.NewEncoder(w).Encode(resp)
}

// sendSuccess 发送成功响应
// 快捷方法，成功时不用每次都写完整的ApiResponse
func sendSuccess(w http.ResponseWriter, data interface{}) {
	sendJSON(w, http.StatusOK, ApiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// sendError 发送错误响应
// 快捷方法，出错时不用每次都写完整的ApiResponse
func sendError(w http.ResponseWriter, httpCode int, message string) {
	sendJSON(w, httpCode, ApiResponse{
		Code:    httpCode,
		Message: message,
	})
}

// ==============================================
// HTTP 接口 Handler：每个接口的处理逻辑
// ==============================================

// homeHandler 首页导航
// 作用：访问根路径时返回一个简单的HTML页面，展示所有接口说明
// 方便用户不用记接口地址，直接打开首页就能看
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// 只处理根路径，其他路径返回404
	if r.URL.Path != "/" {
		sendError(w, http.StatusNotFound, "404 not found")
		return
	}

	// 简单的HTML页面，内联样式，不用额外的静态文件
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>迷你以太坊区块浏览器</title>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; }
        h1 { color: #333; }
        .api-item { margin: 20px 0; padding: 15px; background: #f5f5f5; border-radius: 8px; }
        .method { display: inline-block; padding: 2px 8px; background: #4CAF50; color: white; border-radius: 4px; font-size: 12px; margin-right: 10px; }
        .desc { color: #666; margin-top: 8px; font-size: 14px; }
        code { background: #eaeaea; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>⛓️ 迷你以太坊区块浏览器</h1>
    <p>基于 Go-Ethereum 开发的 MVP 版本，支持区块查询、交易查询、ERC20 事件监听、代币余额查询</p>
    
    <h2>API 接口列表</h2>
    
    <div class="api-item">
        <span class="method">GET</span><code>/api/block/{number_or_hash}</code>
        <div class="desc">查询区块信息，支持区块号（如 12345）或区块哈希</div>
    </div>
    <div class="api-item">
        <span class="method">GET</span><code>/api/tx/{hash}</code>
        <div class="desc">查询交易详情，包含交易信息与回执信息</div>
    </div>
    <div class="api-item">
        <span class="method">GET</span><code>/api/balance/{address}</code>
        <div class="desc">查询指定地址的 ETH 余额</div>
    </div>
    <div class="api-item">
        <span class="method">GET</span><code>/api/token-balance/{address}</code>
        <div class="desc">查询指定地址的 ERC20 代币余额
            <br>参数：
            <br>- <code>contract</code>：可选，合约地址，默认使用配置的ERC20合约
        </div>
    </div>
    <div class="api-item">
        <span class="method">GET</span><code>/api/events</code>
        <div class="desc">查询 ERC20 转账事件列表
            <br>参数：
            <br>- <code>address</code>：可选，过滤涉及该地址的事件
            <br>- <code>page</code>：可选，页码，默认1
            <br>- <code>page_size</code>：可选，每页条数，默认20，最大100
        </div>
    </div>
</body>
</html>`

	// 设置响应头：HTML格式，utf-8编码
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 写入HTML内容
	_, _ = w.Write([]byte(html))
}

// blockHandler 区块查询接口
// 参数：client - 以太坊客户端
// 返回：http.HandlerFunc 处理函数
// 功能：根据区块号或区块哈希查询区块信息
// 路由：GET /api/block/{number_or_hash}
func blockHandler(client *ethclient.Client) http.HandlerFunc {
	// 闭包：捕获外部的client变量，不用每次请求都传
	return func(w http.ResponseWriter, r *http.Request) {
		// 从URL路径中提取最后一段作为参数
		// 比如 /api/block/12345，提取到的是"12345"
		param := path.Base(r.URL.Path)
		// 没有传参数，返回错误
		if param == "block" || param == "" {
			sendError(w, http.StatusBadRequest, "请提供区块号或区块哈希")
			return
		}

		// 获取请求的上下文，包含超时、取消等信息
		ctx := r.Context()
		var block *types.Block
		var err error

		// 判断参数是区块号还是区块哈希
		// 区块哈希的特征：0x开头，长度66位（32字节=64位十六进制+0x前缀）
		if strings.HasPrefix(param, "0x") && len(param) == 66 {
			// 区块哈希：用BlockByHash查询
			hash := common.HexToHash(param)
			block, err = client.BlockByHash(ctx, hash)
		} else {
			// 区块号：先转成uint64，再用BlockByNumber查询
			number, err2 := strconv.ParseUint(param, 10, 64)
			if err2 != nil {
				sendError(w, http.StatusBadRequest, "无效的区块号或哈希")
				return
			}
			block, err = client.BlockByNumber(ctx, big.NewInt(int64(number)))
		}

		// 查询失败，返回404
		if err != nil {
			sendError(w, http.StatusNotFound, fmt.Sprintf("查询区块失败: %v", err))
			return
		}

		// 组装响应数据，把types.Block转成我们定义的BlockInfo
		// 只返回需要的字段，避免返回多余数据
		blockInfo := BlockInfo{
			Number:     block.NumberU64(),
			Hash:       block.Hash().Hex(),
			ParentHash: block.ParentHash().Hex(),
			Timestamp:  block.Time(),
			// 把Unix时间戳转成人类可读的字符串
			TimeStr:  time.Unix(int64(block.Time()), 0).Format("2006-01-02 15:04:05"),
			TxCount:  len(block.Transactions()),
			GasUsed:  block.GasUsed(),
			GasLimit: block.GasLimit(),
			Miner:    block.Coinbase().Hex(),
			Size:     block.Size(),
		}

		// 返回成功响应
		sendSuccess(w, blockInfo)
	}
}

// txHandler 交易查询接口
// 参数：client - 以太坊客户端
// 返回：http.HandlerFunc 处理函数
// 功能：根据交易哈希查询交易详情，包含交易信息和回执信息
// 路由：GET /api/tx/{hash}
func txHandler(client *ethclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从URL路径提取交易哈希
		txHashHex := path.Base(r.URL.Path)
		if txHashHex == "tx" || txHashHex == "" {
			sendError(w, http.StatusBadRequest, "请提供交易哈希")
			return
		}

		// 校验交易哈希格式：0x开头，66位长
		if !strings.HasPrefix(txHashHex, "0x") || len(txHashHex) != 66 {
			sendError(w, http.StatusBadRequest, "无效的交易哈希格式")
			return
		}

		ctx := r.Context()
		// 把字符串转成Hash类型
		txHash := common.HexToHash(txHashHex)

		// ========== 1. 查询交易本身 ==========
		// TransactionByHash返回三个值：交易对象、是否pending、错误
		tx, isPending, err := client.TransactionByHash(ctx, txHash)
		if err != nil {
			sendError(w, http.StatusNotFound, fmt.Sprintf("查询交易失败: %v", err))
			return
		}

		// ========== 2. 解析交易发送方 ==========
		// 交易本身不直接存from地址，需要从签名中恢复
		// types.Sender用签名算法从交易中恢复出发送方地址
		// 坑点：必须用对应链的签名器，不然恢复的地址不对
		from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("解析发送方失败: %v", err))
			return
		}

		// ========== 3. 查询交易回执 ==========
		// 交易回执只有交易上链后才有，pending状态的交易没有回执
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err != nil && !isPending {
			// 不是pending状态但查不到回执，返回错误
			sendError(w, http.StatusNotFound, fmt.Sprintf("查询交易回执失败: %v", err))
			return
		}

		// ========== 4. 组装响应数据 ==========
		// to地址可能为空（合约创建交易没有to）
		toAddr := ""
		if tx.To() != nil {
			toAddr = tx.To().Hex()
		}

		// 把wei转成ETH单位，方便人类阅读
		// 1 ETH = 1e18 wei
		valueEth := new(big.Float).Quo(
			new(big.Float).SetInt(tx.Value()),
			new(big.Float).SetFloat64(1e18),
		)

		// 先填充基础信息，回执信息后面补
		txInfo := TxInfo{
			Hash:        tx.Hash().Hex(),
			From:        from.Hex(),
			To:          toAddr,
			Value:       tx.Value().String(),
			ValueEth:    valueEth.Text('f', 6), // 保留6位小数
			Nonce:       tx.Nonce(),
			Gas:         tx.Gas(),
			GasPrice:    tx.GasPrice().String(),
			Input:       hexutil.Encode(tx.Data()),
			BlockNumber: 0,
			Status:      0,
			GasUsed:     0,
			LogsCount:   0,
			TxIndex:     0,
		}

		// 如果交易已经打包（不是pending），填充回执信息
		if !isPending && receipt != nil {
			txInfo.BlockNumber = receipt.BlockNumber.Uint64()
			txInfo.Status = receipt.Status
			txInfo.GasUsed = receipt.GasUsed
			txInfo.LogsCount = len(receipt.Logs)
			txInfo.TxIndex = receipt.TransactionIndex
		}

		sendSuccess(w, txInfo)
	}
}

// balanceHandler ETH余额查询接口
// 参数：client - 以太坊客户端
// 返回：http.HandlerFunc 处理函数
// 功能：查询指定地址的ETH余额
// 路由：GET /api/balance/{address}
func balanceHandler(client *ethclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从URL路径提取地址
		addressHex := path.Base(r.URL.Path)
		if addressHex == "balance" || addressHex == "" {
			sendError(w, http.StatusBadRequest, "请提供地址")
			return
		}

		// 校验地址格式
		if !common.IsHexAddress(addressHex) {
			sendError(w, http.StatusBadRequest, "无效的地址格式")
			return
		}

		ctx := r.Context()
		address := common.HexToAddress(addressHex)

		// 查询最新区块的余额
		// 第二个参数是区块号，nil表示最新区块
		// 也可以传big.NewInt(12345)查询指定历史区块的余额
		balance, err := client.BalanceAt(ctx, address, nil)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("查询余额失败: %v", err))
			return
		}

		// 转成ETH单位
		balanceEth := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			new(big.Float).SetFloat64(1e18),
		)

		// 组装返回结果，同时返回原始值和人类可读值
		result := map[string]interface{}{
			"address":     address.Hex(),
			"balance":     balance.String(),        // 原始wei值，精确
			"balance_eth": balanceEth.Text('f', 6), // ETH单位，方便阅读
		}

		sendSuccess(w, result)
	}
}

// tokenBalanceHandler ERC20代币余额查询接口
// 参数：
//
//	client - 以太坊客户端
//	parsedABI - 解析后的ABI
//	defaultContract - 默认合约地址
//
// 返回：http.HandlerFunc 处理函数
// 功能：查询指定地址的ERC20代币余额，支持指定合约地址
// 路由：GET /api/token-balance/{address}?contract=xxx
// 原理：调用合约的balanceOf只读方法，属于Call操作，不上链、不耗Gas
func tokenBalanceHandler(client *ethclient.Client, parsedABI abi.ABI, defaultContract common.Address) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从URL路径提取要查询的地址
		addressHex := path.Base(r.URL.Path)
		if addressHex == "token-balance" || addressHex == "" {
			sendError(w, http.StatusBadRequest, "请提供地址")
			return
		}

		if !common.IsHexAddress(addressHex) {
			sendError(w, http.StatusBadRequest, "无效的地址格式")
			return
		}

		// ========== 获取合约地址 ==========
		// 优先用query参数里的contract，没有的话用默认配置的合约
		contractHex := r.URL.Query().Get("contract")
		var contractAddr common.Address
		if contractHex != "" {
			if !common.IsHexAddress(contractHex) {
				sendError(w, http.StatusBadRequest, "无效的合约地址格式")
				return
			}
			contractAddr = common.HexToAddress(contractHex)
		} else {
			contractAddr = defaultContract
		}

		ctx := r.Context()
		address := common.HexToAddress(addressHex)

		// ========== 1. 编码balanceOf方法调用数据 ==========
		// 原理：把方法名和参数按照ABI规范编码成二进制数据，作为交易的input
		// Pack方法：第一个参数是方法名，后面是方法参数，返回编码后的字节数组
		data, err := parsedABI.Pack("balanceOf", address)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("编码方法失败: %v", err))
			return
		}

		// ========== 2. 调用合约只读方法 ==========
		// CallContract：执行只读调用，不上链、不耗Gas、不需要签名
		// 对应RPC方法：eth_call
		// 第二个参数是区块号，nil表示最新区块
		result, err := client.CallContract(ctx, ethereum.CallMsg{
			To:   &contractAddr, // 调用的合约地址
			Data: data,          // 编码后的方法和参数
		}, nil)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("调用合约失败: %v", err))
			return
		}

		// ========== 3. 解码返回值 ==========
		// 把返回的二进制数据按照ABI解码成Go类型
		// balanceOf返回uint256，对应Go的*big.Int
		var balance *big.Int
		if err := parsedABI.UnpackIntoInterface(&balance, "balanceOf", result); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("解码返回值失败: %v", err))
			return
		}

		// ========== 4. 查询代币精度（可选） ==========
		// 精度是代币的小数位数，比如18位就是最小单位是1e-18
		// 用来把原始余额转成人类可读的格式
		var decimals uint8
		// 编码decimals方法，这个方法没有参数
		decimalsData, err := parsedABI.Pack("decimals")
		if err == nil {
			// 调用decimals方法
			decimalsResult, err := client.CallContract(ctx, ethereum.CallMsg{
				To:   &contractAddr,
				Data: decimalsData,
			}, nil)
			if err == nil {
				// 解码返回值，uint8类型
				_ = parsedABI.UnpackIntoInterface(&decimals, "decimals", decimalsResult)
			}
		}

		// ========== 5. 转人类可读单位 ==========
		// 计算公式：可读余额 = 原始余额 / 10^精度
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		balanceReadable := new(big.Float).Quo(new(big.Float).SetInt(balance), divisor)

		// 组装响应
		response := map[string]interface{}{
			"address":          address.Hex(),
			"contract":         contractAddr.Hex(),
			"balance":          balance.String(),                         // 原始最小单位值，精确
			"balance_readable": balanceReadable.Text('f', int(decimals)), // 人类可读值
			"decimals":         decimals,                                 // 代币精度
		}

		sendSuccess(w, response)
	}
}

// eventsHandler 事件查询接口
// 参数：store - 事件存储
// 返回：http.HandlerFunc 处理函数
// 功能：查询ERC20转账事件列表，支持地址过滤、分页
// 路由：GET /api/events?address=xxx&page=1&page_size=20
func eventsHandler(store *EventStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 读取query参数
		address := r.URL.Query().Get("address")
		pageStr := r.URL.Query().Get("page")
		pageSizeStr := r.URL.Query().Get("page_size")

		// 解析分页参数，解析失败就用默认值（Atoi失败返回0）
		page, _ := strconv.Atoi(pageStr)
		pageSize, _ := strconv.Atoi(pageSizeStr)

		// 查询事件列表
		total, data := store.List(address, page, pageSize)

		// 计算总页数
		pages := 0
		if pageSize > 0 {
			// 向上取整：总页数 = ceil(总条数 / 每页条数)
			pages = int(math.Ceil(float64(total) / float64(pageSize)))
		}
		// 修正分页参数，和List方法里的逻辑保持一致
		if page < 1 {
			page = 1
		}
		if pageSize <= 0 || pageSize > maxPageSize {
			pageSize = defaultPageSize
		}

		// 组装响应，带分页信息
		resp := ApiResponse{
			Code:     0,
			Message:  "success",
			Data:     data,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			Pages:    pages,
		}

		sendJSON(w, http.StatusOK, resp)
	}
}

// ==============================================
// 主函数：程序入口
// ==============================================

func main() {
	// ========== 1. 读取配置（从环境变量） ==========
	// 设计原则：配置和代码分离，所有环境相关的配置都从环境变量读
	// 好处：不同环境（本地、测试、生产）不用改代码，只改环境变量就行

	// 优先读ETH_WS_URL（WebSocket地址，支持订阅），没有就读ETH_RPC_URL（HTTP地址）
	// 坑点：HTTP地址不支持事件订阅，所以如果用HTTP地址，订阅会失败
	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if rpcURL == "" {
		log.Fatal("ETH_WS_URL 或 ETH_RPC_URL 环境变量必须设置")
	}

	// 要监听的ERC20合约地址，必填
	contractHex := os.Getenv("ERC20_CONTRACT")
	if contractHex == "" {
		log.Fatal("ERC20_CONTRACT 环境变量必须设置")
	}
	contractAddr := common.HexToAddress(contractHex)

	// HTTP服务监听地址，可选，默认:8080
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	// 历史扫描起始区块，可选
	// 配置了的话，启动时会从这个区块开始扫描历史Transfer事件
	var startBlock uint64 = 0
	startBlockStr := os.Getenv("ERC20_START_BLOCK")
	if startBlockStr != "" {
		parsed, err := strconv.ParseUint(startBlockStr, 10, 64)
		if err == nil {
			startBlock = parsed
		}
	}

	// ========== 2. 初始化上下文与客户端 ==========
	// 创建可取消的根上下文，程序退出时取消，通知所有子协程退出
	ctx, cancel := context.WithCancel(context.Background())
	// 函数退出时自动取消上下文，保证资源释放
	defer cancel()

	// 连接以太坊节点，DialContext支持上下文取消
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("连接以太坊节点失败: %v", err)
	}
	// 程序退出时关闭客户端连接
	defer client.Close()

	// 获取链ID，用于后续交易签名验证
	// 坑点：不同链的链ID不同，签名器必须和链ID匹配，不然解析交易发送方会错
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("获取链ID失败: %v", err)
	}
	log.Printf("连接节点成功，链ID: %s", chainID.String())

	// ========== 3. 解析ABI ==========
	// 把JSON格式的ABI字符串解析成abi.ABI对象，后续编码解码都用这个对象
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		log.Fatalf("解析ABI失败: %v", err)
	}

	// ========== 4. 初始化存储 ==========
	store := NewEventStore(defaultEventLimit)
	log.Printf("事件存储初始化完成，最大存储: %d 条", defaultEventLimit)

	// ========== 5. 历史扫描（如果配置了起始区块） ==========
	// 原理：启动时先扫一遍历史事件，补全数据，再启动实时订阅
	// 好处：服务启动后就能查到历史数据，不用等新事件
	if startBlock > 0 {
		// 先获取最新区块号
		latestBlock, err := client.BlockNumber(ctx)
		if err != nil {
			log.Fatalf("获取最新区块号失败: %v", err)
		}

		// 起始区块小于等于最新区块才扫
		if startBlock <= latestBlock {
			log.Printf("[历史扫描] 开始扫描区块 %d ~ %d 的Transfer事件...", startBlock, latestBlock)
			count, err := scanHistoricalEvents(ctx, client, parsedABI, contractAddr, store, startBlock, latestBlock)
			if err != nil {
				log.Fatalf("[历史扫描] 失败: %v", err)
			}
			log.Printf("[历史扫描] 完成，共扫描到 %d 条事件", count)
		}
	}

	// ========== 6. 启动后台事件订阅协程（带断点续传） ==========
	// 用goroutine运行，不阻塞主流程
	go subscribeTransferEvents(ctx, client, parsedABI, contractAddr, store)

	// ========== 7. 配置HTTP路由 ==========
	// 创建路由复用器
	mux := http.NewServeMux()

	// 注册路由
	// 首页
	mux.HandleFunc("/", homeHandler)
	// API接口，用闭包把依赖（client、store等）传进去
	mux.HandleFunc("/api/block/", blockHandler(client))
	mux.HandleFunc("/api/tx/", txHandler(client))
	mux.HandleFunc("/api/balance/", balanceHandler(client))
	mux.HandleFunc("/api/token-balance/", tokenBalanceHandler(client, parsedABI, contractAddr))
	mux.HandleFunc("/api/events", eventsHandler(store))

	// ========== 8. 启动HTTP服务 ==========
	// 创建HTTP服务器，配置超时时间，防止慢请求占满连接
	server := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,  // 读超时，防止客户端一直发数据占连接
		WriteTimeout: 10 * time.Second, // 写超时，防止处理太慢占连接
	}

	// 用goroutine启动HTTP服务，不阻塞主流程，主流程要等信号
	go func() {
		log.Printf("HTTP 服务启动，监听地址: %s", httpAddr)
		log.Printf("首页访问: http://localhost%s", httpAddr)
		// ListenAndServe是阻塞的，直到服务关闭
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 服务异常退出，直接fatal
			log.Fatalf("HTTP 服务错误: %v", err)
		}
	}()

	// ========== 9. 优雅退出 ==========
	// 原理：捕获系统信号（Ctrl+C、kill命令），然后平滑关闭服务
	// 好处：不会中断正在处理的请求，不会丢数据

	// 创建信号通道，缓冲1个，防止信号丢失
	sigCh := make(chan os.Signal, 1)
	// 注册要捕获的信号：SIGINT（Ctrl+C）、SIGTERM（kill命令）
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号，收到信号才继续往下走
	sig := <-sigCh
	fmt.Printf("\n收到信号 %s，正在优雅关闭...\n", sig.String())

	// 第一步：关闭HTTP服务
	// Shutdown会等所有正在处理的请求完成，再关闭服务
	// 给5秒超时，超时就强制关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP 服务关闭错误: %v", err)
	}

	// 第二步：取消根上下文，通知所有子协程（比如订阅协程）退出
	cancel()

	// 第三步：等1秒，给订阅协程时间清理资源
	time.Sleep(1 * time.Second)

	log.Println("服务已完全退出")
}
