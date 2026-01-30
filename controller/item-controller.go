package controller

import (
	"bpl/repository"
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
		{Method: "POST", Path: "/bulk", HandlerFunc: e.createItemsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
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

// @id CreateItems
// @Description Creates multiple items
// @Tags items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param item_type query string true "Item type"
// @Param body body []string true "Item names"
// @Success 201 {object} map[string]map[string]int
// @Router /items/bulk [post]
func (e *ItemController) createItemsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		itemType := c.Query("item_type")
		if itemType == "" {
			c.JSON(400, gin.H{"error": "item_type is required"})
			return
		}
		var itemNames []string
		if err := c.ShouldBindJSON(&itemNames); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		if err := e.itemService.SaveItems(itemNames, itemType); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		itemMap, err := e.itemService.GetItemMap()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, itemMap)
	}
}
