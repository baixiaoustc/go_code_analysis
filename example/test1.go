package main

import "fmt"
import "strings"

func test1() {
	hello := "Hello"
	world := "World"
	words := []string{hello, world}
	SayHello(words)
}

// SayHello says Hello
func SayHello(words []string) bool {
	fmt.Println(joinStrings(words))
	return true
}

// joinStrings joins strings
func joinStrings(words []string) string {
	return strings.Join(words, ", ")
}
