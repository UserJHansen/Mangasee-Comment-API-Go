package main

import (
	"log"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
)

func setupTls(r *gin.Engine) {
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Email:      "hansenjames09@gmail.com",
		HostPolicy: autocert.HostWhitelist("smsmailing.sytes.net"),
		Cache:      autocert.DirCache("/opt/prometheus/comment-data/autotls"),
	}

	log.Fatal(autotls.RunWithManager(r, &m))
}
