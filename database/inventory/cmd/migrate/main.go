package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"../../config"
	"../../migrations"
	"../../utils"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		command    = flag.String("command", "up", "Migration command: up, down, status")
		timeout    = flag.Duration("timeout", 5*time.Minute, "Command timeout")
	)
	flag.Parse()

	// Load configuration
	cfg := config.DefaultConfig()
	if *configPath != "" {
		// In a real implementation, you would load from file
		fmt.Printf("Using default configuration (config file loading not implemented)\n")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create database manager
	dbManager, err := utils.NewDatabaseManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create database manager: %v", err)
	}
	defer dbManager.Close()

	// Create migrator
	migrator := migrations.NewMigrator(dbManager, &cfg.Database)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Execute command
	switch *command {
	case "up":
		fmt.Println("Running database migrations...")
		if err := migrator.Up(ctx); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back latest migration...")
		if err := migrator.Down(ctx); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")

	case "status":
		fmt.Println("Checking migration status...")
		if err := migrator.Status(ctx); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: up, down, status")
		os.Exit(1)
	}
}