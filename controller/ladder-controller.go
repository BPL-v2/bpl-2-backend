package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"

	"github.com/gin-gonic/gin"
)

type LadderController struct {
	service *service.LadderService
}

type LadderEntry struct {
	UserId        int    `json:"user_id" binding:"required"`
	CharacterName string `json:"character_name" binding:"required"`
	AccountName   string `json:"account_name" binding:"required"`
	Level         int    `json:"level" binding:"required"`
	Class         string `json:"character_class" binding:"required"`
	Experience    int    `json:"experience" binding:"required"`
	Delve         int    `json:"delve" binding:"required"`
	Rank          int    `json:"rank" binding:"required"`
}

func toLadderEntryResponse(entry *repository.LadderEntry) *LadderEntry {
	if entry == nil {
		return nil
	}
	return &LadderEntry{
		CharacterName: entry.Character,
		AccountName:   entry.Account,
		Level:         entry.Level,
		Class:         entry.Class,
		Experience:    entry.Experience,
		Delve:         entry.Delve,
		Rank:          entry.Rank,
	}
}

func NewLadderController() *LadderController {
	return &LadderController{service: service.NewLadderService()}
}

func setupLadderController() []RouteInfo {
	c := NewLadderController()
	baseUrl := "events/:event_id/ladder"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: c.getLadderHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetLadder
// @Description Get the ladder for an event
// @Tags ladder
// @Accept json
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {array} LadderEntry
// @Router /events/{event_id}/ladder [get]
func (c *LadderController) getLadderHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		ladder, err := c.service.GetLadderForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, utils.Map(ladder, toLadderEntryResponse))
	}
}
