package main

import (
	"context"
	"fmt"

	"github.com/baixiaoustc/go_code_analysis/example/inner"
)

func main() {
	fmt.Println("start")

	Test3()
	test3a()
	test3c()

	go receiveFromKafka()
	select {}
}

func Test3() {
	fmt.Println("test3")
	test3b()
}

type XYZ struct {
	Name string
}

func (xyz XYZ) print() {
	fmt.Println(xyz.Name)
	context.WithCancel(nil)
}

func test3a() {
	xyz := XYZ{"hello"}
	xyz.print()
}

func test3b() {
	//test3b()
	inner.Itest1()
}

func test3c() {
	go func() {
		fmt.Println("go")
	}()
	test4a("world")
}
