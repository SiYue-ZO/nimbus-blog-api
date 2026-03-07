package main

// @title Nimbus Blog API
// @version 1.0
// @description Nimbus Blog 后端 API（Admin + Public V1）。
// @BasePath /api
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey AdminSession
// @in cookie
// @name fiber_session

import (
	"log"

	"github.com/scc749/nimbus-blog-api/config"
	_ "github.com/scc749/nimbus-blog-api/docs"
	"github.com/scc749/nimbus-blog-api/internal/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}
	app.Run(cfg)
}
