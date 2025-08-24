package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sirupsen/logrus"

	"github.com/lcanady/depin/services/gpu-discovery/internal"
	"github.com/lcanady/depin/hardware/detection/common"
)

var (
	cfgFile string
	logger  = logrus.New()
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gpu-discovery",
	Short: "GPU Discovery Service for DePIN AI Compute Network",
	Long: `GPU Discovery Service automatically detects and monitors GPU resources
across the network, providing comprehensive hardware information and capabilities
for the DePIN AI compute infrastructure.

Supported GPU vendors:
- NVIDIA (via NVML)
- AMD (via ROCm tools)
- Intel (via GPU tools)`,
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the GPU discovery gRPC server",
	Long: `Start the GPU discovery service as a gRPC server.
This will initialize all available GPU detectors and begin serving
discovery requests.`,
	Run: runServe,
}

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover GPUs and print results",
	Long: `Perform a one-time GPU discovery and print the results.
This is useful for testing and debugging the detection capabilities.`,
	Run: runDiscover,
}

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark [gpu-id]",
	Short: "Run benchmarks on a specific GPU",
	Long: `Run performance benchmarks on a specific GPU.
The GPU ID should be obtained from the discover command.`,
	Args: cobra.ExactArgs(1),
	Run:  runBenchmark,
}

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor GPU changes in real-time",
	Long: `Monitor GPU changes and status updates in real-time.
This will continuously watch for hardware changes and performance updates.`,
	Run: runMonitor,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gpu-discovery.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "text", "log format (text, json)")
	
	// Bind flags to viper
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))

	// Serve command flags
	serveCmd.Flags().Int("port", 50051, "gRPC server port")
	serveCmd.Flags().Int("metrics-port", 9090, "Metrics server port")
	serveCmd.Flags().Bool("enable-nvidia", true, "Enable NVIDIA GPU detection")
	serveCmd.Flags().Bool("enable-amd", true, "Enable AMD GPU detection")
	serveCmd.Flags().Bool("enable-intel", true, "Enable Intel GPU detection")
	serveCmd.Flags().Bool("auto-discovery", true, "Enable automatic periodic discovery")
	serveCmd.Flags().Duration("discovery-interval", 60*time.Second, "Auto-discovery interval")
	
	// Discover command flags
	discoverCmd.Flags().Bool("force-refresh", false, "Force refresh of GPU cache")
	discoverCmd.Flags().StringSlice("vendor", []string{}, "Filter by vendor (nvidia, amd, intel)")
	discoverCmd.Flags().Bool("include-benchmarks", false, "Include benchmark results")
	discoverCmd.Flags().String("output", "table", "Output format (table, json, yaml)")

	// Benchmark command flags
	benchmarkCmd.Flags().StringSlice("types", []string{"compute", "memory"}, "Benchmark types to run")
	benchmarkCmd.Flags().Duration("duration", 30*time.Second, "Benchmark duration")
	benchmarkCmd.Flags().String("output", "table", "Output format (table, json)")

	// Monitor command flags
	monitorCmd.Flags().Duration("interval", 30*time.Second, "Monitoring interval")
	monitorCmd.Flags().Bool("performance-metrics", false, "Include performance metrics updates")

	// Add commands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(discoverCmd)
	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(monitorCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/gpu-discovery")
		viper.SetConfigName(".gpu-discovery")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("GPU_DISCOVERY")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		logger.Info("Using config file:", viper.ConfigFileUsed())
	}

	// Configure logger
	setupLogger()
}

func setupLogger() {
	level, err := logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if viper.GetString("log-format") == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}

