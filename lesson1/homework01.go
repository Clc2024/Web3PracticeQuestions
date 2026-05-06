package main

import (
	"fmt"
	"sort"
	"strings"
)

// 1. 只出现一次的数字
// 给定一个非空整数数组，除了某个元素只出现一次以外，其余每个元素均出现两次。找出那个只出现了一次的元素。
func SingleNumber(nums []int) int {
	// var m map[int]int
	// m = make(map[int]int)
	m := make(map[int]int)
	for _, v := range nums {
		value := m[v]
		// m[v]=value+1
		if value != 0 {
			m[v] = m[v] + 1
		} else {
			m[v] = 1
		}
	}

	var result int
	for k, v := range m {
		if v == 1 {
			result = k
		}
	}

	return result
}

// 2. 回文数
// 判断一个整数是否是回文数
func IsPalindrome(x int) bool {
	// TODO: implement
	//完整反转判断回文数
	//特殊情况：负数、末尾为0且非0的数字直接返回false
	if x < 0 || (x%10 == 0 && x != 0) {
		return false
	}
	original := x //保存原数
	reversed := 0 //存储反转后的数

	//逐个位置反转
	for x > 0 {
		digit := x % 10
		reversed = reversed*10 + digit
		x = x / 10
	}
	//比较反转数和原数
	return reversed == original
}

// 3. 有效的括号
// 给定一个只包括 '(', ')', '{', '}', '[', ']' 的字符串，判断字符串是否有效
func IsValid(s string) bool {
	//验证括号字符串是否有效
	//定义右括号到左括号的映射表，快速匹配
	bracketMap := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}
	//用切片模拟栈，存储左括号
	stack := []rune{}
	//遍历字符串中的每个字符
	for _, char := range s {
		//1.判断是否是右括号（映射表中存在该键）
		if leftBrackt, ok := bracketMap[char]; ok {
			//是右括号：检查栈是否为空 或 栈顶左括号不匹配
			if len(stack) == 0 || stack[len(stack)-1] != leftBrackt {
				return false
			}
			//匹配成功，弹出栈顶元素
			stack = stack[:len(stack)-1]
		} else {
			//2. 是左括号，入栈
			stack = append(stack, char)
		}
	}

	//遍历结束后，栈必须为空（所有左括号都闭合）
	return len(stack) == 0
}

// 4. 最长公共前缀
// 查找字符串数组中的最长公共前缀
func LongestCommonPrefix(strs []string) string {
	//横向扫描法找最长公共前缀
	//边界条件：空数组直接返回空
	if len(strs) == 0 {
		return ""
	}
	//初始前缀为第一个字符串
	prefix := strs[0]
	//遍历剩余字符串
	for i := 1; i < len(strs); i++ {
		//循环缩短前缀，直到当前字符串以该前缀开头
		for !strings.HasPrefix(strs[i], prefix) {
			//前缀缩短一位（去掉最后一个字符串）
			prefix = prefix[:len(prefix)-1]
			//前缀为空，直接返回
			if prefix == "" {
				return ""
			}

		}
	}

	return prefix
}

// 5. 加一
// 给定一个由整数组成的非空数组所表示的非负整数，在该数的基础上加一
func PlusOne(digits []int) []int {
	//大整数数组加1
	carry := 1
	//从最后一位（最低位）逆序遍历
	for i := len(digits) - 1; i >= 0; i-- {
		//计算当前位总和（原数字+进位）
		sum := digits[i] + carry
		//当前为最终值 = 总和 %10
		digits[i] = sum % 10
		//新的进位 = 总和 /10
		carry = sum / 10
		//无进位时提出退出，减少不必要的遍历
		if carry == 0 {
			break
		}
	}

	//若最终仍有进位（如999+1=1000），在数组头部插入1
	if carry == 1 {
		//新建数组，首元素为1，后面拼接原数组(原数组位置已经均为0)
		digits = append([]int{1}, digits...)
	}

	return digits
}

