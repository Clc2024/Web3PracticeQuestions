package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	mode := flag.String("mode", "01", "运行模式：-mode 01 或 02")
	txHashHex := flag.String("tx", "", "transaction hash (for query mode)")
	sendMode := flag.Bool("send", false, "enable send transaction mode")
	toAddrHex := flag.String("to", "", "recipient address (required for send mode)")
	amountEth := flag.Float64("amount", 0, "amount in ETH (required for send mode)")

	blockNumberFlag := flag.Uint64("number", 0, "block number to query (0 means skip)")

	flag.Parse()
	switch *mode {
	case "01":
		log.Println("Starting 查询区块")
		blockOps(blockNumberFlag)
	case "02":
		log.Println("Starting 发送交易")
		txOps(txHashHex, sendMode, toAddrHex, amountEth)

	default:
		log.Fatalf("Unknown mode: %s . User 'server' or 'client'\n", *mode)
		os.Exit(1)
	}
}

// 支持两种操作模式：
// 1. 查询交易：--tx <hash> - 按哈希查询交易与回执，解析关键字段
// 2. 发送交易：--send --to <address> --amount <eth> - 发起 ETH 转账交易
func txOps(txHashHex *string, sendMode *bool, toAddrHex *string, amountEth *float64) {
	// txHashHex := flag.String("tx", "", "transaction hash (for query mode)")
	// sendMode := flag.Bool("send", false, "enable send transaction mode")
	// toAddrHex := flag.String("to", "", "recipient address (required for send mode)")
	// amountEth := flag.Float64("amount", 0, "amount in ETH (required for send mode)")
	// flag.Parse()

	// 判断操作模式
	if *sendMode {
		// 发送交易模式
		if *toAddrHex == "" || *amountEth <= 0 {
			log.Fatal("send mode requires --to and --amount flags")
		}
		sendTransaction(*toAddrHex, *amountEth)
	} else {
		// 查询交易模式
		if *txHashHex == "" {
			log.Fatal("query mode requires --tx flag, or use --send for send mode")
		}
		queryTransaction(*txHashHex)
	}

}

// 发送交易
func sendTransaction(toAddrHex string, amountEth float64) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}

	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("SENDER_PRIVATE_KEY is not set (required for send mode)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	// 解析私钥
	privKey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}

	// 获取发送方地址
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddr := common.HexToAddress(toAddrHex)

	// 获取链 ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("failed to get gas tip cap: %v", err)
	}

	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("failed to get header: %v", err)
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gas price
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("failed to get gas price: %v", err)
		}
		baseFee = gasPrice
	}

	// fee cap = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)

	// 估算 Gas Limit（普通转账固定为 21000）
	gasLimit := uint64(21000)

	// 转换 ETH 金额为 Wei
	// amountEth * 1e18
	amountWei := new(big.Float).Mul(
		big.NewFloat(amountEth),
		big.NewFloat(1e18),
	)
	valueWei, _ := amountWei.Int(nil)

	// 检查余额是否足够
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}

	// 计算总费用：value + gasFeeCap * gasLimit
	totalCost := new(big.Int).Add(
		valueWei,
		new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit))),
	)

	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("insufficient balance: have %s wei, need %s wei", balance.String(), totalCost.String())
	}

	// 构造交易（EIP-1559 动态费用交易）
	txData := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     valueWei,
		Data:      nil,
	}
	tx := types.NewTx(txData)

	// 签名交易
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	// 发送交易
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}

	// 输出交易信息
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountEth), valueWei.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap: %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap: %s Wei\n", gasFeeCap.String())
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
	fmt.Println("\nTransaction is pending. Use --tx flag to query status:")
	fmt.Printf("  go run main.go --tx %s\n", signedTx.Hash().Hex())
}

// 查询交易
func queryTransaction(txHashHex string) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		panic("ETH_RPC_URL environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to dial Ethereum RPC: %v", err)
	}
	defer client.Close()

	txHash := common.HexToHash(txHashHex)
	tx, isPending, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		log.Fatalf("Failed to get transaction by hash: %v", err)
	}
	fmt.Print("Transaction Details: \n\n")
	printTxBasicInfo(tx, isPending)

	fmt.Print("\n\n")
	// 查询交易回执
	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		log.Fatalf("Failed to get transaction receipt: %v", err)
	}
	fmt.Print("Transaction Receipt: \n\n")
	printReceiptInfo(receipt)
}

func printTxBasicInfo(tx *types.Transaction, isPending bool) {
	fmt.Printf("Hash        : %s\n", tx.Hash().Hex())
	fmt.Printf("Nonce       : %d\n", tx.Nonce())
	fmt.Printf("Gas         : %d\n", tx.Gas())
	fmt.Printf("Gas Price   : %s\n", tx.GasPrice().String())
	fmt.Printf("To          : %v\n", tx.To())
	fmt.Printf("Value (Wei) : %s\n", tx.Value().String())
	fmt.Printf("Data Len    : %d bytes\n", len(tx.Data()))
	fmt.Printf("Pending     : %v\n", isPending)
}

