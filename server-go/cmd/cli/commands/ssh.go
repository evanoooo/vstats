package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// SSH connection options
var (
	sshUser     string
	sshPort     int
	sshKey      string
	sshPassword string
)

// sshCmd represents the ssh command group
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Deploy via SSH",
	Long: `Deploy vStats agent or web dashboard to remote servers via SSH.

Uses your system SSH configuration (~/.ssh/config) for host management.
Configure your hosts there for easier access.

Examples:
  vstats ssh agent root@server.com       # Deploy agent via SSH
  vstats ssh agent myserver              # Use SSH config host alias
  vstats ssh web root@dashboard.com      # Deploy web dashboard`,
}

// sshAgentCmd deploys agent to a host via SSH
var sshAgentCmd = &cobra.Command{
	Use:   "agent <host>",
	Short: "Deploy vStats agent via SSH",
	Long: `Deploy the vStats agent to a remote server via SSH.

This command will:
  1. Connect to the server via SSH
  2. Create a new server in vStats Cloud (or use existing)
  3. Download and install the vStats agent
  4. Start the agent service

The agent will automatically report metrics to vStats Cloud.

Examples:
  vstats ssh agent root@192.168.1.1
  vstats ssh agent myserver                    # Use SSH config alias
  vstats ssh agent server.com -u admin
  vstats ssh agent server.com --name "Prod-01"
  vstats ssh agent server.com --server existing-server-id`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireLogin(); err != nil {
			return err
		}

		hostArg := args[0]
		serverName, _ := cmd.Flags().GetString("name")
		existingServerID, _ := cmd.Flags().GetString("server")

		// Parse host (user@host or just host from ssh config)
		user, host := parseSSHHost(hostArg)
		if sshUser != "" {
			user = sshUser
		}
		if user == "" {
			user = "root"
		}

		// Default server name to hostname
		if serverName == "" {
			serverName = host
		}

		client := NewClient()

		// Get or create server
		var serverID string
		var agentKey string

		if existingServerID != "" {
			server, err := findServerByNameOrID(client, existingServerID)
			if err != nil {
				return err
			}
			serverID = server.ID
			agentKey = server.AgentKey
			fmt.Printf("Using existing server: %s\n", server.Name)
		} else {
			fmt.Printf("Creating server '%s'...\n", serverName)
			server, err := client.CreateServer(serverName)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}
			serverID = server.ID
			agentKey = server.AgentKey
			fmt.Printf("✓ Server created: %s\n", server.ID)
		}

		// Build SSH command
		sshArgs := buildSSHArgs(user, host)

		// Get the cloud URL
		cloudURL := cfg.CloudURL
		if cloudURL == "" {
			cloudURL = "https://api.vstats.zsoft.cc"
		}

		// Generate install command
		installCmd := fmt.Sprintf(
			`curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --server "%s" --token "%s" --name "%s"`,
			cloudURL, cfg.Token, serverName,
		)

		fmt.Printf("\nConnecting to %s...\n", hostArg)
		fmt.Println("Deploying vStats agent...")
		fmt.Println()

		// Execute via SSH
		if err := runSSHCommand(sshArgs, installCmd); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}

		fmt.Println()
		fmt.Println("╔═══════════════════════════════════════════════════╗")
		fmt.Println("║        Agent Deployed Successfully!               ║")
		fmt.Println("╚═══════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Printf("  Server ID:  %s\n", serverID)
		fmt.Printf("  Agent Key:  %s\n", agentKey)
		fmt.Println()
		fmt.Println("  View metrics:")
		fmt.Printf("    vstats server metrics %s\n", serverName)
		fmt.Println()

		return nil
	},
}

