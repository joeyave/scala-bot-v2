package controller

import (
	"encoding/json"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"github.com/joeyave/scala-bot-v2/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type WebAppController struct {
	Bot          *gotgbot.Bot
	EventService *service.EventService
	UserService  *service.UserService
}

func (h *WebAppController) CreateEvent(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "create-event.tmpl", gin.H{})
}

func (h *WebAppController) EditEvent(ctx *gin.Context) {

	hex := ctx.Param("id")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return
	}

	event, err := h.EventService.FindOneByID(eventID)
	if err != nil {
		return
	}

	eventJsonBytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	ctx.HTML(http.StatusOK, "edit-event.tmpl", gin.H{
		"Event": string(eventJsonBytes),
	})
}

func (h *WebAppController) EditEventConfirm(ctx *gin.Context) {

	hex := ctx.Param("id")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		// todo
	}

	_, err = h.EventService.FindOneByID(eventID)
	if err != nil {
		return
	}

	queryID := ctx.Query("queryId")

	h.Bot.AnswerWebAppQuery(queryID, nil) // todo

	fmt.Println("got callback from web app")

	ctx.Status(http.StatusOK)
}
