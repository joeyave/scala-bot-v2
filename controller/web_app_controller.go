package controller

import (
	"encoding/json"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strconv"
)

type WebAppController struct {
	Bot          *gotgbot.Bot
	EventService *service.EventService
	UserService  *service.UserService
	BandService  *service.BandService
}

func (h *WebAppController) CreateEvent(ctx *gin.Context) {

	hex := ctx.Query("bandId")
	bandID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return
	}

	band, err := h.BandService.FindOneByID(bandID)
	if err != nil {
		return
	}

	event := &entity.Event{
		BandID: bandID,
		Band:   band,
	}
	eventJsonBytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	eventNames, err := h.EventService.GetMostFrequentEventNames(bandID, 4)
	if err != nil {
		return
	}

	ctx.HTML(http.StatusOK, "edit-event.go.html", gin.H{
		"EventNames": eventNames,
		"EventJS":    string(eventJsonBytes),
		"Action":     "create",
	})
}

func (h *WebAppController) EditEvent(ctx *gin.Context) {

	hex := ctx.Param("id")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return
	}

	messageID := ctx.Query("messageId")
	chatID := ctx.Query("chatId")

	event, err := h.EventService.FindOneByID(eventID)
	if err != nil {
		return
	}

	eventJsonBytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	eventNames, err := h.EventService.GetMostFrequentEventNames(event.BandID, 4)
	if err != nil {
		return
	}

	ctx.HTML(http.StatusOK, "edit-event.go.html", gin.H{
		"MessageID":  messageID,
		"ChatID":     chatID,
		"EventNames": eventNames,
		"Event":      event,
		"EventJS":    string(eventJsonBytes),
		"Action":     "edit",
	})
}

func (h *WebAppController) EditEventConfirm(ctx *gin.Context) {

	hex := ctx.Param("id")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return
	}

	messageIDStr := ctx.Query("messageId")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		return
	}
	chatIDStr := ctx.Query("chatId")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return
	}

	var event *entity.Event
	err = ctx.ShouldBindJSON(&event)
	if err != nil {
		return
	}
	event.ID = eventID

	updatedEvent, err := h.EventService.UpdateOne(*event)
	if err != nil {
		return
	}

	html := h.EventService.ToHtmlStringByEvent(*updatedEvent, "ru")
	if err != nil {
		return
	}

	user, err := h.UserService.FindOneByID(chatID)
	if err != nil {
		return
	}

	markup := gotgbot.InlineKeyboardMarkup{}
	markup.InlineKeyboard = keyboard.EventEdit(event, user, chatID, messageID, "ru")

	user.CallbackCache = entity.CallbackCache{
		MessageID: messageID,
		ChatID:    chatID,
	}
	text := user.CallbackCache.AddToText(html)

	_, _, err = h.Bot.EditMessageText(text, &gotgbot.EditMessageTextOpts{
		ChatId:                chatID,
		MessageId:             messageID,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
		ReplyMarkup:           markup,
	})

	//_, err = h.EventService.FindOneByID(eventID)
	//if err != nil {
	//	return
	//}
	//
	//queryID := ctx.Query("queryId")
	//
	//h.Bot.AnswerWebAppQuery(queryID, nil) // todo
	//
	//fmt.Println("got callback from web app")

	ctx.Status(http.StatusOK)
}
