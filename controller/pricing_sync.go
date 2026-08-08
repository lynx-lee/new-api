package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/QuantumNous/ai-bridge/model"
	"github.com/QuantumNous/ai-bridge/service"

	"github.com/gin-gonic/gin"
)

func SyncNow(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	n, err := service.SyncAllProviders(ctx)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "sync error: " + err.Error(), "data": gin.H{"stored": n}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "sync ok", "data": gin.H{"stored": n}})
}

func GetPricingDiffs(c *gin.Context) {
	diffs, err := service.GetPricingDiffs()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if diffs == nil {
		diffs = []model.ProviderPricing{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": diffs})
}

type applyReq struct {
	IDs []uint `json:"ids"`
}

func ApplyPricingDiff(c *gin.Context) {
	var req applyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := service.ApplyPricing(req.IDs); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "applied"})
}

func RejectPricingDiff(c *gin.Context) {
	var req applyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := service.RejectPricing(req.IDs); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "rejected"})
}
