package main

import (
	"net/http"

	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/lib"
	"github.com/gin-gonic/gin"
	"github.com/mandrigin/gin-spa/spa"
)

func greet(c *gin.Context, name string) {
	greeting := lib.Greet(name)

	c.JSON(http.StatusOK, gin.H{
		"greeting": greeting,
	})
}

func main() {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Cache-Control", "no-cache")
		c.Next()
	})

	r.GET("/api/greet", func(c *gin.Context) {
		greet(c, "stranger")
	})
	r.GET("/api/greet/:name", func(c *gin.Context) {
		name := c.Param("name")
		greet(c, name)
	})

	r.Use(spa.Middleware("/", "./static"))

	r.Run()
}
