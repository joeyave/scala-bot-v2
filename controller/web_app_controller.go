package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
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

	ctx.Status(http.StatusOK)
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

	var songTags []*SelectEntity
	for _, tag := range allTags {
		songTags = append(songTags, &SelectEntity{Name: tag, IsSelected: false})
	}

	ctx.HTML(http.StatusOK, "edit-song.go.html", gin.H{
		"Keys":   valuesForSelect("?", keys, "Key"),
		"BPMs":   valuesForSelect("?", bpms, "BPM"),
		"Times":  valuesForSelect("?", times, "Time"),
		"Tags":   songTags,
		"Action": "create",
	})
}

func (h *WebAppController) EditSong(ctx *gin.Context) {

	fmt.Println(ctx.Request.RequestURI)
	hex := ctx.Param("id")
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

	var songTags []*SelectEntity
	for _, tag := range allTags {
		isSelected := false
		for _, songTag := range song.Tags {
			if songTag == tag {
				isSelected = true
				break
			}
		}
		songTags = append(songTags, &SelectEntity{Name: tag, IsSelected: isSelected})
	}

	lyrics := h.DriveFileService.GetText(song.DriveFileID)

	start := time.Now()
	//doc, err := html.Parse(lyrics)
	//if err != nil {
	//	return
	//}

	//bn, err := Body(doc)
	//if err != nil {
	//	return
	//}
	//body := renderNode(bn)

	sectionsNumber, err := h.DriveFileService.GetSectionsNumber(song.DriveFileID)
	if err != nil {
		return
	}

	sectionsSelect := []*SelectEntity{{Name: "В конец документа", Value: "-1", IsSelected: true}}
	for i := 0; i < sectionsNumber; i++ {
		sectionsSelect = append(sectionsSelect, &SelectEntity{Name: fmt.Sprintf("Вместо %d секции", i+1), Value: fmt.Sprint(i)})
	}

	fmt.Println(time.Since(start).String())
	ctx.HTML(http.StatusOK, "edit-song.go.html", gin.H{
		"MessageID": messageID,
		"ChatID":    chatID,
		"UserID":    userID,

		"Keys":     valuesForSelect(strings.TrimSpace(song.PDF.Key), keys, "Key"),
		"Sections": sectionsSelect,
		"BPMs":     valuesForSelect(strings.TrimSpace(song.PDF.BPM), bpms, "BPM"),
		"Times":    valuesForSelect(strings.TrimSpace(song.PDF.Time), times, "Time"),
		"Tags":     songTags,
		"Lyrics":   lyrics,

		"Song":   song,
		"SongJS": string(songJsonBytes),

		"Action": "edit",
	})
}

var fontSizeRegex = regexp.MustCompile("font-size:.*?;")

func Body(doc *html.Node) (*html.Node, error) {
	var body *html.Node
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "body" {
			body = node
			//return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			for i, attribute := range child.Attr {
				if attribute.Key == "style" {
					child.Attr[i].Val = fontSizeRegex.ReplaceAllString(attribute.Val, "font-size:1em;")
				}
			}
			crawler(child)
		}
	}
	crawler(doc)

	if body == nil {
		return nil, errors.New("Missing <body> in the node tree")
	}

	if body.FirstChild != nil && body.FirstChild.Data == "div" {
		body.RemoveChild(body.FirstChild)
	}
	body.Data = "div"
	body.Attr = []html.Attribute{{Key: "style", Val: "font-size:1em;"}}

	return body, nil
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

type EditSongData struct {
	TransposeSection string   `json:"transposeSection"`
	Name             string   `json:"name"`
	Key              string   `json:"key"`
	BPM              string   `json:"bpm"`
	Time             string   `json:"time"`
	Tags             []string `json:"tags"`
}

var keyRegex = regexp.MustCompile(`(?i)key:(.*?);`)
var bpmRegex = regexp.MustCompile(`(?i)bpm:(.*?);`)
var timeRegex = regexp.MustCompile(`(?i)time:(.*?);`)

