package main

import "github.com/gin-gonic/gin"

func discussionResponse(c *gin.Context) {
	c.JSON(200, &Result[[]Discussion]{Status: "OK", Result: discussions})
}