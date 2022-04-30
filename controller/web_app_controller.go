package controller

import (
	"encoding/json"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strconv"
	"strings"
)

type WebAppController struct {
	Bot               *gotgbot.Bot
	EventService      *service.EventService
	UserService       *service.UserService
	BandService       *service.BandService
	DriveFileService  *service.DriveFileService
	SongService       *service.SongService
	VoiceService      *service.VoiceService
	MembershipService *service.MembershipService
	RoleService       *service.RoleService
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

type Tag struct {
	Name       string
	IsSelected bool
}

func (h *WebAppController) CreateSong(ctx *gin.Context) {

	hex := ctx.Query("userId")
	userID, err := strconv.ParseInt(hex, 10, 64)
	if err != nil {
		return
	}

	user, err := h.UserService.FindOneByID(userID)
	if err != nil {
		return
	}

	allTags, err := h.SongService.GetTags(user.BandID)
	if err != nil {
		return
	}

	var songTags []*Tag
	for _, tag := range allTags {
		songTags = append(songTags, &Tag{Name: tag, IsSelected: false})
	}
	//band, err := h.BandService.FindOneByID(bandID)
	//if err != nil {
	//	return
	//}

	//event := &entity.Event{
	//	BandID: bandID,
	//	Band:   band,
	//}
	//eventJsonBytes, err := json.Marshal(event)
	//if err != nil {
	//	return
	//}

	ctx.HTML(http.StatusOK, "edit-song.go.html", gin.H{
		//"EventJS":    string(eventJsonBytes),
		"Keys":   valuesForSelect("?", keys),
		"BPMs":   bpmsForSelect("?"),
		"Times":  valuesForSelect("?", times),
		"Tags":   songTags,
		"Action": "create",
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

func (h *WebAppController) EditSong(ctx *gin.Context) {

	hex := ctx.Param("id")
	fmt.Print(hex)
	songID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return
	}

	userIDStr := ctx.Query("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return
	}

	user, err := h.UserService.FindOneByID(userID)
	if err != nil {
		return
	}

	messageID := ctx.Query("messageId")
	chatID := ctx.Query("chatId")

	song, err := h.SongService.FindOneByID(songID)
	if err != nil {
		return
	}

	songJsonBytes, err := json.Marshal(song)
	if err != nil {
		return
	}

	allTags, err := h.SongService.GetTags(user.BandID)
	if err != nil {
		return
	}

	var songTags []*Tag
	for _, tag := range allTags {
		isSelected := false
		for _, songTag := range song.Tags {
			if songTag == tag {
				isSelected = true
				break
			}
		}
		songTags = append(songTags, &Tag{Name: tag, IsSelected: isSelected})
	}

	lyrics := h.DriveFileService.GetText(song.DriveFileID)

	ctx.HTML(http.StatusOK, "edit-song.go.html", gin.H{
		"MessageID": messageID,
		"ChatID":    chatID,

		"Keys":   valuesForSelect(strings.TrimSpace(song.PDF.Key), keys),
		"BPMs":   bpmsForSelect(strings.TrimSpace(song.PDF.BPM)),
		"Times":  valuesForSelect(strings.TrimSpace(song.PDF.Time), times),
		"Tags":   songTags,
		"Lyrics": lyrics,

		"Song":   song,
		"SongJS": string(songJsonBytes),
		"Action": "edit",
	})
}

func (h *WebAppController) EditSongConfirm(ctx *gin.Context) {

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

var keys = []string{"C", "C#", "Db", "D", "D#", "Eb", "E", "F", "F#", "Gb", "G", "G#", "Ab", "A", "A#", "Bb", "B"}
var times = []string{"4/4", "3/4", "6/8", "2/2"}

type SelectEntity struct {
	Name       string
	Value      string
	IsSelected bool
}

func valuesForSelect(songKey string, values []string) []*SelectEntity {
	var keysForSelect []*SelectEntity

	exists := false
	for _, key := range values {
		if songKey == key {
			exists = true
			break
		}
	}
	if !exists {
		if songKey == "" || songKey == "?" {
			keysForSelect = append(keysForSelect, &SelectEntity{Name: "Key", Value: "?", IsSelected: true})
		} else {
			keysForSelect = append(keysForSelect, &SelectEntity{Name: songKey, Value: songKey, IsSelected: true})
		}
	}

	for _, key := range values {
		keysForSelect = append(keysForSelect, &SelectEntity{Name: key, Value: key, IsSelected: key == songKey})
	}

	return keysForSelect
}

func bpmsForSelect(songBPM string) []*SelectEntity {
	var bpmsForSelect []*SelectEntity

	songBPMInt, err := strconv.Atoi(songBPM)
	if err != nil || songBPMInt < 60 || songBPMInt > 180 {
		bpmsForSelect = append(bpmsForSelect, &SelectEntity{
			IsSelected: true,
			Name:       "BPM",
			Value:      "?",
		})
	}

	for i := 60; i < 180; i++ {
		bpmsForSelect = append(bpmsForSelect, &SelectEntity{
			IsSelected: songBPMInt == i,
			Name:       strconv.Itoa(i),
			Value:      strconv.Itoa(i),
		})
	}

	return bpmsForSelect
}
