package main

import (
	"context"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ERC20 Transfer事件ABI
const erc20ABIJSON = `[{"anonymous":false,"name":"Transfer","type":"event","inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}]}]`

// TransferEvent 转账事件数据模型
type TransferEvent struct {
	BlockNumber uint64    `json:"blockNumber"`
	TxHash      string    `json:"txHash"`
	Contract    string    `json:"contract"` // 所属合约地址
	From        string    `json:"from"`
	To          string    `json:"to"`
	Value       string    `json:"value"`
	Timestamp   time.Time `json:"timestamp"`
}

/*
AddressIndex 按地址维度的转账索引
核心设计：
1. key: 钱包地址字符串
2. value: 该地址相关的所有转账事件（转入+转出）
3. 读写锁保证并发安全
4. 每个地址最多保留100条，环形缓冲避免内存泄漏
*/
type AddressIndex struct {
	mu      sync.RWMutex
	records map[string][]TransferEvent // address -> 转账事件列表
	limit   int                        // 每个地址最多保留条数
}

// NewAddressIndex 构造函数
func NewAddressIndex(limitPerAddr int) *AddressIndex {
	return &AddressIndex{
		records: make(map[string][]TransferEvent),
		limit:   limitPerAddr,
	}
}

// AddEvent 添加事件，同时给from和to两个地址都建立索引
func (idx *AddressIndex) AddEvent(event TransferEvent) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 给转出地址添加记录
	idx.addToAddress(event.From, event)
	// 给转入地址添加记录
	idx.addToAddress(event.To, event)
}

// addToAddress 内部方法：给指定地址追加事件，环形缓冲
func (idx *AddressIndex) addToAddress(addr string, event TransferEvent) {
	list := idx.records[addr]
	list = append(list, event)
	// 超过限制就丢弃最旧的
	if len(list) > idx.limit {
		list = list[1:]
	}
	idx.records[addr] = list
}

// GetByAddress 查询某个地址的所有转账记录
func (idx *AddressIndex) GetByAddress(addr string) []TransferEvent {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	list := idx.records[strings.ToLower(addr)]
	out := make([]TransferEvent, len(list))
	copy(out, list)
	return out
}

func multi_erc20_monitor() {
	// ========== 1. 读取配置 ==========
	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if rpcURL == "" {
		log.Fatal("ETH_WS_URL or ETH_RPC_URL is not set")
	}

	contractsStr := os.Getenv("ERC20_CONTRACTS")
	if contractsStr == "" {
		log.Fatal("ERC20_CONTRACTS is not set (comma separated)")
	}
	// 逗号分割多个合约地址
	contractHexList := strings.Split(contractsStr, ",")
	var contractAddrs []common.Address
	for _, hex := range contractHexList {
		hex = strings.TrimSpace(hex)
		if hex == "" {
			continue
		}
		contractAddrs = append(contractAddrs, common.HexToAddress(hex))
	}
	if len(contractAddrs) == 0 {
		log.Fatal("no valid contract addresses")
	}

	// ========== 2. 初始化 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to node: %v", err)
	}
	defer client.Close()

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		log.Fatalf("failed to parse ABI: %v", err)
	}

	// 初始化地址索引，每个地址最多保留100条
	addrIndex := NewAddressIndex(100)

	// ========== 3. 订阅多合约事件 ==========
	query := ethereum.FilterQuery{
		Addresses: contractAddrs, // 传入多个合约地址，同时监听
	}

	logsCh := make(chan types.Log, 200)
	sub, err := client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		log.Fatalf("failed to subscribe logs: %v", err)
	}
	defer sub.Unsubscribe()

	log.Printf("subscribed to %d ERC20 contracts", len(contractAddrs))
	for _, addr := range contractAddrs {
		log.Printf("  - %s", addr.Hex())
	}

	// ========== 4. 信号捕获 ==========
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ========== 5. 事件处理主循环 ==========
	log.Println("listening for Transfer events...")

	for {
		select {
		case vLog := <-logsCh:
			if len(vLog.Topics) < 3 {
				continue
			}

			// 解析非indexed参数（value）
			var event struct {
				Value *big.Int
			}
			if err := parsedABI.UnpackIntoInterface(&event, "Transfer", vLog.Data); err != nil {
				log.Printf("failed to unpack log: %v", err)
				continue
			}

			// 解析indexed参数（from、to）
			from := common.BytesToAddress(vLog.Topics[1].Bytes())
			to := common.BytesToAddress(vLog.Topics[2].Bytes())

			// 构造事件对象
			transferEvt := TransferEvent{
				BlockNumber: vLog.BlockNumber,
				TxHash:      vLog.TxHash.Hex(),
				Contract:    vLog.Address.Hex(),
				From:        from.Hex(),
				To:          to.Hex(),
				Value:       event.Value.String(),
				Timestamp:   time.Now(),
			}

			// 存入地址索引
			addrIndex.AddEvent(transferEvt)

			// 打印日志
			log.Printf("[Transfer] %s -> %s, value: %s, contract: %s, block: %d",
				from.Hex(), to.Hex(), event.Value.String(), vLog.Address.Hex(), vLog.BlockNumber)

		case err := <-sub.Err():
			log.Fatalf("subscription error: %v", err)
			return

		case sig := <-sigCh:
			log.Printf("received signal %s, shutting down...", sig)
			// 退出前可以打印统计信息
			addrIndex.mu.RLock()
			log.Printf("total indexed addresses: %d", len(addrIndex.records))
			addrIndex.mu.RUnlock()
			return

		case <-ctx.Done():
			log.Println("context cancelled, exit")
			return
		}
	}
}
