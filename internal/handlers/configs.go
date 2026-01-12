package handlers

import (
	"gibraltar/internal/services"
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

func (h ConfigHandlers) CurrentAvailableConfigs(c *gin.Context) {
	configs, ok := h.deps.Cache.Get(services.СacheKey)
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
