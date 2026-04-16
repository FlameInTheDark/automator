package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/config"
	"github.com/FlameInTheDark/emerald/internal/db"
	"github.com/FlameInTheDark/emerald/internal/db/models"
	"github.com/FlameInTheDark/emerald/internal/db/query"
	pl "github.com/FlameInTheDark/emerald/internal/pipeline"
	sq "github.com/Masterminds/squirrel"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v3"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Question)

func ListPipelines(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	pipelineStore := query.NewPipelineStore(database.DB)
	pipelines, err := pipelineStore.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	if len(pipelines) == 0 {
		fmt.Println("No pipelines found")
		return nil
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row{"ID", "Name", "Status", "Created"})
	for _, p := range pipelines {
		t.AppendRow(table.Row{p.ID, p.Name, p.Status, p.CreatedAt.Format("2006-01-02 15:04")})
	}
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDefault)
	t.Render()

	return nil
}

func GetPipeline(ctx context.Context, cmd *cli.Command) error {
	id := cmd.String("id")
	if id == "" {
		return fmt.Errorf("pipeline id is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	pipelineStore := query.NewPipelineStore(database.DB)
	pipeline, err := pipelineStore.GetByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	fmt.Printf("ID: %s\n", pipeline.ID)
	fmt.Printf("Name: %s\n", pipeline.Name)
	fmt.Printf("Status: %s\n", pipeline.Status)
	fmt.Printf("Created: %s\n", pipeline.CreatedAt)
	fmt.Printf("Updated: %s\n", pipeline.UpdatedAt)
	if pipeline.Description != nil {
		fmt.Printf("Description: %s\n", *pipeline.Description)
	}
	fmt.Printf("\nNodes: %s\n", pipeline.Nodes)
	fmt.Printf("Edges: %s\n", pipeline.Edges)
	return nil
}

func ListPipelineExecutions(ctx context.Context, cmd *cli.Command) error {
	id := cmd.String("id")
	limit := cmd.Int("limit")
	if limit <= 0 {
		limit = 5
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	executions, err := listExecutions(database.DB, id, limit)
	if err != nil {
		return fmt.Errorf("failed to list executions: %w", err)
	}

	if len(executions) == 0 {
		fmt.Println("No executions found for this pipeline")
		return nil
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row{"ID", "Trigger", "Status", "Started"})
	for _, e := range executions {
		t.AppendRow(table.Row{e.ID, e.TriggerType, e.Status, e.StartedAt.Format("2006-01-02 15:04")})
	}
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDefault)
	t.Render()

	return nil
}

func listExecutions(db *sql.DB, pipelineID string, limit int) ([]models.Execution, error) {
	query, args, err := psql.Select("id", "pipeline_id", "trigger_type", "status", "started_at", "completed_at", "error").
		From("executions").
		Where(sq.Eq{"pipeline_id": pipelineID}).
		OrderBy("started_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var executions []models.Execution
	for rows.Next() {
		var e models.Execution
		if err := rows.Scan(&e.ID, &e.PipelineID, &e.TriggerType, &e.Status, &e.StartedAt, &e.CompletedAt, &e.Error); err != nil {
			return nil, fmt.Errorf("scan execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, rows.Err()
}

func GetExecution(ctx context.Context, cmd *cli.Command) error {
	id := cmd.String("id")
	if id == "" {
		return fmt.Errorf("execution id is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	executionStore := query.NewExecutionStore(database.DB)
	execution, err := executionStore.GetByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	fmt.Printf("Execution ID: %s\n", execution.ID)
	fmt.Printf("Pipeline ID:  %s\n", execution.PipelineID)
	fmt.Printf("Trigger:      %s\n", execution.TriggerType)
	fmt.Printf("Status:       %s\n", execution.Status)
	fmt.Printf("Started:      %s\n", execution.StartedAt.Format("2006-01-02 15:04:05"))
	if execution.CompletedAt != nil {
		fmt.Printf("Completed:    %s\n", execution.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if execution.Error != nil {
		fmt.Printf("Error:        %s\n", *execution.Error)
	}
	if execution.Context != nil {
		fmt.Printf("Context:      %s\n", *execution.Context)
	}

	nodeExecutions, err := executionStore.ListByExecution(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to get node executions: %w", err)
	}

	if len(nodeExecutions) > 0 {
		fmt.Println("\n--- Node Executions ---")
		t := table.NewWriter()
		t.AppendHeader(table.Row{"Node ID", "Type", "Status", "Duration"})
		for _, ne := range nodeExecutions {
			duration := ""
			if ne.StartedAt != nil && ne.CompletedAt != nil {
				duration = ne.CompletedAt.Sub(*ne.StartedAt).String()
			}
			t.AppendRow(table.Row{ne.NodeID, ne.NodeType, ne.Status, duration})
		}
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleDefault)
		t.Render()

		fmt.Println("\n--- Node Output/Error Logs ---")
		for _, ne := range nodeExecutions {
			fmt.Printf("\n>>> Node: %s (%s) - %s\n", ne.NodeID, ne.NodeType, ne.Status)
			if ne.Input != nil {
				fmt.Printf("Input:\n%s\n", *ne.Input)
			}
			if ne.Output != nil {
				fmt.Printf("Output:\n%s\n", *ne.Output)
			}
			if ne.Error != nil {
				fmt.Printf("Error: %s\n", *ne.Error)
			}
		}
	}

	return nil
}

func RunPipeline(ctx context.Context, cmd *cli.Command) error {
	return runPipelineCommand(ctx, cmd.String("id"), cmd.String("input"), os.Stdout, newCLIRuntime, newCLIProgressWriter)
}

type cliRuntimeFactory func(ctx context.Context, opts cliRuntimeOptions) (*runtimeBundle, error)
type cliProgressWriterFactory func(output io.Writer) cliProgressWriter

func runPipelineCommand(
	ctx context.Context,
	id string,
	inputJSON string,
	stdout io.Writer,
	runtimeFactory cliRuntimeFactory,
	progressFactory cliProgressWriterFactory,
) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("pipeline id is required")
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if runtimeFactory == nil {
		runtimeFactory = newCLIRuntime
	}
	if progressFactory == nil {
		progressFactory = newCLIProgressWriter
	}

	runtime, err := runtimeFactory(ctx, cliRuntimeOptions{migrate: true})
	if err != nil {
		return err
	}
	defer func() {
		_ = runtime.Close()
	}()

	pipelineModel, err := loadPipelineForRun(ctx, runtime.PipelineStore, id)
	if err != nil {
		return err
	}

	var input map[string]any
	if strings.TrimSpace(inputJSON) != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return fmt.Errorf("failed to parse input JSON: %w", err)
		}
	}

	flowData, err := pl.ParseFlowData(pipelineModel.Nodes, pipelineModel.Edges)
	if err != nil {
		return fmt.Errorf("failed to parse pipeline flow: %w", err)
	}

	if err := runtime.startExecutionServices(ctx); err != nil {
		return err
	}

	progressAdapter := newPipelineProgressAdapter(pipelineModel.Name, *flowData, progressFactory(stdout))
	progressAdapter.Start()
	runtime.ExecutionRunner.SetProgressCallback(progressAdapter.HandleEvent)

	result, runErr := runtime.ExecutionRunner.Run(
		ctx,
		pipelineModel.ID,
		*flowData,
		pl.TriggerSelection{TriggerType: "manual"},
		input,
	)

	runtime.ExecutionRunner.SetProgressCallback(nil)
	progressAdapter.Stop()
	renderPipelineRunSummary(stdout, pipelineModel, result, runErr)

	if runErr != nil {
		return fmt.Errorf("run pipeline %s: %w", pipelineModel.ID, runErr)
	}
	return nil
}

func loadPipelineForRun(ctx context.Context, pipelineStore *query.PipelineStore, id string) (*models.Pipeline, error) {
	pipelineModel, err := pipelineStore.GetByID(ctx, id)
	if err == nil {
		return pipelineModel, nil
	}

	pipelineModel, partialErr := pipelineStore.FindByPartialID(ctx, id)
	if partialErr != nil {
		return nil, fmt.Errorf("pipeline not found: %w", partialErr)
	}

	return pipelineModel, nil
}

func renderPipelineRunSummary(output io.Writer, pipelineModel *models.Pipeline, result *pl.ExecutionRunResult, runErr error) {
	if output == nil {
		output = io.Discard
	}

	status := "failed"
	executionID := ""
	duration := ""
	errText := ""

	if result != nil {
		status = result.Status
		executionID = result.ExecutionID
		duration = result.Duration.String()
		if strings.TrimSpace(result.ErrorMessage) != "" {
			errText = result.ErrorMessage
		}
	}
	if errText == "" && runErr != nil {
		errText = runErr.Error()
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Property", "Value"})
	t.AppendRow(table.Row{"Pipeline", pipelineModel.Name})
	t.AppendRow(table.Row{"Pipeline ID", pipelineModel.ID})
	if executionID != "" {
		t.AppendRow(table.Row{"Execution ID", executionID})
	}
	t.AppendRow(table.Row{"Status", status})
	if duration != "" {
		t.AppendRow(table.Row{"Duration", duration})
	}
	if errText != "" {
		t.AppendRow(table.Row{"Error", errText})
	}
	t.SetOutputMirror(output)
	t.SetStyle(table.StyleDefault)
	t.Render()
}
