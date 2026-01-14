package handlers

import (
	"gibraltar/internal/models"
	"gibraltar/internal/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigHandlers struct {
	deps *services.Dependencies
}

func NewConfigHandler(deps *services.Dependencies) *ConfigHandlers {
	return &ConfigHandlers{
		deps: deps,
	}
}

func (h *ConfigHandlers) CurrentAvailableConfigs(c *gin.Context) {
	configs, ok := h.deps.Cache.Get(services.AvailableKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "configs unavailable retry later",
		})
	}
	resultString := ""
	for _, v := range configs {
		resultString += v.URL + "\n"
	}
	c.String(http.StatusOK, resultString)
}

func (h *ConfigHandlers) RequestConfigsUpdate(c *gin.Context) {
	configs, ok := h.deps.Cache.Get(services.AllKey)
	if !ok {
		go func() {
			h.deps.CalculateAvailableServers()
		}()
		c.String(http.StatusOK, "Your request has been accepted and is being processed.\nPlease try requesting a list of servers in 5 minutes.\n")

	}
	go func(configs []models.VlessConfig) {
		for i := 0; i < services.TestAttempt; i++ {
			services.TestConfigs(configs, h.deps.VlessTestService)
			h.deps.Cache.Set(services.AvailableKey, configs[:services.SortConfigsByTestResult(configs)])
			log.Println("UpdatedSuccessfully")
		}
	}(configs)

	c.String(http.StatusOK, "Your request has been accepted and is being processed.\nPlease try requesting a list of servers in 5 minutes.\n")

}
