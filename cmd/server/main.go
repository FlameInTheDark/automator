package main

import (
	"context"
	"log"
	"os"

	emeraldcli "github.com/FlameInTheDark/emerald/cmd/server/cli"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:        "emerald",
		Description: "AI-powered automation platform",
		Commands: []*cli.Command{
			{
				Name:   "server",
				Usage:  "Start the Emerald web server",
				Action: emeraldcli.RunServer,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Server port (default: from config or 8080)",
					},
					&cli.StringFlag{
						Name:  "host",
						Usage: "Server host (default: from config or 0.0.0.0)",
					},
				},
			},
			{
				Name:  "pipeline",
				Usage: "Pipeline management commands",
				Commands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "List all pipelines",
						Action: emeraldcli.ListPipelines,
					},
					{
						Name:   "get",
						Usage:  "Get a pipeline by ID or name",
						Action: emeraldcli.GetPipeline,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "id",
								Aliases:  []string{"i"},
								Usage:    "Pipeline ID or name",
								Required: true,
							},
						},
					},
					{
						Name:   "executions",
						Usage:  "List executions for a pipeline",
						Action: emeraldcli.ListPipelineExecutions,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "id",
								Aliases:  []string{"i"},
								Usage:    "Pipeline ID",
								Required: true,
							},
							&cli.IntFlag{
								Name:    "limit",
								Aliases: []string{"l"},
								Usage:   "Number of executions to show (default: 50)",
								Value:   5,
							},
						},
					},
					{
						Name:   "execution",
						Usage:  "Get execution details with node logs",
						Action: emeraldcli.GetExecution,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "id",
								Aliases:  []string{"i"},
								Usage:    "Execution ID",
								Required: true,
							},
						},
					},
					{
						Name:   "run",
						Usage:  "Run a pipeline manually",
						Action: emeraldcli.RunPipeline,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "id",
								Aliases:  []string{"i"},
								Usage:    "Pipeline ID (supports partial ID)",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "input",
								Aliases: []string{"in"},
								Usage:   "Input data as JSON",
							},
						},
					},
				},
			},
			{
				Name:  "config",
				Usage: "Inspect and update stored configuration",
				Commands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "List configured resources",
						Action: emeraldcli.ListConfigs,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "resource",
								Usage: "Optional resource type: proxmox_cluster, kubernetes_cluster, channel, llm_provider",
							},
							&cli.BoolFlag{
								Name:  "json",
								Usage: "Render results as JSON",
							},
						},
					},
					{
						Name:   "get",
						Usage:  "Show one configured resource by ID or exact name",
						Action: emeraldcli.GetConfig,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "resource",
								Usage:    "Resource type: proxmox_cluster, kubernetes_cluster, channel, llm_provider",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "id",
								Usage: "Exact resource ID",
							},
							&cli.StringFlag{
								Name:  "name",
								Usage: "Exact resource name (case-insensitive)",
							},
							&cli.BoolFlag{
								Name:  "show-secrets",
								Usage: "Include decrypted secret values when available",
							},
						},
					},
					{
						Name:   "update",
						Usage:  "Update one configured resource with a JSON patch object",
						Action: emeraldcli.UpdateConfig,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "resource",
								Usage:    "Resource type: proxmox_cluster, kubernetes_cluster, channel, llm_provider",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "id",
								Usage: "Exact resource ID",
							},
							&cli.StringFlag{
								Name:  "name",
								Usage: "Exact resource name (case-insensitive)",
							},
							&cli.StringFlag{
								Name:  "patch",
								Usage: "Inline JSON object describing the fields to update",
							},
							&cli.StringFlag{
								Name:  "patch-file",
								Usage: "Path to a JSON file containing the patch object",
							},
							&cli.BoolFlag{
								Name:  "show-secrets",
								Usage: "Include decrypted secret values in the response",
							},
						},
					},
				},
			},
			{
				Name:  "db",
				Usage: "Database management commands",
				Commands: []*cli.Command{
					{
						Name:   "migrate",
						Usage:  "Run database migrations",
						Action: emeraldcli.RunMigrations,
					},
					{
						Name:   "version",
						Usage:  "Show database version",
						Action: emeraldcli.DBVersion,
					},
				},
			},
			{
				Name:  "debug",
				Usage: "Debug tools",
				Commands: []*cli.Command{
					{
						Name:   "sql",
						Usage:  "Interactive SQL CLI",
						Action: emeraldcli.DebugSQL,
					},
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return emeraldcli.RunServer(context.Background(), cmd)
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
