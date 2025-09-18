package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

func main() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated key: %v, addr: %p\n", key, key)

	tokenCfg := &token.Config{
		SigningKey:   key,
		SigningMethod: token.RS256,
		ValidityPeriod: 0,
	}
	fmt.Printf("tokenCfg.SigningKey: %v, addr: %p\n", tokenCfg.SigningKey, tokenCfg.SigningKey)

	fmt.Printf("token.NewService type: %T\n", token.NewService)
	tokenSvc := token.NewService(tokenCfg, token.NewMemoryStore())
	fmt.Println(tokenSvc)
}
