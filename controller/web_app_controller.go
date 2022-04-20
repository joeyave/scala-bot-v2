package controller

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"github.com/joeyave/scala-bot-v2/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type WebAppController struct {
	Bot          *gotgbot.Bot
	EventService *services.EventService
}

func (h *WebAppController) CreateEvent(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "create-event.tmpl", gin.H{})
}

func (h *WebAppController) EditEvent(ctx *gin.Context) {

	hex := ctx.Param("id")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		// todo
	}

	event, err := h.EventService.FindOneByID(eventID)
	if err != nil {
		return
	}

	ctx.HTML(http.StatusOK, "edit-event.tmpl", gin.H{
		"Event": event,
	})
}
