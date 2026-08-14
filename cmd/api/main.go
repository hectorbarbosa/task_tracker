package main

import (
	"log"

	"task_tracker/internal/config"
	"task_tracker/internal/server"
)

//	@title			Task Tracker API
//	@version		1.0
//	@description	REST API service for team task management with role-based access control,
//	@description	change history, Redis caching, and SQL analytics.
//
//	@host		localhost:8080
//	@BasePath	/api/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Bearer {token}

func main() {
	cfg := config.Load()

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
