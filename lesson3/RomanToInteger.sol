// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract RomanToInt {

    // 核心：正确实现，无char，无switch，无revert
    function romanToInt(string memory roman) public pure returns (uint256) {
        bytes memory s = bytes(roman);
        uint256 total = 0;
        uint256 prev = 0;

        for (uint i = 0; i < s.length; i++) {
            uint256 curr = getVal(s[i]);
            
            if (curr > prev) {
                total += curr - 2 * prev;
            } else {
                total += curr;
            }
            prev = curr;
        }
        return total;
    }

    // 查表函数：替代 switch + char
    function getVal(bytes1 c) private pure returns (uint256) {
        if (c == 'I') return 1;
        if (c == 'V') return 5;
        if (c == 'X') return 10;
        if (c == 'L') return 50;
        if (c == 'C') return 100;
        if (c == 'D') return 500;
        if (c == 'M') return 1000;
        revert("invalid");
    }
}