func runServe(cmd *cobra.Command, args []string) {
	logger.Info("Starting GPU Discovery Service")

	// Create service configuration
	config := createServiceConfig(cmd)

	// Create and initialize service
	service, err := internal.NewGPUDiscoveryService(config)
	if err != nil {
		logger.Fatalf("Failed to create GPU discovery service: %v", err)
	}

	// Create and start gRPC server
	server := internal.NewGPUDiscoveryServer(service)
	port, _ := cmd.Flags().GetInt("port")
	
	if err := server.Start(port); err != nil {
		logger.Fatalf("Failed to start gRPC server: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Infof("GPU Discovery Service started on port %d", port)
	logger.Info("Press Ctrl+C to shutdown")

	// Wait for signal
	<-sigChan
	logger.Info("Shutdown signal received")

	// Graceful shutdown
	server.Stop()
	if err := service.Shutdown(); err != nil {
		logger.Errorf("Error during service shutdown: %v", err)
	}

	logger.Info("Service shutdown completed")
}

func runDiscover(cmd *cobra.Command, args []string) {
	logger.Info("Running GPU discovery")

	config := createServiceConfig(cmd)
	config.EnableAutoDiscovery = false // Disable for one-time discovery

	service, err := internal.NewGPUDiscoveryService(config)
	if err != nil {
		logger.Fatalf("Failed to create GPU discovery service: %v", err)
	}
	defer service.Shutdown()

	forceRefresh, _ := cmd.Flags().GetBool("force-refresh")
	vendorFilter, _ := cmd.Flags().GetStringSlice("vendor")
	includeBenchmarks, _ := cmd.Flags().GetBool("include-benchmarks")
	outputFormat, _ := cmd.Flags().GetString("output")

	gpus, err := service.DiscoverGPUs(forceRefresh, vendorFilter)
	if err != nil {
		logger.Fatalf("GPU discovery failed: %v", err)
	}

	if len(gpus) == 0 {
		fmt.Println("No GPUs found")
		return
	}

	// Get additional info if requested
	if includeBenchmarks {
		logger.Info("Running benchmarks on discovered GPUs...")
		// This would extend the GPU info with benchmarks
	}

	// Output results
	switch outputFormat {
	case "json":
		outputJSON(gpus)
	case "yaml":
		outputYAML(gpus)
	default:
		outputTable(gpus)
	}
}

func runBenchmark(cmd *cobra.Command, args []string) {
	gpuID := args[0]
	logger.Infof("Running benchmarks on GPU: %s", gpuID)

	config := createServiceConfig(cmd)
	config.EnableAutoDiscovery = false

	service, err := internal.NewGPUDiscoveryService(config)
	if err != nil {
		logger.Fatalf("Failed to create GPU discovery service: %v", err)
	}
	defer service.Shutdown()

	benchmarkTypes, _ := cmd.Flags().GetStringSlice("types")
	duration, _ := cmd.Flags().GetDuration("duration")
	outputFormat, _ := cmd.Flags().GetString("output")

	results, err := service.RunBenchmark(gpuID, benchmarkTypes, duration)
	if err != nil {
		logger.Fatalf("Benchmark failed: %v", err)
	}

	// Output results
	switch outputFormat {
	case "json":
		outputBenchmarkJSON(results)
	default:
		outputBenchmarkTable(results)
	}
}

func runMonitor(cmd *cobra.Command, args []string) {
	logger.Info("Starting GPU monitoring")

	config := createServiceConfig(cmd)
	
	service, err := internal.NewGPUDiscoveryService(config)
	if err != nil {
		logger.Fatalf("Failed to create GPU discovery service: %v", err)
	}
	defer service.Shutdown()

	// Setup monitoring callback
	err = service.StartMonitoring(func(change common.GPUChange) {
		fmt.Printf("[%s] %s: %s - %s\n",
			change.Timestamp.Format("15:04:05"),
			change.Type,
			change.GPU.Name,
			change.Description)
	})
	if err != nil {
		logger.Fatalf("Failed to start monitoring: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("GPU monitoring started. Press Ctrl+C to stop.")
	<-sigChan
	fmt.Println("\nStopping monitor...")
}

func createServiceConfig(cmd *cobra.Command) *internal.ServiceConfig {
	config := internal.DefaultServiceConfig()

	// Update from command flags
	if cmd.Flags().Changed("port") {
		port, _ := cmd.Flags().GetInt("port")
		config.GRPCPort = port
	}

	if cmd.Flags().Changed("enable-nvidia") {
		enabled, _ := cmd.Flags().GetBool("enable-nvidia")
		config.DetectorConfig.EnableNVIDIA = enabled
	}

	if cmd.Flags().Changed("enable-amd") {
		enabled, _ := cmd.Flags().GetBool("enable-amd")
		config.DetectorConfig.EnableAMD = enabled
	}

	if cmd.Flags().Changed("enable-intel") {
		enabled, _ := cmd.Flags().GetBool("enable-intel")
		config.DetectorConfig.EnableIntel = enabled
	}

	if cmd.Flags().Changed("auto-discovery") {
		enabled, _ := cmd.Flags().GetBool("auto-discovery")
		config.EnableAutoDiscovery = enabled
	}

	if cmd.Flags().Changed("discovery-interval") {
		interval, _ := cmd.Flags().GetDuration("discovery-interval")
		config.AutoDiscoveryInterval = interval
	}

	// Update from viper config
	config.LogLevel = viper.GetString("log-level")
	config.LogFormat = viper.GetString("log-format")

	return config
}

// Output formatting functions
func outputTable(gpus []common.GPUInfo) {
	fmt.Printf("%-4s %-20s %-10s %-12s %-10s %-8s %-10s\n",
		"ID", "Name", "Vendor", "Memory(MB)", "State", "Util%", "Temp(C)")
	fmt.Println(strings.Repeat("-", 80))
	
	for i, gpu := range gpus {
		fmt.Printf("%-4d %-20s %-10s %-12d %-10s %-8d %-10d\n",
			i,
			truncate(gpu.Name, 20),
			gpu.Vendor,
			gpu.Specs.MemoryTotalMB,
			gpu.Status.State.String(),
			gpu.Status.GPUUtilization,
			gpu.Status.TemperatureGPU)
	}
}

func outputJSON(gpus []common.GPUInfo) {
	// JSON marshaling would go here
	fmt.Println("JSON output not implemented yet")
}

func outputYAML(gpus []common.GPUInfo) {
	// YAML marshaling would go here
	fmt.Println("YAML output not implemented yet")
}

func outputBenchmarkTable(results []common.BenchmarkResult) {
	fmt.Printf("%-15s %-20s %-10s %-10s %-10s\n",
		"Type", "Test", "Score", "Unit", "Duration")
	fmt.Println(strings.Repeat("-", 70))
	
	for _, result := range results {
		fmt.Printf("%-15s %-20s %-10.2f %-10s %-10ds\n",
			result.BenchmarkType,
			truncate(result.TestName, 20),
			result.Score,
			result.Unit,
			result.DurationSeconds)
	}
}

func outputBenchmarkJSON(results []common.BenchmarkResult) {
	// JSON marshaling would go here
	fmt.Println("JSON output not implemented yet")
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}