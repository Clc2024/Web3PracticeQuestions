package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rectangle struct {
	width  float64
	height float64
}

func (r Rectangle) Area() float64 {
	return r.width * r.height
}
func (r Rectangle) Perimeter() float64 {
	return 2 * (r.width + r.height)
}

type Circle struct {
	radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.radius * c.radius
}
func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.radius
}

func PrintInfo(s Shape) {
	fmt.Println("面积：", s.Area())
	fmt.Println("周长：", s.Perimeter())
}

type Person struct {
	Name string
	Age  int
}

func (p Person) PrintInfo() {
	fmt.Println("Name:", p.Name, "Age:", p.Age)
}

type Employee struct {
	Person
	EmployeeID int
}

func (e Employee) PrintInfo() {
	fmt.Println("Name:", e.Name, "Age:", e.Age, "EmployeeID:", e.EmployeeID)
}

func main() {
	/*
	   1.题目 ：定义一个 Shape 接口，包含 Area() 和 Perimeter() 两个方法。然后创建 Rectangle 和 Circle 结构体，实现 Shape 接口。在主函数中，创建这两个结构体的实例，并调用它们的 Area() 和 Perimeter() 方法。
	   考察点 ：接口的定义与实现、面向对象编程风格。
	   2.题目 ：使用组合的方式创建一个 Person 结构体，包含 Name 和 Age 字段，再创建一个 Employee 结构体，组合 Person 结构体并添加 EmployeeID 字段。为 Employee 结构体实现一个 PrintInfo() 方法，输出员工的信息。
	   考察点 ：组合的使用、方法接收者。
	*/
	fmt.Println("1.题目 ：定义一个 Shape 接口，包含 Area() 和 Perimeter() 两个方法。然后创建 Rectangle 和 Circle 结构体，实现 Shape 接口。在主函数中，创建这两个结构体的实例，并调用它们的 Area() 和 Perimeter() 方法。")
	r := Rectangle{width: 5, height: 3}
	PrintInfo(r)
	c := Circle{radius: 4}
	PrintInfo(c)

	fmt.Println("2.题目 ：使用组合的方式创建一个 Person 结构体，包含 Name 和 Age 字段，再创建一个 Employee 结构体，组合 Person 结构体并添加 EmployeeID 字段。为 Employee 结构体实现一个 PrintInfo() 方法，输出员工信息。")
	p := Person{Name: "张三", Age: 18}
	p.PrintInfo()
	e := Employee{Person: p, EmployeeID: 1001}
	e.PrintInfo()
}