// 6. 删除有序数组中的重复项
// 给你一个有序数组 nums ，请你原地删除重复出现的元素，使每个元素只出现一次，返回删除后数组的新长度。
// 不要使用额外的数组空间，你必须在原地修改输入数组并在使用 O(1) 额外空间的条件下完成。
func RemoveDuplicates(nums []int) int {
	//边界条件：空数组直接返回0
	if len(nums) == 0 {
		return 0
	}

	//慢指针：标记唯一元素的末尾位置
	slow := 0
	//快指针：遍历整个数组
	for fast := 1; fast < len(nums); fast++ {
		//遇到与慢指针位置不同的元素（新的唯一元素）
		if nums[fast] != nums[slow] {
			slow++                  //慢指针后移，准备存储新元素
			nums[slow] = nums[fast] //更新慢指针位置的值
		}

		//相等则跳过，快指针继续前进 fast++
	}

	//唯一元素数量=慢指针索引+1
	return slow + 1
}

// 7. 合并区间
// 以数组 intervals 表示若干个区间的集合，其中单个区间为 intervals[i] = [starti, endi] 。
// 请你合并所有重叠的区间，并返回一个不重叠的区间数组，该数组需恰好覆盖输入中的所有区间。
func Merge(intervals [][]int) [][]int {
	//合并重叠区间
	//边界条件：空数组或只有一个区间，直接返回
	if len(intervals) <= 1 {
		return intervals
	}
	//步骤1：按照区间起始值升序排序
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })

	//步骤2：初始化结果集，加入第一个区间
	result := [][]int{intervals[0]}

	//步骤3：遍历剩余区间，逐个合并(从第二个区间开始)
	for i := 1; i < len(intervals); i++ {
		//取结果集最后一个区间
		last := result[len(result)-1]
		//当前遍历的区间
		current := intervals[i]

		//情况1：重叠/相邻，合并区间
		if current[0] <= last[1] {
			//合并后的结束值取两者的最大值(end比较大小)
			last[1] = max(last[1], current[1])
			//直接修改结果集最后一个区间（原地更新）
			//(步骤一 升序排序，保证了 last取到的是开头最小的start)
			result[len(result)-1] = last
		} else {
			//情况2：不重叠，加入结果集
			result = append(result, current)
		}

	}

	return result
}

// 8. 两数之和
// 给定一个整数数组 nums 和一个目标值 target，请你在该数组中找出和为目标值的那两个整数
func TwoSum(nums []int, target int) []int {
	//暴力枚举法找两数之和的下标
	//外层循环：遍历每个元素

	for i := 0; i < len(nums); i++ {
		//内层循环：遍历i之后的元素（避免重复使用同一元素）
		for j := i + 1; j < len(nums); j++ {
			if nums[i]+nums[j] == target {
				return []int{i, j}
			}
		}
	}

	return nil
}

