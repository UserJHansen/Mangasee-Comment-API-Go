package main

import "github.com/gin-gonic/gin"


func userResponse(c *gin.Context) {
	c.JSON(200, &Result[[]Username]{Status: "OK", Result: userMap})
}