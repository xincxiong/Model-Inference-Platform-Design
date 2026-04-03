package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xincxiong/model-inference-platform/backend/internal/config"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"github.com/xincxiong/model-inference-platform/backend/internal/handler"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"go.uber.org/zap"
)

func Setup(cfg *config.Config, s *store.Store, eng engine.Engine, logger *zap.Logger) *gin.Engine {
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

	chatH := handler.NewChatCompletionsHandler(s, eng)
	complH := handler.NewCompletionsHandler(s, eng)
	respH := handler.NewResponsesHandler(s, eng)
	embedH := handler.NewEmbeddingsHandler(s, eng)
	rerankH := handler.NewRerankHandler(s, eng)
	imagesH := handler.NewImagesHandler(s, eng)
	modelsH := handler.NewModelsHandler(s)
	keysH := handler.NewAPIKeysHandler(s)
	billingH := handler.NewBillingHandler(s)

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
	}

	return r
}
