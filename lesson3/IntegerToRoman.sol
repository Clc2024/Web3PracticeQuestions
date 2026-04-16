// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
// 4. solidity 实现罗马数字转数整数
contract IntToRoman {
    // 按从大到小顺序，列出所有罗马数字的「值-符号」对（包含减法形式）
    uint256 private constant NUM_VALUES = 13;
    uint256[NUM_VALUES] private values = [
        1000,
        900,
        500,
        400,
        100,
        90,
        50,
        40,
        10,
        9,
        5,
        4,
        1
    ];
    string[NUM_VALUES] private symbols = [
        "M",
        "CM",
        "D",
        "CD",
        "C",
        "XC",
        "L",
        "XL",
        "X",
        "IX",
        "V",
        "IV",
        "I"
    ];

    /**
     * @dev 整数转罗马数字
     * @param num 输入整数（范围 1 ~ 3999，符合罗马数字标准范围）
     * @return 转换后的罗马数字字符串
     */
    function intToRoman(uint256 num) public view returns (string memory) {
        // 输入合法性校验：罗马数字标准范围 1-3999
        require(num > 0 && num <= 3999, "Input out of range (1-3999)");

        // 用 bytes 动态拼接字符串（Solidity 高效拼接方案）
        bytes memory resultBytes;

        // 遍历所有值-符号对
        for (uint256 i = 0; i < NUM_VALUES; i++) {
            uint256 val = values[i];
            string memory sym = symbols[i];

            // 尽可能多地减去当前值，拼接对应符号
            while (num >= val) {
                resultBytes = abi.encodePacked(resultBytes, sym);
                num -= val;
            }

            // 提前终止：num 为 0 时无需继续遍历
            if (num == 0) break;
        }

        return string(resultBytes);
    }

    /**
     * @dev 测试用例：覆盖所有典型场景
     * @return 测试结果元组
     */
    function testCases()
        public
        view
        returns (
            string memory, // 3 → "III"
            string memory, // 4 → "IV"
            string memory, // 9 → "IX"
            string memory, // 58 → "LVIII"
            string memory, // 1994 → "MCMXCIV"
            string memory // 12 → "XII"
        )
    {
        return (
            intToRoman(3),
            intToRoman(4),
            intToRoman(9),
            intToRoman(58),
            intToRoman(1994),
            intToRoman(12)
        );
    }
}
