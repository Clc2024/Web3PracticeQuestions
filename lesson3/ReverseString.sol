// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/*
2. ✅ 反转字符串 (Reverse String)
题目描述：反转一个字符串。输入 "abcde"，输出 "edcba"
*/
contract ReverseString {
    function reverse(string memory s) public pure returns (string memory) {
        bytes memory b = bytes(s);
        uint256 len = b.length;

        // 处理空字符串和单字符字符串的边界情况
        if (len <= 1) {
            return s;
        }
        for (uint256 i = 0; i < len / 2; i++) {
            (b[i], b[len - 1 - i]) = (b[len - 1 - i], b[i]);
        }

        return string(b);
    }

    function testReverse() public pure returns (string memory) {
        return reverse("abcde");
    }
}
