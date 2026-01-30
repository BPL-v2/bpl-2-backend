package controller

import (
	"bpl/service"

	"github.com/gin-gonic/gin"
)

type ItemController struct {
	itemService *service.ItemService
}

func NewItemController() *ItemController {
	return &ItemController{
		itemService: service.NewItemService(),
	}
}

func setupItemController() []RouteInfo {
	e := NewItemController()
	basePath := "items"
	routes := []RouteInfo{
		{Method: "GET", Path: "/map", HandlerFunc: e.getItemMapHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetItemMap
// @Description Returns a map of item types to item-name-to-ID maps
// @Tags items
// @Produce json
// @Success 200 {object} map[string]map[string]int
// @Router /items/map [get]
func (e *ItemController) getItemMapHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := e.itemService.GetItemMap()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, items)
	}
}