func printReceiptInfo(r *types.Receipt) {
	fmt.Printf("Status      : %d\n", r.Status)
	fmt.Printf("BlockNumber : %d\n", r.BlockNumber.Uint64())
	fmt.Printf("BlockHash   : %s\n", r.BlockHash.Hex())
	fmt.Printf("TxIndex     : %d\n", r.TransactionIndex)
	fmt.Printf("Gas Used    : %d\n", r.GasUsed)
	fmt.Printf("Logs        : %d\n", len(r.Logs))
	if len(r.Logs) > 0 {
		fmt.Printf("First Log Address : %s\n", r.Logs[0].Address.Hex())
	}
}

// trim0x 移除十六进制字符串前缀 "0x"
func trim0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

// go run main.go -number 11136690
// 查询 指定区块 信息。
func blockOps(blockNumberFlag *uint64) {
	// blockNumberFlag := flag.Uint64("number", 0, "block number to query (0 means skip)")
	// flag.Parse()

	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		panic("ETH_RPC_URL environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 初始化 Ethereum 客户端
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to dial Ethereum RPC: %v", err)
	}
	defer client.Close()

	// 查询最新区块
	// latestBlock, err := client.BlockByNumber(ctx, nil)
	// if err != nil {
	// 	log.Fatalf("Failed to get latest block number: %v", err)
	// }
	// printBlockInfo("Latest Block", latestBlock)

	// 查询指定区块信息
	if *blockNumberFlag > 0 {
		num := big.NewInt(0).SetUint64(*blockNumberFlag)
		block, err := fetchBlockWithRetry(ctx, client, num, 3)
		if err != nil {
			log.Fatalf("Failed to fetch block %s: %v", num.String(), err)
		}
		printBlockInfo(fmt.Sprintf("Block %d", *blockNumberFlag), block)
	}
}

// fetchBlockWithRetry 带重试机制的区块查询
func fetchBlockWithRetry(ctx context.Context, client *ethclient.Client,
	blockNumber *big.Int, maxRetries int) (*types.Block, error) {

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		block, err := client.BlockByNumber(reqCtx, blockNumber)
		cancel()

		if err == nil {
			return block, nil
		}
		lastErr = err
		if i < maxRetries-1 {
			backoff := time.Duration(i+1) * 500 * time.Millisecond
			log.Printf("[WARN] Failed to fetch block %s, retry %d/%d,after %v:%v",
				blockNumber.String(), i+1, maxRetries, backoff, err)
			// 等待 backoff 时间后重试
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("failed after %d retries, last error: %v", maxRetries, lastErr)
}

// printBlockInfo 打印区块信息
func printBlockInfo(title string, block *types.Block) {
	fmt.Println("======================================")
	fmt.Println(title)
	fmt.Println("======================================")
	fmt.Printf("Block: %+v\n", block)

	// 基本信息
	fmt.Printf("Number       : %d\n", block.Number().Uint64())
	fmt.Printf("Hash         : %s\n", block.Hash().Hex())
	fmt.Printf("Parent Hash  : %s\n", block.ParentHash().Hex())

	// 时间信息
	blockTime := time.Unix(int64(block.Time()), 0)
	fmt.Printf("Time         : %s\n", blockTime.Format(time.RFC3339))
	fmt.Printf("Time (Local) : %s\n", blockTime.Local().Format("2006-01-02 15:04:05 MST"))

	// Gas 信息
	gasUsed := block.GasUsed()
	gasLimit := block.GasLimit()
	gasUsagePercent := float64(gasUsed) / float64(gasLimit) * 100
	fmt.Printf("Gas Used     : %d (%.2f%%)\n", gasUsed, gasUsagePercent)
	fmt.Printf("Gas Limit    : %d\n", gasLimit)

	// 交易信息
	txCount := len(block.Transactions())
	fmt.Printf("Tx Count     : %d\n", txCount)

	// 区块根信息（Merkle 树根）
	fmt.Printf("State Root   : %s\n", block.Root().Hex())
	fmt.Printf("Tx Root      : %s\n", block.TxHash().Hex())
	fmt.Printf("Receipt Root : %s\n", block.ReceiptHash().Hex())

	// 区块大小估算（简化版，实际大小还包括其他字段）
	if txCount > 0 {
		fmt.Printf("\nFirst Tx Hash: %s\n", block.Transactions()[0].Hash().Hex())
		if txCount > 1 {
			fmt.Printf("Last Tx Hash : %s\n", block.Transactions()[txCount-1].Hash().Hex())
		}
	}

	// 难度信息（PoW 相关，PoS 后基本固定）
	fmt.Printf("Difficulty   : %s\n", block.Difficulty().String())

	// 区块奖励相关信息
	coinbase := block.Coinbase()
	if coinbase != (common.Address{}) {
		fmt.Printf("Coinbase     : %s\n", coinbase.Hex())
	}

	fmt.Println("======================================")
	fmt.Println()
}