func main() {

	// 1. 只出现一次的数字
	// 给定一个非空整数数组，除了某个元素只出现一次以外，其余每个元素均出现两次。找出那个只出现了一次的元素。
	numbers := []int{1, 2, 3, 4, 5, 6, 5, 4, 3, 2, 1}
	sint := SingleNumber(numbers)
	fmt.Printf("1.只出现一次的数字：%d \n", sint)

	// 2. 回文数
	// 判断一个整数是否是回文数
	//回文数：正向和反向读取结果完全相同的整数（如 121、12321 是回文数；-121、123 不是，负数因负号存在直接排除）
	testCases2 := []int{121, -121, 120, 12321, 0, 1234}
	for _, v := range testCases2 {
		fmt.Printf("2.数字 %d 是否是回文数 ：%t\n", v, IsPalindrome(v))
	}

	// 3. 有效的括号
	// 给定一个只包括 '(', ')', '{', '}', '[', ']' 的字符串，判断字符串是否有效
	testCases3 := []struct {
		s        string
		expected bool
	}{
		{"{}", true},
		{"{}()[]", true},
		{"{", false},
		{"}{}", false},
		{"{[(])}", false}, //左括号必须以正确的顺序闭合
	}
	//执行测试并输出结果
	for _, tc := range testCases3 {
		result := IsValid(tc.s)
		fmt.Printf("3. 有效的括号 字符串：%-8s 预期：%t  实际：%t\n", tc.s, tc.expected, result)
	}

	// 4. 最长公共前缀
	// 查找字符串数组中首字符的最长公共前缀
	testCases4 := [][]string{
		{"flower", "flow", "flight"},
		{"dog", "racecar", "car"},
		{"apple", "apple", "apple3"},
		{},
		{"a"},
		{"ab", "a"},
		{"b", "a", "bc"},
	}
	for _, v := range testCases4 {
		result := LongestCommonPrefix(v)
		fmt.Printf("4. 最长公共前缀: %-20v 最长公共前缀 ：%q\n", v, result)
	}

	// 5. 加一
	// 给定一个由整数组成的非空数组所表示的非负整数，在该数的基础上加一
	testCases5 := [][]int{
		{1, 2, 3},
		{4, 3, 2, 1},
		{9},
		{9, 9, 9},
		{0},
		{1, 9, 9},
	}

	//执行测试并输出结果
	for _, v := range testCases5 {
		//拷贝原数组（避免函数内修改影响原数组）
		input := make([]int, len(v))
		copy(input, v)
		result := PlusOne(input)
		fmt.Printf("5. 加一,原数组：%-8v 加1后 ： %v\n", v, result)
	}

	// 6. 删除有序数组中的重复项
	// 给你一个有序数组 nums ，请你原地删除重复出现的元素，使每个元素只出现一次，返回删除后数组的新长度。
	// 不要使用额外的数组空间，你必须在原地修改输入数组并在使用 O(1) 额外空间的条件下完成。
	testCases6 := [][]int{
		{1, 1, 2},
		{0, 0, 1, 1, 1, 2, 2, 3, 3, 4},
		{},
		{1},
		{2, 2, 2, 2},
	}
	//执行测试并输出结果

	for _, v := range testCases6 {
		//拷贝原数组（避免函数内修改影响原数组展示）
		input := make([]int, len(v))
		copy(input, v)
		k := RemoveDuplicates(input)
		//输出原数组、新长度、去重后的前K个元素
		fmt.Printf("\n 6.删除有序数组中的重复项 原数组：%v \n 新长度：%d \n  去重后前k个元素：%v\n", v, k, input[:k])
	}

	// 7. 合并区间
	// 以数组 intervals 表示若干个区间的集合，其中单个区间为 intervals[i] = [starti, endi] 。
	// 请你合并所有重叠的区间，并返回一个不重叠的区间数组，该数组需恰好覆盖输入中的所有区间。

	testCases7 := [][][]int{
		{{1, 3}, {2, 6}, {8, 10}, {15, 18}}, // 基础重叠：合并为[[1,6],[8,10],[15,18]]
		{{1, 4}, {4, 5}},                    // 相邻区间：合并为[[1,5]]
		{{1, 4}, {2, 3}},                    // 完全包含：合并为[[1,4]]
		{},                                  // 空数组：返回[]
		{{1, 2}},                            // 单区间：返回[[1,2]]
		{{2, 3}, {1, 4}},                    // 无序区间：排序后合并为[[1,4]]
	}
	for _, tc := range testCases7 {
		result := Merge(tc)
		fmt.Printf("7. 合并区间,原区间数组： %v 合并后： %v\n", tc, result)
	}

	// 8. 两数之和
	// 给定一个整数数组 nums 和一个目标值 target，请你在该数组中找出和为目标值的那两个整数
	testCases8 := []struct {
		nums   []int
		target int
	}{
		{[]int{2, 7, 11, 15}, 9},
		{[]int{3, 2, 4}, 6},
		{[]int{3, 3}, 6},
	}

	for _, tc := range testCases8 {
		result := TwoSum(tc.nums, tc.target)
		fmt.Printf("8. 整数数组 两数之和,数组：%v 目标值：%d  结果下标:%v \n", tc.nums, tc.target, result)
	}
}
