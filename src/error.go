package main

import (
	"time"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"
)

func errorHandler(c *gin.Context, info ratelimit.Info) {
	c.JSON(429, &Result[string]{
		Status: "RATE-LIMITED", 
		Result: "Too many requests. Try again in " + time.Until(info.ResetTime).String(),
	})
}

func fourofour(c *gin.Context) {
	c.JSON(404, &Result[string]{
		Status: "FILE_NOT_FOUND",
		Result: "Oops, we can't seem to find "+c.FullPath(),
	})
}
