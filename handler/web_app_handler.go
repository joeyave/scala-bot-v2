package handler

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"scala-bot-v2/service"
)

type WebAppHandler struct {
	Bot          *gotgbot.Bot
	EventService *service.EventService
}

func (h *WebAppHandler) CreateEvent(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "create-event.tmpl", gin.H{})
}

func (h *WebAppHandler) EditEvent(ctx *gin.Context) {

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
