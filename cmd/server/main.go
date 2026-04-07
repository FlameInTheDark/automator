package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FlameInTheDark/automator/internal/api"
	"github.com/FlameInTheDark/automator/internal/channels"
	"github.com/FlameInTheDark/automator/internal/config"
	"github.com/FlameInTheDark/automator/internal/crypto"
	"github.com/FlameInTheDark/automator/internal/db"
	"github.com/FlameInTheDark/automator/internal/db/query"
	"github.com/FlameInTheDark/automator/internal/node"
	"github.com/FlameInTheDark/automator/internal/node/action"
	"github.com/FlameInTheDark/automator/internal/node/logic"
	"github.com/FlameInTheDark/automator/internal/node/lua"
	"github.com/FlameInTheDark/automator/internal/node/trigger"
	"github.com/FlameInTheDark/automator/internal/pipeline"
	"github.com/FlameInTheDark/automator/internal/pipelineops"
	"github.com/FlameInTheDark/automator/internal/scheduler"
	"github.com/FlameInTheDark/automator/internal/shellcmd"
	"github.com/FlameInTheDark/automator/internal/skills"
	"github.com/FlameInTheDark/automator/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	var encryptor *crypto.Encryptor
	if cfg.EncryptionKey != "" {
		encryptor, err = crypto.NewEncryptor(cfg.EncryptionKey)
		if err != nil {
			log.Fatalf("failed to initialize encryptor: %v", err)
		}
	}

	clusterStore := query.NewClusterStore(database.DB, encryptor)
	llmProviderStore := query.NewLLMProviderStore(database.DB, encryptor)
	channelStore := query.NewChannelStore(database.DB, encryptor)
	channelContactStore := query.NewChannelContactStore(database.DB)
	pipelineStore := query.NewPipelineStore(database.DB)
	executionStore := query.NewExecutionStore(database.DB)
	workingDir, err := os.Getwd()
	if err != nil {
		log.Printf("failed to resolve working directory: %v", err)
		workingDir = "."
	}

	skillStore := skills.NewStore(filepath.Join(workingDir, ".agents", "skills"), 2*time.Second)
	if err := skillStore.Start(context.Background()); err != nil {
		log.Printf("failed to start skill store: %v", err)
	}
	defer skillStore.Stop()

	shellRunner := shellcmd.NewRunner(workingDir)

	var engine *pipeline.Engine
	var executionRunner *pipeline.ExecutionRunner
	channelService := channels.NewService(channelStore, channelContactStore, func(ctx context.Context, event trigger.ChannelEvent) error {
		if engine == nil || executionRunner == nil {
			return nil
		}

		eventCtx := trigger.WithChannelEvent(ctx, event)
		activePipelines, err := pipelineStore.ListActive(eventCtx)
		if err != nil {
			return err
		}

		executionContext := map[string]any{
			"channel_id":       event.ChannelID,
			"channel_name":     event.ChannelName,
			"channel_type":     event.ChannelType,
			"contact_id":       event.ContactID,
			"external_user_id": event.ExternalUserID,
			"external_chat_id": event.ExternalChatID,
			"text":             event.Text,
			"message":          event.Message,
		}

		for _, pipelineModel := range activePipelines {
			flowData, err := pipeline.ParseFlowData(pipelineModel.Nodes, pipelineModel.Edges)
			if err != nil {
				log.Printf("failed to parse channel pipeline %s: %v", pipelineModel.ID, err)
				continue
			}
			if !pipeline.HasMatchingRootTrigger(eventCtx, *flowData, "channel") {
				continue
			}

			result, err := executionRunner.Run(eventCtx, pipelineModel.ID, *flowData, "channel", executionContext)
			if err != nil {
				log.Printf("channel pipeline %s execution failed: %v", pipelineModel.ID, err)
				continue
			}
			if result.Status == "failed" && result.Error != nil {
				log.Printf("channel pipeline %s execution failed: %v", pipelineModel.ID, result.Error)
			}
		}

		return nil
	})

	registry := node.NewRegistry()
	registry.Register(node.TypeTriggerManual, &trigger.ManualTrigger{})
	registry.Register(node.TypeTriggerCron, &trigger.CronTrigger{})
	registry.Register(node.TypeTriggerWebhook, &trigger.WebhookTrigger{})
	registry.Register(node.TypeTriggerChannel, &trigger.ChannelMessageTrigger{})
	registry.Register(node.TypeActionListNodes, &action.ListNodesAction{Clusters: clusterStore})
	registry.Register(node.TypeActionListVMsCTs, &action.ListVMsCTsAction{Clusters: clusterStore})
	registry.Register(node.TypeActionVMStart, &action.VMStartAction{Clusters: clusterStore})
	registry.Register(node.TypeActionVMStop, &action.VMStopAction{Clusters: clusterStore})
	registry.Register(node.TypeActionVMClone, &action.VMCloneAction{Clusters: clusterStore})
	registry.Register(node.TypeActionHTTP, &action.HTTPAction{})
	registry.Register(node.TypeActionShell, &action.ShellCommandAction{Runner: shellRunner})
	registry.Register(node.TypeActionChannelSend, &action.ChannelSendAction{
		Channels: channelStore,
		Contacts: channelContactStore,
		Sender:   channelService,
	})
	registry.Register(node.TypeActionChannelWait, &action.ChannelSendAndWaitAction{
		Channels: channelStore,
		Contacts: channelContactStore,
		Sender:   channelService,
		Waiter:   channelService,
	})
	registry.Register(node.TypeLogicReturn, &logic.ReturnNode{})
	registry.Register(node.TypeLogicCondition, &logic.ConditionNode{})
	registry.Register(node.TypeLogicSwitch, &logic.SwitchNode{})
	registry.Register(node.TypeLogicMerge, &logic.MergeNode{})
	registry.Register(node.TypeLogicAggregate, &logic.AggregateNode{})
	llmPromptNode := &logic.LLMPromptNode{Providers: llmProviderStore}
	registry.Register(node.TypeLLMPrompt, llmPromptNode)
	registry.Register(node.TypeLLMPromptLegacy, llmPromptNode)
	registry.Register(node.TypeLLMAgent, &logic.LLMAgentNode{Providers: llmProviderStore, Skills: skillStore})
	registry.Register(node.TypeActionLua, &lua.LuaNode{})

	engine = pipeline.NewEngine(registry)
	wsHub := ws.NewHub()
	go wsHub.Run()
	executionRunner = pipeline.NewExecutionRunner(executionStore, engine, wsHub)
	pipelineInvoker := pipeline.NewInvoker(database.DB, pipelineStore, engine, executionRunner)
	pipelineRunner := func(ctx context.Context, pipelineID string) error {
		flowData, err := scheduler.LoadFlowData(database.DB, pipelineID)
		if err != nil {
			return err
		}
		result, err := executionRunner.Run(ctx, pipelineID, *flowData, "cron", nil)
		if err != nil {
			return err
		}
		if result.Status == "failed" && result.Error != nil {
			return result.Error
		}
		return nil
	}
	cronScheduler := scheduler.New(database.DB, pipelineRunner)
	pipelineManager := pipelineops.NewService(pipelineStore, cronScheduler)
	registry.Register(node.TypeToolListNodes, &action.ListNodesToolNode{Clusters: clusterStore})
	registry.Register(node.TypeToolListVMsCTs, &action.ListVMsCTsToolNode{Clusters: clusterStore})
	registry.Register(node.TypeToolVMStart, &action.VMStartToolNode{Clusters: clusterStore})
	registry.Register(node.TypeToolVMStop, &action.VMStopToolNode{Clusters: clusterStore})
	registry.Register(node.TypeToolVMClone, &action.VMCloneToolNode{Clusters: clusterStore})
	registry.Register(node.TypeToolHTTP, &action.HTTPToolNode{})
	registry.Register(node.TypeToolShell, &action.ShellCommandToolNode{Runner: shellRunner})
	registry.Register(node.TypeToolChannelWait, &action.ChannelSendAndWaitToolNode{
		Channels: channelStore,
		Contacts: channelContactStore,
		Sender:   channelService,
		Waiter:   channelService,
	})
	registry.Register(node.TypeActionRunPipeline, &action.RunPipelineAction{Runner: pipelineInvoker})
	registry.Register(node.TypeToolListPipelines, &action.PipelineListToolNode{Pipelines: pipelineStore})
	registry.Register(node.TypeToolCreatePipeline, &action.PipelineCreateToolNode{Manager: pipelineManager})
	registry.Register(node.TypeToolUpdatePipeline, &action.PipelineUpdateToolNode{Manager: pipelineManager})
	registry.Register(node.TypeToolDeletePipeline, &action.PipelineDeleteToolNode{Manager: pipelineManager})
	registry.Register(node.TypeToolRunPipeline, &action.PipelineRunToolNode{
		Pipelines: pipelineStore,
		Runner:    pipelineInvoker,
	})
	cronScheduler.Start()
	defer cronScheduler.Stop()

	if err := channelService.Start(); err != nil {
		log.Printf("failed to start channel service: %v", err)
	}
	defer channelService.Stop()

	app := api.New(api.Config{
		DB:              database,
		Scheduler:       cronScheduler,
		ChannelService:  channelService,
		EncryptionKey:   cfg.EncryptionKey,
		ExecutionRunner: executionRunner,
		WSHub:           wsHub,
		SkillStore:      skillStore,
		ShellRunner:     shellRunner,
	})

	go func() {
		if err := app.Listen(":" + cfg.Server.Port); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}()

	log.Printf("server started on port %s", cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
}
