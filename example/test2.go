package main

import (
	"context"
	"fmt"
)

func test2(a string, b int) {
	context.WithCancel(nil) //000

	if _, err := context.WithCancel(nil); err != nil { //111
		context.WithCancel(nil) //222
	} else {
		context.WithCancel(nil) //333
	}

	_, _ = context.WithCancel(nil) //444

	go context.WithCancel(nil) //555

	go func() {
		context.WithCancel(nil) //666
	}()

	defer context.WithCancel(nil) //777

	defer func() {
		context.WithCancel(nil) //888
	}()

	data := map[string]interface{}{
		"x2": context.WithValue(nil, "k", "v"), //999
	}
	fmt.Println(data)

	/*
		for i := context.WithCancel(nil); i; i = false {//aaa
			context.WithCancel(nil)//bbb
		}
	*/

	var keys []string = []string{"ccc"}
	for _, k := range keys {
		fmt.Println(k)
		context.WithCancel(nil)
	}
}
