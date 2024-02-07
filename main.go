package main

import (
	"fmt"
	"monitor/cmd"
)

func main() {
	_cmd := cmd.NewCmd()
	_cmd.Run()
	fmt.Println("Exiting the program...")
}