func (h *WebAppController) EditSongConfirm(ctx *gin.Context) {

	hex := ctx.Param("id")
	songID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return
	}

	userIDStr := ctx.Query("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
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

	var data *EditSongData
	err = ctx.ShouldBindJSON(&data)
	if err != nil {
		return
	}

	song, err := h.SongService.FindOneByID(songID)
	if err != nil {
		return
	}

	song.Tags = data.Tags

	if song.PDF.Name != data.Name {
		err := h.DriveFileService.Rename(song.DriveFileID, data.Name)
		if err != nil {
			return
		}
		song.PDF.Name = data.Name
	}

	// todo
	if song.PDF.Key != data.Key {
		//_, err := h.DriveFileService.ReplaceAllTextByRegex(song.DriveFileID, keyRegex, fmt.Sprintf("KEY: %s;", data.Key))
		//if err != nil {
		//	return
		//}

		section, err := strconv.Atoi(data.TransposeSection)
		if err != nil {
			return
		}
		_, err = h.DriveFileService.TransposeOne(song.DriveFileID, data.Key, section)

		//song.PDF.Key = data.Key
	}

	if song.PDF.BPM != data.BPM {
		_, err := h.DriveFileService.ReplaceAllTextByRegex(song.DriveFileID, bpmRegex, fmt.Sprintf("BPM: %s;", data.BPM))
		if err != nil {
			return
		}
		song.PDF.BPM = data.BPM
	}

	if song.PDF.Time != data.Time {
		_, err := h.DriveFileService.ReplaceAllTextByRegex(song.DriveFileID, timeRegex, fmt.Sprintf("TIME: %s;", data.Time))
		if err != nil {
			return
		}
		song.PDF.Time = data.Time
	}

	fakeTime, _ := time.Parse("2006", "2006")
	song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

	song, err = h.SongService.UpdateOne(*song)
	if err != nil {
		return
	}

	user, err := h.UserService.FindOneByID(userID)
	if err != nil {
		return
	}

	user.CallbackCache = entity.CallbackCache{
		ChatID:    chatID,
		MessageID: messageID,
		UserID:    userID,
	}
	caption := user.CallbackCache.AddToText(song.Caption())

	reader, err := h.DriveFileService.DownloadOneByID(song.DriveFileID)
	if err != nil {
		return
	}

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard.SongEdit(song, user, chatID, messageID, "ru"),
	}

	_, _, err = h.Bot.EditMessageMedia(gotgbot.InputMediaDocument{
		Media: gotgbot.NamedFile{
			File:     *reader,
			FileName: fmt.Sprintf("%s.pdf", song.PDF.Name),
		},
		Caption:   caption,
		ParseMode: "HTML",
	}, &gotgbot.EditMessageMediaOpts{
		ChatId:      chatID,
		MessageId:   messageID,
		ReplyMarkup: markup,
	})

	ctx.Status(http.StatusOK)
}

var keys = []string{"C", "C#", "Db", "D", "D#", "Eb", "E", "F", "F#", "Gb", "G", "G#", "Ab", "A", "A#", "Bb", "B"}
var times = []string{"4/4", "3/4", "6/8", "2/2"}
var bpms []string

func init() {
	for i := 60; i < 180; i++ {
		bpms = append(bpms, strconv.Itoa(i))
	}
}

type SelectEntity struct {
	Name       string
	Value      string
	IsSelected bool
}

func valuesForSelect(songVal string, values []string, name string) []*SelectEntity {
	keysForSelect := []*SelectEntity{
		{
			Name:       name,
			Value:      "?",
			IsSelected: false,
		},
	}

	somethingWasSelected := false
	for _, key := range values {
		if key == songVal {
			somethingWasSelected = true
		}
		keysForSelect = append(keysForSelect, &SelectEntity{Name: key, Value: key, IsSelected: key == songVal})
	}

	if !somethingWasSelected && songVal == "" || songVal == "?" {
		keysForSelect[0].IsSelected = true
	} else if !somethingWasSelected {
		keysForSelect = append(keysForSelect, &SelectEntity{Name: songVal, Value: songVal, IsSelected: true})
	}

	return keysForSelect
}
