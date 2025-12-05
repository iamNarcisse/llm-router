package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	config "llm-router/services/router/internal/config"
	embedding "llm-router/services/router/internal/embedding"
	router "llm-router/services/router/internal/router"
	server "llm-router/services/router/internal/server"
	pb "llm-router/services/router/pkg/pb"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize embedding client
	embedClient, err := embedding.NewClient(cfg.Embedding.Address, cfg.Embedding.Timeout)
	if err != nil {
		log.Fatalf("Failed to create embedding client: %v", err)
	}
	defer embedClient.Close()

	// Initialize router
	rtr := router.New(embedClient, &cfg.Qdrant, &cfg.Routing)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	routerServer := server.NewRouterServer(rtr)
	pb.RegisterRouterServiceServer(grpcServer, routerServer)
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start HTTP health server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: server.NewHealthHandler(rtr),
	}

	go func() {
		log.Printf("HTTP health server listening on :%d", cfg.Server.HTTPPort)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	grpcServer.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	log.Println("Server stopped")
}
