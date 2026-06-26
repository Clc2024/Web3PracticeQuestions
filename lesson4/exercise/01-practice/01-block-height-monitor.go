package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

/*
功能：区块高度监控脚本
核心逻辑：
1. 定时轮询最新区块高度
2. 记录上次区块更新时间
3. 长时间块高不变则触发告警
4. 支持Ctrl+C优雅退出
*/
func block_height_monitor() {
	// ========== 1. 读取配置 ==========
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable is not set")
	}

	// 监控参数（可根据需求调整）
	checkInterval := 10 * time.Second  // 检查间隔：10秒
	alertThreshold := 30 * time.Second // 告警阈值：30秒没出新块就告警

	// ========== 2. 初始化上下文与节点连接 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to node: %v", err)
	}
	defer client.Close()

	// ========== 3. 初始化状态变量 ==========
	var lastBlockNumber uint64
	lastUpdateTime := time.Now()

	// ========== 4. 启动定时检查 ==========
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// 信号捕获，优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Println("block height monitor started, check interval:", checkInterval)
	log.Println("alert threshold (no new block):", alertThreshold)

	for {
		select {
		case <-ticker.C:
			// 查询最新区块号
			blockNum, err := client.BlockNumber(ctx)
			if err != nil {
				log.Printf("[ERROR] failed to get block number: %v", err)
				continue
			}

			// 块高发生了变化
			if blockNum != lastBlockNumber {
				log.Printf("[INFO] latest block: %d", blockNum)
				lastBlockNumber = blockNum
				lastUpdateTime = time.Now()
				continue
			}

			// 块高没变化，检查是否超过告警阈值
			elapsed := time.Since(lastUpdateTime)
			if elapsed > alertThreshold {
				log.Printf("[WARN] block height stagnant! current: %d, unchanged for %v",
					blockNum, elapsed.Round(time.Second))
			}

		case sig := <-sigCh:
			log.Printf("received signal %s, shutting down monitor...", sig)
			return

		case <-ctx.Done():
			log.Println("context cancelled, exit")
			return
		}
	}
}
