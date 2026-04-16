// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SortedArrayMerge {
    // 合并两个升序数组
    function mergeSortedArrays(
        uint[] memory arr1,
        uint[] memory arr2
    ) public pure returns (uint[] memory) {
        uint[] memory merged = new uint[](arr1.length + arr2.length);
        uint i = 0; // 数组1指针
        uint j = 0; // 数组2指针
        uint k = 0; // 合并数组指针

        while (i < arr1.length && j < arr2.length) {
            if (arr1[i] <= arr2[j]) {
                merged[k] = arr1[i];
                i++;
            } else {
                merged[k] = arr2[j];
                j++;
            }
            k++;
        }

        // 添加剩余元素
        while (i < arr1.length) {
            merged[k] = arr1[i];
            i++;
            k++;
        }

        while (j < arr2.length) {
            merged[k] = arr2[j];
            j++;
            k++;
        }

        return merged;
    }

    // 测试用例
    function test() public pure returns (uint[] memory) {
        uint[] memory a = new uint[](3);
        a[0] = 1;
        a[1] = 3;
        a[2] = 4;
        uint[] memory b = new uint[](3);
        b[0] = 2;
        b[1] = 3;
        b[2] = 5;
        b[2] = 6;
        return mergeSortedArrays(a, b);
        // 输出 [1,2,3,4,5,6]
    }
}
