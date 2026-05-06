## 5.搞清楚以下几个问题：
- a：ERC20/ERC721合约中的transferFrom接口如何使用
- b：ERC721中的授权接口跟ERC20有何不同
- c：ERC721合约中的safeTransfer等带safe前缀的接口提供了什么安全措施


问题解答：

ERC20 同质化代币，ERC721非同质化代币

a.
1）ERC20/ERC721合约中的 transferFrom 接口的使用流程：需要调用者先授权给合约使用approve，再通过合约调用 transferFrom 转走代币，不同的是 ERC721 非同质化代币授权接口有两个approve和setApprovalForAll；
2）合约调用 发送人的地址、接收人的地址， 不同点在于 ERC20 同质化代币 转的是代币数量 amount ，ERC721非同质化代币转的是tokenId唯一代币。

b.
approve(address spender, uint256 amount)|approve(address to, uint256 tokenId)、setApprovalForAll(address operator, bool _approved)
1)ERC20 授权 只有一种 approve ，授权固定金额，使用授权金额会减少，用完需要重新授权，只能针对同一个代币
2)ERC721 授权 有两种 approve 只授权某一个tokenId,转完就失效； setApprovalForAll 授权给合约下所有NFT

c.
ERC721合约中的safeTransfer等带safe前缀的接口的目的，防止NFT发送到无法处理NFT的合约，导致永久丢失。它做了2件关键安全检查
1)如果接受方是合约地址，会检查该合约是否实现了 onERC721Received;
2)如果接收方合约没有实现这个方法，转账会直接失败，revert! 保护资产；
3)普通的 Transfer  不会检查接收方，一旦出现错误会直接丢失。

