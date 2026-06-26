package main

import (
	"04-task/bindings" // 替换为你的模块路径
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 未找到 .env 文件: %v", err)
	}

	// ========== 1. 连接 Sepolia 测试网 ==========
	rpcURL := os.Getenv("SEPOLIA_RPC")
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("连接 RPC 失败: %v", err)
	}
	defer client.Close()
	fmt.Println("✅ 成功连接到 Sepolia 测试网")

	// 获取链 ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("获取链 ID 失败: %v", err)
	}
	fmt.Printf("🔗 链 ID: %s\n", chainID)

	// ========== 2. 加载私钥和创建交易签名器 ==========
	privateKeyHex := strings.TrimPrefix(os.Getenv("PRIVATE_KEY"), "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("创建交易签名器失败: %v", err)
	}

	// 获取钱包地址
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Printf("👛 钱包地址: %s\n", fromAddress.Hex())

	// 查询 ETH 余额
	ethBalance, err := client.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		log.Fatalf("查询 ETH 余额失败: %v", err)
	}
	ethBalanceFloat := new(big.Float).Quo(new(big.Float).SetInt(ethBalance), big.NewFloat(1e18))
	fmt.Printf("💰 ETH 余额: %.4f ETH\n", ethBalanceFloat)

	// ========== 3. 加载 ERC20 合约实例 ==========
	contractAddress := common.HexToAddress(os.Getenv("CONTRACT_ADDRESS"))
	token, err := bindings.NewMyERC20(contractAddress, client)
	if err != nil {
		log.Fatalf("加载合约失败: %v", err)
	}
	fmt.Printf("📜 ERC20 合约已加载: %s\n", contractAddress.Hex())

	// ========== 4. 调用只读方法（View Functions） ==========
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("📊 调用只读方法")
	fmt.Println("═══════════════════════════════════════")

	// 4.1 获取代币名称
	name, err := token.Name(&bind.CallOpts{})
	if err != nil {
		log.Fatalf("获取代币名称失败: %v", err)
	}
	fmt.Printf("🏷️  代币名称: %s\n", name)

	// 4.2 获取代币符号
	symbol, err := token.Symbol(&bind.CallOpts{})
	if err != nil {
		log.Fatalf("获取代币符号失败: %v", err)
	}
	fmt.Printf("🔤 代币符号: %s\n", symbol)

	// 4.3 获取小数位数
	decimals, err := token.Decimals(&bind.CallOpts{})
	if err != nil {
		log.Fatalf("获取小数位数失败: %v", err)
	}
	fmt.Printf("🔢 小数位数: %d\n", decimals)

	// 4.4 获取总供应量
	totalSupply, err := token.TotalSupply(&bind.CallOpts{})
	if err != nil {
		log.Fatalf("获取总供应量失败: %v", err)
	}
	// 转换为人类可读格式（除以 10^decimals）
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	totalSupplyFloat := new(big.Float).Quo(new(big.Float).SetInt(totalSupply), divisor)
	fmt.Printf("📦 总供应量: %.2f %s\n", totalSupplyFloat, symbol)

	// 4.5 查询当前钱包余额
	balance, err := token.BalanceOf(&bind.CallOpts{}, fromAddress)
	if err != nil {
		log.Fatalf("查询余额失败: %v", err)
	}
	balanceFloat := new(big.Float).Quo(new(big.Float).SetInt(balance), divisor)
	fmt.Printf("💵 当前钱包余额: %.2f %s\n", balanceFloat, symbol)

	// ========== 5. 调用写方法（Write Functions） ==========
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("✍️  调用写方法")
	fmt.Println("═══════════════════════════════════════")

	// 5.1 铸造代币（如果合约支持 mint）
	fmt.Println("\n🏭 正在铸造 100 代币...")
	mintAmount := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	tx, err := token.Mint(auth, fromAddress, mintAmount)
	if err != nil {
		log.Printf("⚠️  铸造失败（可能没有权限）: %v", err)
	} else {
		fmt.Printf("   交易哈希: %s\n", tx.Hash().Hex())
		fmt.Println("   等待交易上链...")

		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if err != nil {
			log.Fatalf("等待交易确认失败: %v", err)
		}
		fmt.Printf("   ✅ 铸造成功，Gas 消耗: %d\n", receipt.GasUsed)

		// 查询铸造后的余额
		newBalance, _ := token.BalanceOf(&bind.CallOpts{}, fromAddress)
		newBalanceFloat := new(big.Float).Quo(new(big.Float).SetInt(newBalance), divisor)
		fmt.Printf("   💵 铸造后余额: %.2f %s\n", newBalanceFloat, symbol)
	}

	// 5.2 转账代币
	recipient := common.HexToAddress(os.Getenv("RECIPIENT_ADDRESS"))
	transferAmount := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))

	fmt.Printf("\n💸 正在转账 %.2f %s 到 %s...\n", float64(10), symbol, recipient.Hex()[:10]+"...")
	tx, err = token.Transfer(auth, recipient, transferAmount)
	if err != nil {
		log.Fatalf("转账失败: %v", err)
	}
	fmt.Printf("   交易哈希: %s\n", tx.Hash().Hex())
	fmt.Println("   等待交易上链...")

	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatalf("等待交易确认失败: %v", err)
	}
	fmt.Printf("   ✅ 转账成功，Gas 消耗: %d\n", receipt.GasUsed)

	// 查询转账后的余额
	balanceAfter, _ := token.BalanceOf(&bind.CallOpts{}, fromAddress)
	balanceAfterFloat := new(big.Float).Quo(new(big.Float).SetInt(balanceAfter), divisor)
	fmt.Printf("   💵 转账后余额: %.2f %s\n", balanceAfterFloat, symbol)

	// 查询接收方余额
	recipientBalance, _ := token.BalanceOf(&bind.CallOpts{}, recipient)
	recipientBalanceFloat := new(big.Float).Quo(new(big.Float).SetInt(recipientBalance), divisor)
	fmt.Printf("   📥 接收方余额: %.2f %s\n", recipientBalanceFloat, symbol)

	// 5.3 授权（Approve）
	fmt.Println("\n🔐 正在授权 50 代币给接收方...")
	approveAmount := new(big.Int).Mul(big.NewInt(50), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	tx, err = token.Approve(auth, recipient, approveAmount)
	if err != nil {
		log.Fatalf("授权失败: %v", err)
	}
	fmt.Printf("   交易哈希: %s\n", tx.Hash().Hex())
	fmt.Println("   等待交易上链...")

	receipt, err = bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatalf("等待交易确认失败: %v", err)
	}
	fmt.Printf("   ✅ 授权成功，Gas 消耗: %d\n", receipt.GasUsed)

	// 查询授权额度
	allowance, _ := token.Allowance(&bind.CallOpts{}, fromAddress, recipient)
	allowanceFloat := new(big.Float).Quo(new(big.Float).SetInt(allowance), divisor)
	fmt.Printf("   📋 授权额度: %.2f %s\n", allowanceFloat, symbol)

	// ========== 6. 监听事件 ==========
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("👂 开始监听 Transfer 事件（按 Ctrl+C 退出）")
	fmt.Println("═══════════════════════════════════════")

	// 创建事件通道
	transferEvents := make(chan *bindings.MyERC20Transfer)
	approvalEvents := make(chan *bindings.MyERC20Approval)

	// 监听 Transfer 事件
	transferSub, err := token.WatchTransfer(&bind.WatchOpts{Context: context.Background()}, transferEvents, nil, nil)
	if err != nil {
		log.Fatalf("监听 Transfer 事件失败: %v", err)
	}
	defer transferSub.Unsubscribe()

	// 监听 Approval 事件
	approvalSub, err := token.WatchApproval(&bind.WatchOpts{Context: context.Background()}, approvalEvents, nil, nil)
	if err != nil {
		log.Fatalf("监听 Approval 事件失败: %v", err)
	}
	defer approvalSub.Unsubscribe()

	// 持续监听
	for {
		select {
		case err := <-transferSub.Err():
			log.Printf("Transfer 订阅错误: %v", err)
		case err := <-approvalSub.Err():
			log.Printf("Approval 订阅错误: %v", err)
		case event := <-transferEvents:
			fmt.Printf("\n📢 【Transfer 事件】")
			fmt.Printf("\n   发送方: %s", event.From.Hex())
			fmt.Printf("\n   接收方: %s", event.To.Hex())
			amountFloat := new(big.Float).Quo(new(big.Float).SetInt(event.Value), divisor)
			fmt.Printf("\n   数量: %.2f %s\n", amountFloat, symbol)
		case event := <-approvalEvents:
			fmt.Printf("\n📢 【Approval 事件】")
			fmt.Printf("\n   授权人: %s", event.Owner.Hex())
			fmt.Printf("\n   被授权人: %s", event.Spender.Hex())
			amountFloat := new(big.Float).Quo(new(big.Float).SetInt(event.Value), divisor)
			fmt.Printf("\n   额度: %.2f %s\n", amountFloat, symbol)
		}
	}
}
