package router

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xincxiong/model-inference-platform/backend/internal/circuitbreaker"
	"github.com/xincxiong/model-inference-platform/backend/internal/config"
	"github.com/xincxiong/model-inference-platform/backend/internal/hami"
	"github.com/xincxiong/model-inference-platform/backend/internal/handler"
	"github.com/xincxiong/model-inference-platform/backend/internal/health"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"github.com/xincxiong/model-inference-platform/backend/internal/volcano"
	"go.uber.org/zap"
)

// SetupInferenceRouter configures the inference data plane (推理请求处理).
func SetupInferenceRouter(cfg *config.Config, s *store.Store, mr *modelrouter.ModelRouter, hc *health.Checker, cb *circuitbreaker.Manager, logger *zap.Logger) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL, "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		report := hc.Latest()
		status := http.StatusOK
		if report.Status == health.StatusUnhealthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	})
	r.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/health/ready", func(c *gin.Context) {
		report := hc.Check()
		status := http.StatusOK
		if report.Status == health.StatusUnhealthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	})
	r.GET("/health/circuit-breakers", func(c *gin.Context) { c.JSON(http.StatusOK, cb.Snapshot()) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	chatH := handler.NewChatCompletionsHandler(s, mr, s.SemanticCache, cfg.SemanticCache.EmbeddingModel)
	complH := handler.NewCompletionsHandler(s, mr, s.SemanticCache, cfg.SemanticCache.EmbeddingModel)
	respH := handler.NewResponsesHandler(s, mr)
	embedH := handler.NewEmbeddingsHandler(s, mr)
	rerankH := handler.NewRerankHandler(s, mr)
	imagesH := handler.NewImagesHandler(s, mr)
	videoH := handler.NewVideoHandler(s, mr)
	audioH := handler.NewAudioHandler(s, mr)
	modelsH := handler.NewModelsHandler(s)

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(s.DB, s.Redis))
	v1.Use(middleware.RateLimitMiddleware(s.Redis))
	{
		v1.POST("/chat/completions", chatH.Create)
		v1.POST("/completions", complH.Create)
		v1.POST("/responses", respH.Create)
		v1.GET("/responses/:id", respH.Get)
		v1.GET("/responses/:id/input_items", respH.GetInputItems)
		v1.POST("/embeddings", embedH.Create)
		v1.POST("/rerank", rerankH.Create)
		v1.POST("/images/generations", imagesH.Generate)
		v1.POST("/videos/generations", videoH.Generate)
		v1.POST("/audio/transcriptions", audioH.CreateTranscription)
		v1.POST("/audio/speech", audioH.CreateSpeech)
		v1.GET("/models", modelsH.List)
	}

	return r
}

