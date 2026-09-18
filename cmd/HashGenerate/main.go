package main

import (
	"flag"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	plain := flag.String("p", "", "需要生成哈希的明文密码")
	flag.Parse()

	hash, err := bcrypt.GenerateFromPassword([]byte(*plain), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("哈希码生成失败: %v\n", err)
		return
	}

	fmt.Println(string(hash))
}
