package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// WebInstance represents a deployed web dashboard instance
type WebInstance struct {
	ID          string     `json:"id" yaml:"id"`
	Name        string     `json:"name" yaml:"name"`
	Host        string     `json:"host" yaml:"host"`
	Port        int        `json:"port" yaml:"port"`
	URL         string     `json:"url" yaml:"url"`
	Status      string     `json:"status" yaml:"status"`
	Version     string     `json:"version" yaml:"version"`
	CloudMode   bool       `json:"cloud_mode" yaml:"cloud_mode"`
	SSLEnabled  bool       `json:"ssl_enabled" yaml:"ssl_enabled"`
	CreatedAt   time.Time  `json:"created_at" yaml:"created_at"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty" yaml:"last_check_at,omitempty"`
}

// UserPlan represents user subscription plan
type UserPlan struct {
	Plan         string `json:"plan" yaml:"plan"`
	MaxWebApps   int    `json:"max_web_apps" yaml:"max_web_apps"`
	CurrentCount int    `json:"current_count" yaml:"current_count"`
	IsPro        bool   `json:"is_pro" yaml:"is_pro"`
}

// webCmd represents the web command group
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Manage web dashboard instances",
	Long: `Manage vStats web dashboard deployments.

Web dashboards connect to vStats Cloud to display your server metrics.
Deploy via SSH using: vstats ssh web <host>

Free users: 1 web instance
Pro users: Unlimited web instances

Examples:
  vstats web list              # List all web instances
  vstats web status            # Show plan & web limits
  vstats web check <id>        # Check instance health
  vstats web remove <id>       # Remove a web instance
  vstats ssh web root@server   # Deploy web via SSH`,
}

// webListCmd lists all web instances
var webListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all web dashboard instances",
	Long:    `List all web dashboard instances associated with your account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireLogin(); err != nil {
			return err
		}

		client := NewClient()
		instances, err := client.ListWebInstances()
		if err != nil {
			return fmt.Errorf("failed to list web instances: %w", err)
		}

		switch outputFmt {
		case "json":
			return OutputJSON(instances)
		case "yaml":
			return OutputYAML(instances)
		default:
			if len(instances) == 0 {
				fmt.Println("No web instances found.")
				fmt.Println("Use 'vstats ssh web <host>' to deploy a web dashboard.")
				return nil
			}

			table := NewTable("NAME", "HOST", "PORT", "STATUS", "URL", "CREATED")
			for _, w := range instances {
				table.AddRow(
					w.Name,
					w.Host,
					fmt.Sprintf("%d", w.Port),
					formatWebStatus(w.Status),
					w.URL,
					formatTimeAgo(&w.CreatedAt),
				)
			}
			table.Render()
		}
		return nil
	},
}

// webRemoveCmd removes a web instance
var webRemoveCmd = &cobra.Command{
	Use:     "remove <id>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a web dashboard instance",
	Long: `Remove a web dashboard instance from your account.

This only removes the registration from vStats Cloud.
To uninstall from the server, SSH in and run:
  curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- --uninstall`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireLogin(); err != nil {
			return err
		}

		instanceID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		client := NewClient()

		// Find instance
		instance, err := client.GetWebInstance(instanceID)
		if err != nil {
			return fmt.Errorf("web instance not found: %s", instanceID)
		}

		// Confirm removal
		if !force {
			fmt.Printf("Are you sure you want to remove web instance '%s'? [y/N] ", instance.Name)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Remove from cloud
		if err := client.RemoveWebInstance(instance.ID); err != nil {
			return fmt.Errorf("failed to remove instance: %w", err)
		}

		fmt.Printf("✓ Web instance '%s' removed\n", instance.Name)
		fmt.Println()
		fmt.Println("To uninstall from the server, SSH in and run:")
		fmt.Println("  curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- --uninstall")
		return nil
	},
}

// webStatusCmd shows plan and web status
var webStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show web deployment status and limits",
	Long:  `Show your subscription plan and web instance limits.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireLogin(); err != nil {
			return err
		}

		client := NewClient()

		plan, err := client.GetUserPlan()
		if err != nil {
			return fmt.Errorf("failed to get plan: %w", err)
		}

		instances, err := client.ListWebInstances()
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		switch outputFmt {
		case "json":
			return OutputJSON(map[string]interface{}{
				"plan":      plan,
				"instances": instances,
			})
		case "yaml":
			return OutputYAML(map[string]interface{}{
				"plan":      plan,
				"instances": instances,
			})
		default:
			fmt.Println("Web Dashboard Status")
			fmt.Println("====================")
			fmt.Println()
			fmt.Printf("Plan:            %s\n", plan.Plan)
			fmt.Printf("Web Instances:   %d / %s\n", plan.CurrentCount, formatLimit(plan.MaxWebApps))
			fmt.Println()

			if len(instances) > 0 {
				fmt.Println("Deployed Instances:")
				for _, w := range instances {
					fmt.Printf("  • %s (%s) - %s\n", w.Name, w.URL, formatWebStatus(w.Status))
				}
			} else {
				fmt.Println("No web instances deployed yet.")
				fmt.Println("Deploy with: vstats ssh web <host>")
			}

			if !plan.IsPro {
				fmt.Println()
				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				fmt.Println("Upgrade to Pro for unlimited web instances!")
				fmt.Println("  https://vstats.zsoft.cc/pricing")
				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			}
		}
		return nil
	},
}

