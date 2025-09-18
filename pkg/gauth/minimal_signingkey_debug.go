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
		SigningKey:    key,
		SigningMethod: token.RS256,
		ValidityPeriod: 0,
	}
	fmt.Printf("tokenCfg.SigningKey: %v, addr: %p\n", tokenCfg.SigningKey, tokenCfg.SigningKey)

	// Print the type and package path of token.NewService for debugging
	fmt.Printf("token.NewService type: %T\n", token.NewService)

	// Print the resolved import path for the token package
	// Run this in your terminal: go list -f '{{.ImportPath}}' github.com/Gimel-Foundation/gauth/pkg/token

	tokenSvc := token.NewService(tokenCfg, token.NewMemoryStore())
	fmt.Println(tokenSvc)
}
