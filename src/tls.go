package main

import (
	"log"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
)

func setupTls(r *gin.Engine) {
	log.Fatal(autotls.Run(r, "example1.com"))
}