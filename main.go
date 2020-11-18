package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	go scheduleEPGUpdate()

	r := gin.Default()
	r.Use(gin.Logger())
	r.GET("/list.m3u", handleList)
	r.GET("/epg.xml", handleEPG)
	r.GET("/stream/:id", handleStream)
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