// sshWebCmd deploys web dashboard to a host via SSH
var sshWebCmd = &cobra.Command{
	Use:   "web <host>",
	Short: "Deploy vStats web dashboard via SSH",
	Long: `Deploy a vStats web dashboard to a remote server via SSH.

The web dashboard connects to vStats Cloud to display your server metrics.

Free users: 1 web instance
Pro users: Unlimited web instances

Examples:
  vstats ssh web root@192.168.1.1
  vstats ssh web myserver --name "Home Dashboard"
  vstats ssh web server.com --port 8080
  vstats ssh web server.com --ssl --domain dashboard.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireLogin(); err != nil {
			return err
		}

		hostArg := args[0]
		webName, _ := cmd.Flags().GetString("name")
		webPort, _ := cmd.Flags().GetInt("web-port")
		domain, _ := cmd.Flags().GetString("domain")
		enableSSL, _ := cmd.Flags().GetBool("ssl")

		// Check user plan
		client := NewClient()
		plan, err := client.GetUserPlan()
		if err != nil {
			return fmt.Errorf("failed to check plan: %w", err)
		}

		if !plan.IsPro && plan.CurrentCount >= plan.MaxWebApps {
			fmt.Println("╔═══════════════════════════════════════════════════╗")
			fmt.Println("║           Web Instance Limit Reached              ║")
			fmt.Println("╚═══════════════════════════════════════════════════╝")
			fmt.Println()
			fmt.Printf("  Your plan: %s\n", plan.Plan)
			fmt.Printf("  Web instances: %d / %d\n", plan.CurrentCount, plan.MaxWebApps)
			fmt.Println()
			fmt.Println("  Upgrade to Pro for unlimited web instances:")
			fmt.Println("    https://vstats.zsoft.cc/pricing")
			fmt.Println()
			return fmt.Errorf("web instance limit reached")
		}

		// Parse host
		user, host := parseSSHHost(hostArg)
		if sshUser != "" {
			user = sshUser
		}
		if user == "" {
			user = "root"
		}

		// Defaults
		if webName == "" {
			webName = fmt.Sprintf("web-%s", host)
		}
		if webPort == 0 {
			webPort = 3001
		}

		fmt.Printf("Deploying web dashboard '%s'...\n", webName)
		fmt.Printf("  Host: %s\n", hostArg)
		fmt.Printf("  Port: %d\n", webPort)
		if domain != "" {
			fmt.Printf("  Domain: %s\n", domain)
		}
		if enableSSL {
			fmt.Println("  SSL: enabled")
		}
		fmt.Println()

		// Register web instance in cloud
		instance, err := client.RegisterWebInstance(&WebInstance{
			Name: webName,
			Host: host,
			Port: webPort,
			URL:  buildWebURL(host, webPort, domain, enableSSL),
		})
		if err != nil {
			return fmt.Errorf("failed to register web instance: %w", err)
		}

		// Build SSH command
		sshArgs := buildSSHArgs(user, host)

		// Get cloud URL
		cloudURL := cfg.CloudURL
		if cloudURL == "" {
			cloudURL = "https://api.vstats.zsoft.cc"
		}

		// Generate install command
		installCmd := fmt.Sprintf(
			`curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- --cloud-mode --cloud-url "%s" --cloud-token "%s" --port %d`,
			cloudURL, cfg.Token, webPort,
		)
		if enableSSL && domain != "" {
			installCmd += fmt.Sprintf(` --ssl --domain "%s"`, domain)
		}

		fmt.Printf("Connecting to %s...\n", hostArg)
		fmt.Println("Installing vStats web dashboard...")
		fmt.Println()

		// Execute via SSH
		if err := runSSHCommand(sshArgs, installCmd); err != nil {
			// Cleanup on failure
			_ = client.RemoveWebInstance(instance.ID)
			return fmt.Errorf("deployment failed: %w", err)
		}

		// Update status
		instance.Status = "online"
		_ = client.UpdateWebInstance(instance)

		fmt.Println()
		fmt.Println("╔═══════════════════════════════════════════════════╗")
		fmt.Println("║       Web Dashboard Deployed Successfully!        ║")
		fmt.Println("╚═══════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Printf("  Name:        %s\n", instance.Name)
		fmt.Printf("  Instance ID: %s\n", instance.ID)
		fmt.Printf("  URL:         %s\n", instance.URL)
		fmt.Println()
		fmt.Println("  The dashboard is connected to vStats Cloud")
		fmt.Println("  and will display all your monitored servers.")
		fmt.Println()

		return nil
	},
}

// parseSSHHost parses user@host format, returns (user, host)
func parseSSHHost(hostArg string) (string, string) {
	if strings.Contains(hostArg, "@") {
		parts := strings.SplitN(hostArg, "@", 2)
		return parts[0], parts[1]
	}
	return "", hostArg
}

// buildSSHArgs builds SSH command arguments
func buildSSHArgs(user, host string) []string {
	args := []string{}

	// Add port if specified
	if sshPort != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", sshPort))
	}

	// Add key if specified
	if sshKey != "" {
		args = append(args, "-i", sshKey)
	}

	// Add target
	target := host
	if user != "" {
		target = user + "@" + host
	}
	args = append(args, target)

	return args
}

// runSSHCommand executes a command via SSH using the system ssh client
func runSSHCommand(sshArgs []string, command string) error {
	// Check for ssh
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH. Please install OpenSSH")
	}

	// Build full args: ssh [args] command
	fullArgs := append(sshArgs, command)

	cmd := exec.Command(sshPath, fullArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func init() {
	// Add subcommands
	sshCmd.AddCommand(sshAgentCmd)
	sshCmd.AddCommand(sshWebCmd)

	// Agent deploy flags
	sshAgentCmd.Flags().StringVarP(&sshUser, "user", "u", "", "SSH username (default: root)")
	sshAgentCmd.Flags().IntVarP(&sshPort, "port", "p", 0, "SSH port (uses ssh config default)")
	sshAgentCmd.Flags().StringVarP(&sshKey, "key", "i", "", "SSH private key path")
	sshAgentCmd.Flags().String("name", "", "Server name in vStats")
	sshAgentCmd.Flags().String("server", "", "Use existing server ID instead of creating new")

	// Web deploy flags
	sshWebCmd.Flags().StringVarP(&sshUser, "user", "u", "", "SSH username (default: root)")
	sshWebCmd.Flags().IntVarP(&sshPort, "port", "p", 0, "SSH port (uses ssh config default)")
	sshWebCmd.Flags().StringVarP(&sshKey, "key", "i", "", "SSH private key path")
	sshWebCmd.Flags().String("name", "", "Web dashboard name")
	sshWebCmd.Flags().Int("web-port", 3001, "Web dashboard port")
	sshWebCmd.Flags().String("domain", "", "Custom domain for the dashboard")
	sshWebCmd.Flags().Bool("ssl", false, "Enable SSL (requires domain)")
}
