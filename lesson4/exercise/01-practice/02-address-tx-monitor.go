package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ERC20转账事件的ABI（仅需解析Transfer事件）
const erc20ABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

var (
	erc20ABIObject abi.ABI
	// ERC20 Transfer事件签名（keccak256("Transfer(address,address,uint256)")）
	transferEventSig = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
)

func init() {
	// 初始化ERC20 ABI
	var err error
	erc20ABIObject, err = abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		log.Fatalf("failed to parse ERC20 ABI: %v", err)
	}
}

/*
功能：指定地址交易监控脚本（修复版）
核心优化：
1. 区分原生ETH交易和ERC20代币交易
2. 解析ERC20 Transfer事件，正确识别转账的from/to
3. 修复To地址为空（合约创建）的边界问题
4. 补充交易输入数据解析，适配代币转账场景
*/
func address_tx_monitor() {
	// ========== 1. 读取配置 ==========
	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if rpcURL == "" {
		log.Fatal("ETH_WS_URL or ETH_RPC_URL is not set")
	}

	targetAddrHex := os.Getenv("TARGET_ADDRESS")
	if targetAddrHex == "" {
		log.Fatal("TARGET_ADDRESS environment variable is not set")
	}
	targetAddr := common.HexToAddress(targetAddrHex)

	// ========== 2. 初始化连接与上下文 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to node: %v", err)
	}
	defer client.Close()

	// 获取链ID，用于交易签名恢复
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain ID: %v", err)
	}
	signer := types.LatestSignerForChainID(chainID)

	// ========== 3. 订阅新区块头 ==========
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatalf("failed to subscribe new head: %v", err)
	}
	defer sub.Unsubscribe()

	log.Printf("monitoring transactions for address: %s", targetAddr.Hex())
	log.Println("waiting for new blocks...")

	// ========== 4. 信号捕获 ==========
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ========== 5. 主循环处理 ==========
	for {
		select {
		case header := <-headers:
			// 根据区块号获取完整区块（包含所有交易）
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(header.Number.Uint64())))
			if err != nil {
				log.Printf("failed to get block %d: %v", header.Number.Uint64(), err)
				continue
			}

			log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			log.Printf("new block #%d, transactions: %d", block.NumberU64(), len(block.Transactions()))

			// 遍历区块内所有交易
			for _, tx := range block.Transactions() {
				// 从交易签名中恢复发送方地址（交易发起者）
				from, err := signer.Sender(tx)
				if err != nil {
					continue // 签名无效的交易跳过
				}

				// 处理原生ETH交易
				handleNativeETHTransaction(tx, from, targetAddr)

				// 处理ERC20代币交易（解析Transfer事件）
				handleERC20Transaction(ctx, client, tx, block.Hash(), block.NumberU64(), targetAddr)
			}

		case err := <-sub.Err():
			log.Fatalf("subscription error: %v", err)
			return

		case sig := <-sigCh:
			log.Printf("received signal %s, shutting down...", sig)
			return

		case <-ctx.Done():
			log.Println("context cancelled, exit")
			return
		}
	}
}

// 处理原生ETH交易（To/From直接匹配目标地址）
func handleNativeETHTransaction(tx *types.Transaction, from common.Address, targetAddr common.Address) {
	to := tx.To()
	var toAddr common.Address
	if to != nil {
		toAddr = *to
	}

	// 匹配：from是目标地址 或 to是目标地址
	isMatch := from == targetAddr || (to != nil && toAddr == targetAddr)
	if !isMatch {
		return
	}

	// 打印原生ETH交易信息
	direction := "IN"
	if from == targetAddr {
		direction = "OUT"
	}
	valueEth := new(big.Float).Quo(
		new(big.Float).SetInt(tx.Value()),
		new(big.Float).SetFloat64(1e18),
	)

	fmt.Printf("  [ETH-%s] Tx: %s\n", direction, tx.Hash().Hex())
	fmt.Printf("    From:  %s\n", from.Hex())
	fmt.Printf("    To:    %s\n", toAddr.Hex())
	fmt.Printf("    Value: %s ETH\n", valueEth.Text('f', 6))
	fmt.Printf("    Gas:   %d\n", tx.Gas())
}

// 处理ERC20代币交易（解析Transfer事件）
func handleERC20Transaction(ctx context.Context, client *ethclient.Client, tx *types.Transaction, blockHash common.Hash, blockNum uint64, targetAddr common.Address) {
	// 1. 获取交易收据（包含事件日志）
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		// 交易未确认/失败时跳过
		return
	}

	// 2. 遍历交易日志，寻找ERC20 Transfer事件
	for _, log := range receipt.Logs {
		// 过滤：日志签名必须是Transfer事件
		if log.Topics[0] != transferEventSig {
			continue
		}

		// 解析Transfer事件参数
		var transferEvent struct {
			From  common.Address
			To    common.Address
			Value *big.Int
		}
		if len(log.Topics) < 3 {
			continue
		}
		// Topics[1] = from（indexed）, Topics[2] = to（indexed）
		transferEvent.From = common.HexToAddress(log.Topics[1].Hex()[26:])
		transferEvent.To = common.HexToAddress(log.Topics[2].Hex()[26:])
		// Data = value（非indexed）
		if len(log.Data) >= 32 {
			transferEvent.Value = new(big.Int).SetBytes(log.Data[:32])
		} else {
			continue
		}

		// 匹配目标地址（from或to）
		isMatch := transferEvent.From == targetAddr || transferEvent.To == targetAddr
		if !isMatch {
			continue
		}

		// 计算代币金额（默认18位小数，可根据合约调整）
		tokenValue := new(big.Float).Quo(
			new(big.Float).SetInt(transferEvent.Value),
			new(big.Float).SetFloat64(1e18),
		)

		// 打印ERC20交易信息
		direction := "IN"
		if transferEvent.From == targetAddr {
			direction = "OUT"
		}
		fmt.Printf("  [ERC20-%s] Tx: %s\n", direction, tx.Hash().Hex())
		fmt.Printf("    Token Contract: %s\n", log.Address.Hex())
		fmt.Printf("    From:           %s\n", transferEvent.From.Hex())
		fmt.Printf("    To:             %s\n", transferEvent.To.Hex())
		fmt.Printf("    Value:          %s Token\n", tokenValue.Text('f', 6))
		fmt.Printf("    Block:          %d\n", blockNum)
	}
}
