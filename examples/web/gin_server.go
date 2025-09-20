// Example: GAuth Integration with Gin Web Framework
// [GAuth] Only GAuth protocol logic is used here.
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

func main() {
	r := gin.Default()
	config := &gauth.Config{
		ClientID:          "WebService",
		AccessTokenExpiry: time.Hour,
	}
	gauthInstance, err := gauth.New(config, nil)
	if err != nil {
		panic(err)
	}

	// Issue a grant and token for demo
	grantReq := gauth.AuthorizationRequest{
		ClientID: "WebService",
		Scopes:   []string{"web:access"},
	}
	grant, err := gauthInstance.InitiateAuthorization(grantReq)
	if err != nil {
		panic(err)
	}
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   []string{"web:access"},
		Context: context.Background(),
	}
	tokenResp, err := gauthInstance.RequestToken(tokenReq)
	if err != nil {
		panic(err)
	}

	// Issue a power of attorney for demo
	r.GET("/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"token": tokenResp.Token,
			"valid_until": tokenResp.ValidUntil,
		})
	})


	// NOTE: You must run 'go get github.com/gin-gonic/gin' to use this example.
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
