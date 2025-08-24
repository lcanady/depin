package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"../../config"
	"../../scripts"
	"../../utils"
)

func main() {
	var (
		configPath  = flag.String("config", "config.yaml", "Path to configuration file")
		command     = flag.String("command", "create", "Backup command: create, restore, list, cleanup, verify, schedule")
		backupType  = flag.String("type", "full", "Backup type: full, incremental")
		backupPath  = flag.String("path", "", "Path to backup file (for restore/verify)")
		targetDB    = flag.String("target", "", "Target database name (for restore)")
		timeout     = flag.Duration("timeout", 30*time.Minute, "Command timeout")
	)
	flag.Parse()

	// Load configuration
	cfg := config.DefaultConfig()
	if *configPath != "" {
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

	// Create backup manager
	backupManager := scripts.NewBackupManager(cfg, dbManager)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Execute command
	switch *command {
	case "create":
		fmt.Printf("Creating %s backup...\n", *backupType)
		
		var result *scripts.BackupResult
		var err error
		
		if *backupType == "incremental" {
			result, err = backupManager.CreateIncrementalBackup(ctx, "")
		} else {
			result, err = backupManager.CreateBackup(ctx, *backupType)
		}
		
		if err != nil {
			log.Fatalf("Backup creation failed: %v", err)
		}
		
		fmt.Printf("Backup created successfully:\n")
		fmt.Printf("  Path: %s\n", result.BackupPath)
		fmt.Printf("  Size: %d bytes\n", result.Size)
		fmt.Printf("  Duration: %v\n", result.Duration)
		fmt.Printf("  Tables: %d\n", len(result.Tables))

	case "restore":
		if *backupPath == "" {
			log.Fatal("Backup path is required for restore command")
		}
		if *targetDB == "" {
			log.Fatal("Target database is required for restore command")
		}
		
		fmt.Printf("Restoring from backup: %s to database: %s\n", *backupPath, *targetDB)
		if err := backupManager.RestoreBackup(ctx, *backupPath, *targetDB); err != nil {
			log.Fatalf("Restore failed: %v", err)
		}
		fmt.Println("Restore completed successfully")

	case "list":
		fmt.Println("Listing available backups...")
		backups, err := backupManager.ListBackups()
		if err != nil {
			log.Fatalf("Failed to list backups: %v", err)
		}
		
		if len(backups) == 0 {
			fmt.Println("No backups found")
		} else {
			fmt.Printf("Found %d backups:\n", len(backups))
			fmt.Println("Name\t\t\t\t\t\tSize\t\tCreated")
			fmt.Println("----\t\t\t\t\t\t----\t\t-------")
			for _, backup := range backups {
				fmt.Printf("%s\t%d\t%s\n", 
					backup.Name, 
					backup.Size, 
					backup.CreatedAt.Format("2006-01-02 15:04:05"))
			}
		}

	case "cleanup":
		fmt.Println("Cleaning up old backups...")
		if err := backupManager.CleanupOldBackups(ctx); err != nil {
			log.Fatalf("Cleanup failed: %v", err)
		}

	case "verify":
		if *backupPath == "" {
			log.Fatal("Backup path is required for verify command")
		}
		
		fmt.Printf("Verifying backup: %s\n", *backupPath)
		if err := backupManager.VerifyBackup(ctx, *backupPath); err != nil {
			log.Fatalf("Verification failed: %v", err)
		}
		fmt.Println("Backup verification successful")

	case "schedule":
		fmt.Println("Starting backup scheduler...")
		fmt.Println("Press Ctrl+C to stop")
		backupManager.ScheduleBackups(ctx)

	case "dr-test":
		fmt.Println("Running disaster recovery test...")
		dr := scripts.NewDisasterRecovery(backupManager, &cfg.Database)
		if err := dr.TestDisasterRecovery(ctx); err != nil {
			log.Fatalf("Disaster recovery test failed: %v", err)
		}
		fmt.Println("Disaster recovery test completed successfully")

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: create, restore, list, cleanup, verify, schedule, dr-test")
		os.Exit(1)
	}
}