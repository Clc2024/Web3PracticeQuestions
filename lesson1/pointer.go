package main

import "fmt"

//incrementByTen 接收整数指针参数，将指针指向的值加10
//形参是*intl类型（整数指针），指向主函数中变量的内存地址
func incrementByTen(numPtr *int) {
	//解引用指针：*numPtr 表示访问指针指向的内存地址中的值
	*numPtr += 10 //等价于 *numPtr=*numPtr+10

}

//doubleSlice 接收整数切片的指针，将每个元素乘以2
//形参是*[]int类型（切片指针），指向主函数中的切片变量的内存地址
func doubleSlice(slicePtr *[]int) {
	//1.先校验指针是否为nil，避免运行时painc（工程化最佳实践）
	if slicePtr == nil {
		fmt.Println("错误：传入了nil指针")
		return
	}
	//2.解引用切片指针，获取底层的切片
	//*slicePtr 表示访问指针指向的切片本身
	slice := *slicePtr
	//3.遍历切片，修改每个元素的值
	for i := range slice {
		slice[i] *= 2 //等价于slice[i]=slice[i]*2
	}
	//也可以直接嵌套解引用，省略中间变量slice
	//for i:=range*slicePtr{
	//(*slicePtr)[i]*=2
	//}

}
func main() {
	/*指针
	  1.题目 ：编写一个Go程序，定义一个函数，该函数接收一个整数指针作为参数，在函数内部将该指针指向的值增加10，然后在主函数中调用该函数并输出修改后的值。
	  考察点 ：指针的使用、值传递与引用传递的区别。
	  2.题目 ：实现一个函数，接收一个整数切片的指针，将切片中的每个元素乘以2。
	  考察点 ：指针运算、切片操作。
	*/

	//1.定义一个普通整数变量
	num := 5
	fmt.Println("修改前的值：", num)
	//2.调用函数，传入num的地址（&num获取变量的内存地址，类型为*int）
	incrementByTen(&num)
	//3.输出修改后的值（num的值已经被函数修改）
	fmt.Println("修改后的值：", num)

	//1.定义一个整数切片
	nums := []int{1, 2, 3, 4, 5}
	fmt.Println("修改前的切片：", nums)
	//2.调用函数，传入切片的地址
	doubleSlice(&nums)
	//3.输出修改后的切片
	fmt.Println("修改后的切片：", nums)
}
