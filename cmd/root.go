package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/takah/loopback-manager/internal/config"
	"github.com/takah/loopback-manager/internal/manager"
)

var (
	cfgFile string
	mgr     *manager.Manager
	version = "dev" // This will be overridden by ldflags during build
)

var rootCmd = &cobra.Command{
	Use:   "loopback-manager",
	Short: "Manage loopback IP addresses for Docker Compose projects",
	Long: `A CLI tool to manage loopback IP addresses for GitHub organization repositories
with Docker Compose configurations. Automatically assigns unique IP addresses
and prevents conflicts.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		mgr = manager.New(cfg)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all IP assignments",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.List(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var assignCmd = &cobra.Command{
	Use:   "assign <org/repo>",
	Short: "Assign IP to repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Error: Invalid format. Use: org/repo\n")
			os.Exit(1)
		}
		ip, _ := cmd.Flags().GetString("ip")
		if err := mgr.Assign(parts[0], parts[1], ip); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <org/repo>",
	Short: "Remove IP assignment",
	Aliases: []string{"rm", "del"},
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Error: Invalid format. Use: org/repo\n")
			os.Exit(1)
		}
		if err := mgr.Remove(parts[0], parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for unassigned repositories",
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.Scan(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var autoAssignCmd = &cobra.Command{
	Use:   "auto-assign",
	Short: "Auto-assign IPs to all unassigned repositories (dry-run by default)",
	Aliases: []string{"auto"},
	Run: func(cmd *cobra.Command, args []string) {
		execute, _ := cmd.Flags().GetBool("execute")
		if err := mgr.AutoAssign(execute); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for duplicate IPs",
	Aliases: []string{"validate"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.CheckDuplicates(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("loopback-manager version %s\n", getVersion())
	},
}

func getVersion() string {
	// First, check if version was set via ldflags (e.g., from Makefile)
	if version != "dev" {
		return version
	}
	
	// If not, try to get version from build info (for go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	
	return "dev"
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/loopback-manager/config.yaml)")
	
	assignCmd.Flags().StringP("ip", "i", "", "Specific IP address to assign")
	autoAssignCmd.Flags().BoolP("execute", "e", false, "Execute the assignments (without this flag, only shows what would be done)")
	
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(assignCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(autoAssignCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.config/loopback-manager")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
