package controller

import (
	"bpl/repository"
	"bpl/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ItemWishController struct {
	itemWishService *service.ItemWishService
	userService     *service.UserService
}

func NewItemWishController() *ItemWishController {
	return &ItemWishController{
		itemWishService: service.NewItemWishService(),
		userService:     service.NewUserService(),
	}
}

func setupItemWishController() []RouteInfo {
	e := NewItemWishController()
	basePath := "events/:event_id/teams/:team_id/item_wishes"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getItemWishesForTeamHandler(), Authenticated: true},
		{Method: "PUT", Path: "", HandlerFunc: e.saveItemWishHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/:wish_id", HandlerFunc: e.deleteItemWishHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetItemWishesForTeam
// @Description Get item wishes for a team in an event
// @Tags item_wishes
// @Produce json
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Success 200 {array} ItemWish
// @Router /events/{event_id}/teams/{team_id}/item_wishes [get]
func (e *ItemWishController) getItemWishesForTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid team ID"})
			return
		}
		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil || teamId != teamUser.TeamId {
			c.JSON(403, gin.H{"error": "You are not part of this team"})
			return
		}

		itemWishes, err := e.itemWishService.GetItemWishesForTeam(event.Id, teamId)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get item wishes"})
			return
		}
		c.JSON(200, itemWishes)

	}
}

// @id SaveItemWish
// @Description Save an item wish for a user in a team
// @Tags item_wishes
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Param item_wish body ItemWishRequest true "Item Wish"
// @Success 200 {object} ItemWish
// @Router /events/{event_id}/teams/{team_id}/item_wishes [put]
func (e *ItemWishController) saveItemWishHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil {
			c.JSON(403, gin.H{"error": "You are not part of a team"})
			return
		}

		var itemWishReq ItemWishRequest
		if err := c.ShouldBindJSON(&itemWishReq); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		itemWish := &repository.ItemWish{
			UserID:    teamUser.UserId,
			EventID:   event.Id,
			ItemField: itemWishReq.ItemField,
			Value:     itemWishReq.Value,
			Fulfilled: itemWishReq.Fulfilled,
		}

		savedItemWish, err := e.itemWishService.SaveItemWish(itemWish)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to save item wish"})
			return
		}
		c.JSON(200, toItemWishModel(*savedItemWish))
	}
}

// @id DeleteItemWish
// @Description Delete an item wish for a user in a team
// @Tags item_wishes
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Param wish_id path int true "Wish ID"
// @Success 204
// @Router /events/{event_id}/teams/{team_id}/item_wishes/{wish_id} [delete]
func (e *ItemWishController) deleteItemWishHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		wishId, err := strconv.Atoi(c.Param("wish_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid wish ID"})
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		itemWish, err := e.itemWishService.GetItemWishById(wishId)
		if err != nil {
			c.JSON(404, gin.H{"error": "Failed to get item wish"})
			return
		}
		if itemWish.UserID != user.Id {
			c.JSON(403, gin.H{"error": "You can only delete your own item wishes"})
			return
		}
		err = e.itemWishService.DeleteItemWish(wishId)
		c.Status(204)
	}
}

type ItemWish struct {
	UserId    int                  `json:"user_id" binding:"required"`
	ItemField repository.ItemField `json:"item_field" binding:"required"`
	Value     string               `json:"value" binding:"required"`
	Fulfilled bool                 `json:"fulfilled" binding:"required"`
	Priority  int                  `json:"priority"`
}

type ItemWishRequest struct {
	ItemField repository.ItemField `json:"item_field" binding:"required"`
	Value     string               `json:"value" binding:"required"`
	Fulfilled bool                 `json:"fulfilled" binding:"required"`
	Priority  int                  `json:"priority"`
}

func toItemWishModel(iw repository.ItemWish) *ItemWish {
	return &ItemWish{
		UserId:    iw.UserID,
		ItemField: iw.ItemField,
		Value:     iw.Value,
		Fulfilled: iw.Fulfilled,
	}
}
