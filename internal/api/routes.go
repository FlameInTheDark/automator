package api

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/FlameInTheDark/automator/internal/api/handlers"
	"github.com/FlameInTheDark/automator/internal/channels"
	"github.com/FlameInTheDark/automator/internal/crypto"
	"github.com/FlameInTheDark/automator/internal/db"
	"github.com/FlameInTheDark/automator/internal/db/query"
	"github.com/FlameInTheDark/automator/internal/pipeline"
	"github.com/FlameInTheDark/automator/internal/scheduler"
	"github.com/FlameInTheDark/automator/internal/shellcmd"
	"github.com/FlameInTheDark/automator/internal/skills"
	"github.com/FlameInTheDark/automator/internal/ws"
)

//go:embed web/dist
var embeddedFS embed.FS

type Config struct {
	DB              *db.DB
	Scheduler       *scheduler.Scheduler
	ChannelService  *channels.Service
	EncryptionKey   string
	ExecutionRunner *pipeline.ExecutionRunner
	WSHub           *ws.Hub
	SkillStore      skills.Reader
	ShellRunner     shellcmd.Runner
}

func New(cfg Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Proxmox Automator",
		ServerHeader: "Automator",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	var encryptor *crypto.Encryptor
	if cfg.EncryptionKey != "" {
		var err error
		encryptor, err = crypto.NewEncryptor(cfg.EncryptionKey)
		if err != nil {
			log.Printf("warning: failed to create encryptor: %v", err)
		}
	}

	wsHub := cfg.WSHub
	if wsHub == nil {
		wsHub = ws.NewHub()
		go wsHub.Run()
	}

	clusterStore := query.NewClusterStore(cfg.DB.DB, encryptor)
	clusterHandler := handlers.NewClusterHandler(clusterStore)
	channelStore := query.NewChannelStore(cfg.DB.DB, encryptor)
	channelContactStore := query.NewChannelContactStore(cfg.DB.DB)
	channelHandler := handlers.NewChannelHandler(channelStore, channelContactStore, cfg.ChannelService)

	pipelineStore := query.NewPipelineStore(cfg.DB.DB)
	llmProviderStore := query.NewLLMProviderStore(cfg.DB.DB, encryptor)
	executionStore := query.NewExecutionStore(cfg.DB.DB)
	llmProviderHandler := handlers.NewLLMProviderHandler(llmProviderStore)
	dashboardHandler := handlers.NewDashboardHandler(clusterStore, pipelineStore, executionStore, channelStore, cfg.Scheduler)

	pipelineRunHandler := handlers.NewPipelineRunHandler(
		pipelineStore,
		cfg.ExecutionRunner,
	)
	llmChatHandler := handlers.NewLLMChatHandler(llmProviderStore, clusterStore, pipelineStore, cfg.ExecutionRunner, cfg.Scheduler, cfg.SkillStore, cfg.ShellRunner)
	executionHandler := handlers.NewExecutionHandler(executionStore, cfg.ExecutionRunner)

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	api.Get("/dashboard/stats", dashboardHandler.Stats)

	clusters := api.Group("/clusters")
	clusters.Get("/", clusterHandler.List)
	clusters.Post("/", clusterHandler.Create)
	clusters.Get("/:id", clusterHandler.Get)
	clusters.Put("/:id", clusterHandler.Update)
	clusters.Delete("/:id", clusterHandler.Delete)

	channelRoutes := api.Group("/channels")
	channelRoutes.Get("/", channelHandler.List)
	channelRoutes.Post("/", channelHandler.Create)
	channelRoutes.Get("/:id", channelHandler.Get)
	channelRoutes.Put("/:id", channelHandler.Update)
	channelRoutes.Delete("/:id", channelHandler.Delete)
	channelRoutes.Get("/:id/contacts", channelHandler.ListContacts)
	channelRoutes.Post("/connect", channelHandler.Connect)

	pipelineHandler := pipelineHandler(pipelineStore, cfg.Scheduler)
	pipelines := api.Group("/pipelines")
	pipelines.Get("/", pipelineHandler.List)
	pipelines.Post("/", pipelineHandler.Create)
	pipelines.Get("/:id", pipelineHandler.Get)
	pipelines.Put("/:id", pipelineHandler.Update)
	pipelines.Delete("/:id", pipelineHandler.Delete)
	pipelines.Post("/:id/run", pipelineRunHandler.Run)

	llmProviders := api.Group("/llm-providers")
	llmProviders.Get("/", llmProviderHandler.List)
	llmProviders.Post("/", llmProviderHandler.Create)
	llmProviders.Get("/:id", llmProviderHandler.Get)
	llmProviders.Put("/:id", llmProviderHandler.Update)
	llmProviders.Delete("/:id", llmProviderHandler.Delete)
	llmProviders.Get("/:id/models", llmProviderHandler.ListModels)

	api.Post("/llm/chat", llmChatHandler.Chat)

	executions := api.Group("/executions")
	executions.Get("/pipelines/:id", executionHandler.ListByPipeline)
	executions.Get("/pipelines/:id/active", executionHandler.ListActiveByPipeline)
	executions.Get("/:executionId", executionHandler.Get)
	executions.Post("/:executionId/cancel", executionHandler.Cancel)

	app.Get("/ws/:channel", ws.WSUpgrader(wsHub))

	app.Use("*", serveEmbedded())

	return app
}

func pipelineHandler(store *query.PipelineStore, scheduler *scheduler.Scheduler) *handlers.PipelineHandler {
	return handlers.NewPipelineHandler(store, scheduler)
}

func serveEmbedded() fiber.Handler {
	embedded, err := fs.Sub(embeddedFS, "web/dist")
	if err != nil {
		panic(err)
	}

	return filesystem.New(filesystem.Config{
		Root:         http.FS(embedded),
		Index:        "index.html",
		NotFoundFile: "index.html",
	})
}
