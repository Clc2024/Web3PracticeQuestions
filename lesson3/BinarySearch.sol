// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BinarySearch {
    /**
     * @dev 二分查找：在升序有序数组中查找目标值
     * @param arr 升序有序数组
     * @param target 要查找的目标数字
     * @return 找到返回下标索引，未找到返回 uint256.max
     */
    function binarySearch(
        uint[] memory arr,
        uint target
    ) public pure returns (uint) {
        uint left = 0;
        uint right = arr.length - 1;

        while (left <= right) {
            // 安全计算中间值，避免溢出
            uint mid = left + (right - left) / 2;

            if (arr[mid] == target) {
                // 找到目标，返回索引
                return mid;
            } else if (arr[mid] < target) {
                // 目标在右侧
                left = mid + 1;
            } else {
                // 目标在左侧
                right = mid - 1;
            }
        }

        // 未找到，返回一个特殊值（uint最大值）
        return type(uint).max;
    }

    // 测试用例
    function test() public pure returns (uint, uint) {
        uint[] memory arr = new uint[](5);
        arr[0] = 1;
        arr[1] = 3;
        arr[2] = 5;
        arr[3] = 7;
        arr[4] = 9;

        uint foundIndex = binarySearch(arr, 5); // 找到 → 返回 2
        uint notFound = binarySearch(arr, 10); // 未找到 → 返回 uint最大值

        return (foundIndex, notFound);
    }
}