// webCheckCmd checks the status of a web instance
var webCheckCmd = &cobra.Command{
	Use:   "check <id>",
	Short: "Check web instance health",
	Long:  `Check the health and connectivity of a web instance.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireLogin(); err != nil {
			return err
		}

		instanceID := args[0]
		client := NewClient()

		instance, err := client.GetWebInstance(instanceID)
		if err != nil {
			return fmt.Errorf("web instance not found: %s", instanceID)
		}

		fmt.Printf("Checking web instance '%s'...\n", instance.Name)
		fmt.Println()

		// Check HTTP connectivity
		status, err := client.CheckWebInstance(instance.ID)
		if err != nil {
			fmt.Printf("✗ Health check failed: %v\n", err)
			return nil
		}

		switch outputFmt {
		case "json":
			return OutputJSON(status)
		case "yaml":
			return OutputYAML(status)
		default:
			fmt.Printf("Status:       %s\n", formatWebStatus(status.Status))
			fmt.Printf("URL:          %s\n", instance.URL)
			fmt.Printf("Response:     %s\n", status.ResponseTime)
			fmt.Printf("Version:      %s\n", status.Version)
			fmt.Printf("Cloud Sync:   %s\n", formatBool(status.CloudConnected))
			fmt.Printf("Last Check:   %s\n", formatTime(status.CheckedAt))
		}

		return nil
	},
}

// Helper function to format web status
func formatWebStatus(status string) string {
	switch status {
	case "online":
		if noColor {
			return "● online"
		}
		return "\033[32m● online\033[0m"
	case "offline":
		if noColor {
			return "○ offline"
		}
		return "\033[31m○ offline\033[0m"
	case "pending":
		if noColor {
			return "◐ pending"
		}
		return "\033[33m◐ pending\033[0m"
	default:
		return status
	}
}

// Helper function to format limit
func formatLimit(limit int) string {
	if limit < 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", limit)
}

// Helper function to format bool
func formatBool(b bool) string {
	if b {
		return "✓ connected"
	}
	return "✗ disconnected"
}

// Helper function to build web URL
func buildWebURL(host string, port int, domain string, ssl bool) string {
	scheme := "http"
	if ssl {
		scheme = "https"
	}

	if domain != "" {
		if ssl && port == 443 {
			return fmt.Sprintf("%s://%s", scheme, domain)
		} else if !ssl && port == 80 {
			return fmt.Sprintf("%s://%s", scheme, domain)
		}
		return fmt.Sprintf("%s://%s:%d", scheme, domain, port)
	}

	if port == 80 || port == 443 {
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// Client methods for web instance management
func (c *Client) ListWebInstances() ([]WebInstance, error) {
	var instances []WebInstance
	err := c.get("/web/instances", &instances)
	return instances, err
}

func (c *Client) GetWebInstance(id string) (*WebInstance, error) {
	var instance WebInstance
	err := c.get("/web/instances/"+id, &instance)
	return &instance, err
}

func (c *Client) RegisterWebInstance(instance *WebInstance) (*WebInstance, error) {
	var result WebInstance
	err := c.post("/web/instances", instance, &result)
	return &result, err
}

func (c *Client) UpdateWebInstance(instance *WebInstance) error {
	return c.put("/web/instances/"+instance.ID, instance, nil)
}

func (c *Client) RemoveWebInstance(id string) error {
	return c.delete("/web/instances/" + id)
}

func (c *Client) CheckWebInstance(id string) (*WebInstanceStatus, error) {
	var status WebInstanceStatus
	err := c.get("/web/instances/"+id+"/check", &status)
	return &status, err
}

func (c *Client) GetUserPlan() (*UserPlan, error) {
	var plan UserPlan
	err := c.get("/user/plan", &plan)
	return &plan, err
}

// WebInstanceStatus represents health check result
type WebInstanceStatus struct {
	Status         string     `json:"status"`
	ResponseTime   string     `json:"response_time"`
	Version        string     `json:"version"`
	CloudConnected bool       `json:"cloud_connected"`
	CheckedAt      *time.Time `json:"checked_at"`
}

func init() {
	// Add subcommands
	webCmd.AddCommand(webListCmd)
	webCmd.AddCommand(webRemoveCmd)
	webCmd.AddCommand(webStatusCmd)
	webCmd.AddCommand(webCheckCmd)

	// Remove flags
	webRemoveCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
}
