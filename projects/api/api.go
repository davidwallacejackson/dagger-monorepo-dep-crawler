package main

import (
	"net/http"

	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/lib"
	"github.com/gin-gonic/gin"
)

func greet(c *gin.Context, name string) {
	greeting := lib.Greet(name)

	c.JSON(http.StatusOK, gin.H{
		"greeting": greeting,
	})
}

func main() {
	r := gin.Default()

	r.GET("/greet", func(c *gin.Context) {
		greet(c, "stranger")
	})
	r.GET("/greet/:name", func(c *gin.Context) {
		name := c.Param("name")
		greet(c, name)
	})
	r.Run()
}
