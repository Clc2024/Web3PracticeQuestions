# ERC20 合约交互实战：Go \+ Sepolia 完整教程

---

# 二、环境准备

## 2\.1 前置条件

- Go 1\.18\+ 已安装

- Sepolia 测试网 RPC 端点（Alchemy / Infura / QuickNode）

- 测试网 ETH（用于支付 Gas）

- 钱包私钥（用于签名交易）

- 已部署的 ERC20 合约地址

## 2\.2 项目结构

```plaintext
my-erc20-project/
├── bindings/               # abigen 生成的 Go 绑定代码
│   └── MyERC20.go          # 你已有的文件
├── cmd/
│   └── main.go             # 主程序：与合约交互
├── .env                    # 环境变量
├── go.mod
└── go.sum
```

## 2\.3 配置环境变量

创建 `.env` 文件：

```dotenv
# Sepolia RPC 端点
SEPOLIA_RPC=https://sepolia.infura.io/v3/YOUR_INFURA_KEY

# 钱包私钥（不含 0x 前缀）
PRIVATE_KEY=your_private_key_here

# 已部署的 ERC20 合约地址
CONTRACT_ADDRESS=0xYourERC20ContractAddress

# 接收地址（测试转账用）
RECIPIENT_ADDRESS=0xRecipientAddress
```

**安全提醒：**切勿将 `.env` 文件提交到 Git，务必添加到 `.gitignore` 中！

## 2\.4 安装依赖

```powershell
# 初始化 Go 模块（如果还没有）
go mod init my-erc20-project

# 安装 go-ethereum
go get github.com/ethereum/go-ethereum

# 安装环境变量加载库
go get github.com/joho/godotenv

# 同步依赖
go mod tidy
```

---

# 三、完整的 Go 交互脚本

## 3\.1 主程序代码

创建 `cmd/main.go`：

```go
package main

import (
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

	"my-erc20-project/bindings" // 替换为你的模块路径
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
```

---

# 四、运行步骤

## 4\.1 部署合约（如果还没有部署）

1. 确保 `MyERC20.go` 中包含合约字节码（bin）

2. 使用 `bindings.DeployMyERC20()` 方法部署合约

3. 将部署后的合约地址填入 `.env`

```go
// 部署 ERC20 合约
name := "My Token"
symbol := "MTK"
initialSupply := big.NewInt(1000000) // 初始供应量（最小单位）
recipient := fromAddress // 接收初始供应的地址

contractAddress, tx, _, err := bindings.DeployMyERC20(auth, client, name, symbol, initialSupply, recipient)
if err != nil {
    log.Fatalf("部署合约失败: %v", err)
}
fmt.Printf("🚀 合约部署中... 交易哈希: %s\n", tx.Hash().Hex())
fmt.Printf("📝 合约地址: %s\n", contractAddress.Hex())
```

## 4\.2 运行交互脚本

```powershell
# 运行主程序
go run main.go
```

## 4\.3 预期输出

```plaintext
✅ 成功连接到 Sepolia 测试网
🔗 链 ID: 11155111
👛 钱包地址: 0x...
💰 ETH 余额: 0.5000 ETH
📜 ERC20 合约已加载: 0x...

═══════════════════════════════════════
📊 调用只读方法
═══════════════════════════════════════
🏷️  代币名称: My Token
🔤 代币符号: MTK
🔢 小数位数: 18
📦 总供应量: 1000000.00 MTK
💵 当前钱包余额: 1000000.00 MTK

═══════════════════════════════════════
✍️  调用写方法
═══════════════════════════════════════

🏭 正在铸造 100 代币...
   交易哈希: 0x...
   等待交易上链...
   ✅ 铸造成功，Gas 消耗: 43210
   💵 铸造后余额: 1000100.00 MTK

💸 正在转账 10.00 MTK 到 0x1234...
   交易哈希: 0x...
   等待交易上链...
   ✅ 转账成功，Gas 消耗: 51234
   💵 转账后余额: 1000090.00 MTK
   📥 接收方余额: 10.00 MTK

🔐 正在授权 50 代币给接收方...
   交易哈希: 0x...
   等待交易上链...
   ✅ 授权成功，Gas 消耗: 45678
   📋 授权额度: 50.00 MTK

═══════════════════════════════════════
👂 开始监听 Transfer 事件（按 Ctrl+C 退出）
═══════════════════════════════════════
```

---

# 五、常见问题排查

| 问题                         | 解决方案                                             |
| ---------------------------- | ---------------------------------------------------- |
| `insufficient funds`         | 钱包 ETH 余额不足，去 Sepolia 水龙头领取测试 ETH     |
| `nonce too low`              | 同一笔交易重复发送，等待上一笔确认或提高 nonce       |
| `execution reverted`         | 合约执行失败，可能是余额不足或没有权限（如 mint）    |
| `ERC20InsufficientBalance`   | 转账金额超过余额，检查余额是否足够                   |
| `ERC20InsufficientAllowance` | transferFrom 时授权额度不足，先调用 approve          |
| 连接 RPC 超时                | 检查 RPC URL 是否正确，或更换 RPC 服务商             |
| 私钥格式错误                 | 确保私钥是 64 位十六进制，不含 0x 前缀               |
| mint 失败                    | mint 通常需要 owner 权限，确认调用者是否为合约所有者 |

---

# 六、进阶技巧

## 6\.1 Gas 价格优化

```go
// 获取建议的 Gas Price
gasPrice, err := client.SuggestGasPrice(context.Background())
if err != nil {
    log.Fatal(err)
}

// 设置 Gas Price（可适当提高以加快确认）
auth.GasPrice = gasPrice

// 设置 Gas Limit
auth.GasLimit = uint64(300000)
```

## 6\.2 批量查询余额

```go
addresses := []common.Address{addr1, addr2, addr3}
for _, addr := range addresses {
    balance, err := token.BalanceOf(&bind.CallOpts{}, addr)
    if err != nil {
        log.Printf("查询 %s 失败: %v", addr.Hex(), err)
        continue
    }
    fmt.Printf("%s: %s\n", addr.Hex(), balance)
}
```

## 6\.3 解析历史事件

```go
// 查询过去 1000 个区块内的 Transfer 事件
startBlock := uint64(5000000)
endBlock := uint64(5001000)

filterOpts := &bind.FilterOpts{
    Start:   startBlock,
    End:     &endBlock,
    Context: context.Background(),
}

// 过滤指定地址的转出事件
fromFilter := []common.Address{fromAddress}
toFilter := []common.Address{} // 空数组表示不过滤

it, err := token.FilterTransfer(filterOpts, fromFilter, toFilter)
if err != nil {
    log.Fatal(err)
}
defer it.Close()

for it.Next() {
    event := it.Event
    fmt.Printf("区块 %d: %s -> %s, 数量: %s\n",
        event.Raw.BlockNumber,
        event.From.Hex(),
        event.To.Hex(),
        event.Value,
    )
}
```

**最佳实践：**将常用的合约交互逻辑封装到独立的 service 层，便于复用和测试。

> （注：部分内容可能由 AI 生成）
