/*
Copyright © 2024 shynome <shynome@gmail.com>
*/
package main

import (
	"github.com/shynome/go-wagi/cmd"
)

var Version = "dev"

func main() {
	cmd.Execute(Version)
}
