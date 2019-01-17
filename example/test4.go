package main

import (
	"context"
	"fmt"
)

func test4a(a string) {
	fmt.Println(a)
	context.WithCancel(nil)
}

func test4b(a string) {
	fmt.Println(a)
	context.WithCancel(nil)
}

func receiveFromKafka() {
	test4a("kafka")
	test4b("kafka")
}
