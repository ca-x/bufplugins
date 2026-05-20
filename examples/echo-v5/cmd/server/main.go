package main

import (
	"log"

	"github.com/ca-x/bufplugins/examples/echo-v5/internal/server"
	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()
	if err := server.RegisterRoutes(e, server.Config{}); err != nil {
		log.Fatal(err)
	}
	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
