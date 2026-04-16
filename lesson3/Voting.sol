// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/*
1.创建一个名为Voting的合约，包含以下功能：
一个mapping来存储候选人的得票数
一个vote函数，允许用户投票给某个候选人
一个getVotes函数，返回某个候选人的得票数
一个resetVotes函数，重置所有候选人的得票数
*/
contract Voting {
    //1.核心数据结构：mapping存储候选人得票数
    mapping(address => uint256) private   _voteCounts;
    
    address[] public candidates;
    //2.权限控制
    address public immutable owner;

    //3.防止重复投票
    mapping(address => bool) private _hasVoted;

    constructor() {
        owner=msg.sender;
    }

    //4.vote函数 用户为指定候选人投票
    function vote(address candidate) external {
        require(candidate!=address(0), "Voting invalid candidate");
        require(candidate!=msg.sender, "Voting cannot vote for yourself");
        require(!_hasVoted[msg.sender], "Voting already voted");

        _hasVoted[msg.sender]=true;
        _voteCounts[candidate]++;
        //将候选人加入列表 
        if (_voteCounts[candidate]==1) {
            candidates.push(candidate);
        }
    }

    //5.getVotes函数 重置所有候选人的得票数
    function getVotes(address candidate)external view returns (uint256) {
        return _voteCounts[candidate];
    }

    //6.resetVotes函数 重置所有候选人的得票数
    function resetVotes() external {
        require(msg.sender==owner, "Voting only owner can reset votes");

        //遍历所有候选人，清空票数
         for (uint i=0; i<candidates.length; i++) 
         {
            delete _voteCounts[candidates[i]];
            //delete _hasVoted[candidates[i]];
         }
         //清空候选人列表
         delete candidates;
         //创建一个新的空的
    }

    function hasVoted(address voter) public view returns (bool)  {
        return _hasVoted[voter];
    }
}