// SetupManagementRouter configures the control plane (管理操作处理).
func SetupManagementRouter(cfg *config.Config, s *store.Store, mr *modelrouter.ModelRouter, hc *health.Checker, cb *circuitbreaker.Manager, hamiScheduler *hami.Scheduler, volcanoClient *volcano.Client, logger *zap.Logger) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL, "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		report := hc.Latest()
		status := http.StatusOK
		if report.Status == health.StatusUnhealthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	})
	r.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/health/ready", func(c *gin.Context) {
		report := hc.Check()
		status := http.StatusOK
		if report.Status == health.StatusUnhealthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	})
	r.GET("/health/circuit-breakers", func(c *gin.Context) { c.JSON(http.StatusOK, cb.Snapshot()) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	modelsH := handler.NewModelsHandler(s)
	keysH := handler.NewAPIKeysHandler(s)
	billingH := handler.NewBillingHandler(s)
	dedH := handler.NewDedicatedEndpointsHandler(s, hamiScheduler, logger)
	ftH := handler.NewFineTuningHandler(s, volcanoClient, cfg.Volcano, logger)
	filesH := handler.NewFilesHandler(s)
	batchH := handler.NewBatchHandler(s)
	dsH := handler.NewDatasetsHandler(s)
	deployH := handler.NewDeploymentsHandler(s)
	membersH := handler.NewMembersHandler(s)
	mvH := handler.NewModelVersionsHandler(s)
	poolSubH := handler.NewPoolSubscriptionsHandler(s)
	computePoolH := handler.NewComputePoolsHandler(s)

	v0 := r.Group("/v0")
	v0.Use(middleware.AuthMiddleware(s.DB, s.Redis))
	{
		v0.GET("/dedicated_endpoints/templates", dedH.ListTemplates)
		v0.GET("/dedicated_endpoints", dedH.List)
		v0.POST("/dedicated_endpoints", dedH.Create)
		v0.PATCH("/dedicated_endpoints/:id", dedH.Patch)
		v0.DELETE("/dedicated_endpoints/:id", dedH.Delete)

		v0.GET("/pools/skus", poolSubH.ListSKUs)
		v0.GET("/pools/subscriptions", poolSubH.ListSubscriptions)
		v0.GET("/pools/subscriptions/overview", poolSubH.Overview)
		v0.POST("/pools/subscriptions", poolSubH.Purchase)
		v0.POST("/pools/subscriptions/:id/cancel", poolSubH.Cancel)
		v0.POST("/pools/subscriptions/:id/renew", poolSubH.Renew)
		v0.GET("/pools/subscriptions/:id/invoices", poolSubH.ListInvoices)

		v0.GET("/pools/compute", computePoolH.List)
		v0.POST("/pools/compute", computePoolH.Create)
		v0.GET("/pools/compute/:id", computePoolH.Get)
		v0.PATCH("/pools/compute/:id", computePoolH.Patch)
		v0.POST("/pools/compute/:id/archive", computePoolH.Archive)
		v0.GET("/pools/compute/:id/usage", computePoolH.Usage)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(s.DB, s.Redis))
	{
		v1.POST("/fine_tuning/jobs", ftH.Create)
		v1.GET("/fine_tuning/jobs", ftH.List)
		v1.GET("/fine_tuning/jobs/:id", ftH.Get)
		v1.POST("/fine_tuning/jobs/:id/cancel", ftH.Cancel)

		v1.POST("/files", filesH.Upload)
		v1.GET("/files", filesH.List)
		v1.GET("/files/:id", filesH.Get)
		v1.DELETE("/files/:id", filesH.Delete)
		v1.GET("/files/:id/content", filesH.GetContent)

		v1.POST("/batches", batchH.Create)
		v1.GET("/batches", batchH.List)
		v1.GET("/batches/:id", batchH.Get)
		v1.POST("/batches/:id/cancel", batchH.Cancel)

		v1.POST("/datasets", dsH.Create)
		v1.GET("/datasets", dsH.List)
		v1.GET("/datasets/:id", dsH.Get)
		v1.PATCH("/datasets/:id", dsH.Patch)
		v1.DELETE("/datasets/:id", dsH.Delete)
		v1.GET("/datasets/:id/content", dsH.GetContent)
		v1.GET("/datasets/:id/export", dsH.Export)
		v1.GET("/datasets/:id/query", dsH.Query)

		v1.POST("/deployments", deployH.Create)
		v1.GET("/deployments", deployH.List)
		v1.GET("/deployments/:id", deployH.Get)
		v1.PATCH("/deployments/:id", deployH.Patch)
		v1.DELETE("/deployments/:id", deployH.Delete)

		v1.GET("/models/:model_id/versions", mvH.List)
		v1.POST("/models/:model_id/versions", mvH.Create)
		v1.PATCH("/models/:model_id/versions/:version_id", mvH.Patch)
		v1.POST("/models/:model_id/versions/:version_id/activate", mvH.Activate)
		v1.POST("/models/:model_id/versions/:version_id/deactivate", mvH.Deactivate)
		v1.PUT("/models/:model_id/ab-test", mvH.SetABTest)
	}

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(s.DB, s.Redis))
	{
		api.GET("/models", modelsH.ListDetailed)
		api.POST("/api-keys", keysH.Create)
		api.GET("/api-keys", keysH.List)
		api.DELETE("/api-keys/:id", keysH.Delete)
		api.GET("/usage", billingH.GetUsage)
		api.POST("/billing/redeem", billingH.RedeemPromo)

		api.POST("/members", membersH.Invite)
		api.GET("/members", membersH.List)
		api.PATCH("/members/:id", membersH.PatchRole)
		api.DELETE("/members/:id", membersH.Remove)
	}

	return r
}
