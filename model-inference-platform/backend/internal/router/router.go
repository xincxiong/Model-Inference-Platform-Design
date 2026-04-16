package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xincxiong/model-inference-platform/backend/internal/config"
	"github.com/xincxiong/model-inference-platform/backend/internal/handler"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"go.uber.org/zap"
)

func Setup(cfg *config.Config, s *store.Store, mr *modelrouter.ModelRouter, logger *zap.Logger) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL, "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	chatH := handler.NewChatCompletionsHandler(s, mr)
	complH := handler.NewCompletionsHandler(s, mr)
	respH := handler.NewResponsesHandler(s, mr)
	embedH := handler.NewEmbeddingsHandler(s, mr)
	rerankH := handler.NewRerankHandler(s, mr)
	imagesH := handler.NewImagesHandler(s, mr)
	modelsH := handler.NewModelsHandler(s)
	keysH := handler.NewAPIKeysHandler(s)
	billingH := handler.NewBillingHandler(s)
	dedH := handler.NewDedicatedEndpointsHandler(s)
	ftH := handler.NewFineTuningHandler(s)
	filesH := handler.NewFilesHandler(s)
	batchH := handler.NewBatchHandler(s)
	dsH := handler.NewDatasetsHandler(s)
	deployH := handler.NewDeploymentsHandler(s)
	membersH := handler.NewMembersHandler(s)

	v0 := r.Group("/v0")
	v0.Use(middleware.AuthMiddleware(s.DB, s.Redis))
	{
		v0.GET("/dedicated_endpoints/templates", dedH.ListTemplates)
		v0.GET("/dedicated_endpoints", dedH.List)
		v0.POST("/dedicated_endpoints", dedH.Create)
		v0.PATCH("/dedicated_endpoints/:id", dedH.Patch)
		v0.DELETE("/dedicated_endpoints/:id", dedH.Delete)
	}

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
		v1.GET("/models", modelsH.List)
		v1.POST("/fine_tuning/jobs", ftH.Create)
		v1.GET("/fine_tuning/jobs", ftH.List)
		v1.GET("/fine_tuning/jobs/:id", ftH.Get)
		v1.POST("/fine_tuning/jobs/:id/cancel", ftH.Cancel)

		// Files API
		v1.POST("/files", filesH.Upload)
		v1.GET("/files", filesH.List)
		v1.GET("/files/:id", filesH.Get)
		v1.DELETE("/files/:id", filesH.Delete)
		v1.GET("/files/:id/content", filesH.GetContent)

		// Batch API
		v1.POST("/batches", batchH.Create)
		v1.GET("/batches", batchH.List)
		v1.GET("/batches/:id", batchH.Get)
		v1.POST("/batches/:id/cancel", batchH.Cancel)

		// Datasets API
		v1.POST("/datasets", dsH.Create)
		v1.GET("/datasets", dsH.List)
		v1.GET("/datasets/:id", dsH.Get)
		v1.PATCH("/datasets/:id", dsH.Patch)
		v1.DELETE("/datasets/:id", dsH.Delete)
		v1.GET("/datasets/:id/content", dsH.GetContent)
		v1.GET("/datasets/:id/export", dsH.Export)
		v1.GET("/datasets/:id/query", dsH.Query)

		// Deployments API
		v1.POST("/deployments", deployH.Create)
		v1.GET("/deployments", deployH.List)
		v1.GET("/deployments/:id", deployH.Get)
		v1.PATCH("/deployments/:id", deployH.Patch)
		v1.DELETE("/deployments/:id", deployH.Delete)
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

		// Members API
		api.POST("/members", membersH.Invite)
		api.GET("/members", membersH.List)
		api.PATCH("/members/:id", membersH.PatchRole)
		api.DELETE("/members/:id", membersH.Remove)
	}

	return r
}
