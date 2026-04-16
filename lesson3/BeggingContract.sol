// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title 讨饭合约
 * @dev 允许用户向合约捐赠ETH，记录捐赠信息，只有合约所有者可以提取资金
 */
contract BeggingContract {
    //合约所有者
    address public owner;

    //记录每个地址的捐赠总额
    mapping(address => uint256) public donations;

    address[] public allDonors;

    //捐赠事件
    event Donation(address indexed donor, uint256 amount, uint256 timestamp);

    //提款事件
    event Withdrawal(
        address indexed recipient,
        uint256 amount,
        uint256 timestamp
    );

    //所有者权限修饰符
    modifier onlyOwner() {
        require(msg.sender == owner, "Only the owner can call this function.");
        _;
    }

    //构造函数
    constructor() {
        owner = msg.sender;
    }

    /**
     * @dev 接收ETH捐赠
     * 任何人都可以调用此函数向合约发送ETH
     * 会自动记录捐赠金额
     */
    function donate() external payable {
        require(msg.value > 0, "Donation amount must be greater than 0");

        if (donations[msg.sender]==0) {
            allDonors.push(msg.sender);
        }
        
        //记录捐赠金额
        donations[msg.sender] += msg.value;
        //触发捐赠事件
        emit Donation(msg.sender, msg.value, block.timestamp);
    }

    /**
     * @dev 查询指定地址的捐赠总额
     * @param donor 捐赠者地址
     * @return 该地址的捐赠总额
     */
    function getDonation(address donor) external view returns (uint256) {
        return donations[donor];
    }

    /**
     * @dev 提取合约中的所有ETH
     * 只有合约所有者可以调用
     */
    function withdrawal() external onlyOwner {
        uint256 balance = address(this).balance;
        require(balance > 0, "No funds to withdraw");
        //提款
        (bool success, ) = msg.sender.call{value: balance}("");
        require(success, "Withdrawal failed");
        //触发提款事件
        emit Withdrawal(msg.sender, balance, block.timestamp);
    }

    /**
     * @dev 获取合约当前余额
     */
    function getBalance() external view returns (uint256) {
        return address(this).balance;
    }

    /**
     * @dev 接收ETH的备用函数
     * 如果用户直接向合约地址转账，也会被记录
     */
    receive() external payable {
        require(msg.value > 0, "Donation amount must be greater than 0");
        donations[msg.sender] += msg.value;
        emit Donation(msg.sender, msg.value, block.timestamp);
    }

    //可选功能 捐赠排行榜
    struct DonorInfo {
        address donor;
        uint256 amount;
    }

    /**获取捐赠前三的用户
     */
    function getTop3Donors() external view returns (DonorInfo[3] memory) {
        DonorInfo[3] memory top3;

        // 遍历所有捐赠者
        for (uint i = 0; i < allDonors.length; i++) {
            address addr = allDonors[i];
            uint256 amount = donations[addr];

            // 插入排序进 top3
            for (uint j = 0; j < 3; j++) {
                if (amount > top3[j].amount) {
                    // 后移
                    for (uint k = 2; k > j; k--) {
                        top3[k] = top3[k - 1];
                    }
                    top3[j] = DonorInfo(addr, amount);
                    break;
                }
            }
        }
        return top3;
    }
}
