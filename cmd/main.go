package main

import (
	"log"
	"net/http"

	"merchShop/internal/config"
	"merchShop/internal/handler"
	"merchShop/internal/handler/mw"
	"merchShop/internal/repository"
	"merchShop/internal/server"
	"merchShop/internal/usecase"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	repo, err := repository.NewPostgresRepo(cfg.DSN())
	if err != nil {
		log.Fatalf("failed to init repository: %v", err)
	}

	mw.SetSecretKey([]byte(cfg.JWTSecret))

	svc := usecase.NewService(repo)
	h := handler.NewHandler(svc)
	r := server.NewRouter(h)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	server.StartHTTPServer(srv)
}
