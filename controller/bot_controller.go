package controller

import (
	"encoding/json"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/dto"
	"github.com/joeyave/scala-bot-v2/entities"
	"github.com/joeyave/scala-bot-v2/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"strings"
	"sync"
	"time"
)

type BotController struct {
	UserService       *services.UserService
	DriveFileService  *services.DriveFileService
	SongService       *services.SongService
	VoiceService      *services.VoiceService
	BandService       *services.BandService
	MembershipService *services.MembershipService
	EventService      *services.EventService
	RoleService       *services.RoleService
}

func (h *BotController) Menu(b *gotgbot.Bot, ctx *ext.Context) error {

	replyMarkup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard: [][]gotgbot.KeyboardButton{
			{
				{Text: "➕ Добавить собрание", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/create-event"}},
			},
		},
		ResizeKeyboard: true,
	}

	_, err := ctx.EffectiveChat.SendMessage(b, "Меню:", &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *BotController) CreateEvent(b *gotgbot.Bot, ctx *ext.Context) error {

	var data *dto.CreateEventData
	err := json.Unmarshal([]byte(ctx.EffectiveMessage.WebAppData.Data), &data)
	if err != nil {
		return err
	}

	user, err := h.UserService.FindOneByID(ctx.EffectiveUser.Id)
	if err != nil {
		return err
	}

	eventDate, err := time.Parse("2006-01-02", data.Event.Date)
	if err != nil {
		return err
	}

	event := entities.Event{
		Time:   eventDate,
		Name:   data.Event.Name,
		BandID: user.BandID,
	}
	createdEvent, err := h.EventService.UpdateOne(event)
	if err != nil {
		return err
	}

	eventHTML := h.eventToHTML(createdEvent)

	replyMarkup := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "🎶 Аккорды", CallbackData: fmt.Sprintf("eventChords:%s", createdEvent.ID.Hex())},
				{Text: "︎✍️ Редактировать", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/edit-event/" + createdEvent.ID.Hex()}},
			},
		},
	}

	_, err = ctx.EffectiveChat.SendMessage(b, eventHTML, &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *BotController) EventChords(b *gotgbot.Bot, ctx *ext.Context) error {

	hex := strings.TrimPrefix(ctx.CallbackQuery.Data, "eventChords:")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	event, err := h.EventService.FindOneByID(eventID)
	if err != nil {
		return err
	}

	var driveFileIDs []string
	for _, song := range event.Songs {
		driveFileIDs = append(driveFileIDs, song.DriveFileID)
	}

	if len(driveFileIDs) == 0 {
		_, err = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text: "В списке нет песен.",
		})
		if err != nil {
			return err
		}
		return nil
	}

	err = h.sendDriveFilesAlbum(b, ctx, driveFileIDs)
	if err != nil {
		return err
	}

	_, err = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "PDF файлы готовы!",
	})
	if err != nil {
		return err
	}

	return nil
}

// todo
func (h *BotController) Events(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := h.UserService.FindOneByID(ctx.EffectiveUser.Id)
	if err != nil {
		return err
	}

	_, err = h.EventService.FindManyFromTodayByBandID(user.BandID)
	if err != nil {
		return err
	}

	return nil
}

func (h *BotController) eventToHTML(event *entities.Event) string {
	eventString := fmt.Sprintf("<b>%s</b>", event.Alias())

	var currRoleID primitive.ObjectID
	for _, membership := range event.Memberships {
		if membership.User == nil {
			continue
		}

		if currRoleID != membership.RoleID {
			currRoleID = membership.RoleID
			eventString = fmt.Sprintf("%s\n\n<b>%s:</b>", eventString, membership.Role.Name)
		}

		eventString = fmt.Sprintf("%s\n - <a href=\"tg://user?id=%d\">%s</a>", eventString, membership.User.ID, membership.User.Name)
	}

	if len(event.Songs) > 0 {
		eventString = fmt.Sprintf("%s\n\n<b>📝 Список:</b>", eventString)

		var waitGroup sync.WaitGroup
		waitGroup.Add(len(event.Songs))
		songNames := make([]string, len(event.Songs))
		for i := range event.Songs {
			go func(i int) {
				defer waitGroup.Done()

				driveFile, err := h.DriveFileService.FindOneByID(event.Songs[i].DriveFileID)
				if err != nil {
					return
				}

				songName := fmt.Sprintf("%d. <a href=\"%s\">%s</a>  (%s)",
					i+1, driveFile.WebViewLink, driveFile.Name, event.Songs[i].Caption())
				songNames[i] = songName
			}(i)
		}
		waitGroup.Wait()

		eventString += "\n" + strings.Join(songNames, "\n")
	}

	if event.Notes != "" {
		eventString += "\n\n<b>✏️ Заметки:</b>\n" + event.Notes
	}

	return eventString
}

func (h *BotController) sendDriveFilesAlbum(bot *gotgbot.Bot, ctx *ext.Context, driveFileIDs []string) error {

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(driveFileIDs))
	bigAlbum := make([]gotgbot.InputMedia, len(driveFileIDs))

	for i := range driveFileIDs {
		go func(i int) {
			defer waitGroup.Done()

			song, driveFile, err := h.SongService.FindOrCreateOneByDriveFileID(driveFileIDs[i])
			if err != nil {
				return
			}

			if song.PDF.TgFileID == "" {
				reader, err := h.DriveFileService.DownloadOneByID(driveFile.Id)
				if err != nil {
					return
				}

				bigAlbum[i] = gotgbot.InputMediaDocument{
					Media:   gotgbot.NamedFile{File: *reader, FileName: fmt.Sprintf("%s.pdf", song.PDF.Name)},
					Caption: song.Caption(),
				}
			} else {
				bigAlbum[i] = gotgbot.InputMediaDocument{
					Media:   song.PDF.TgFileID,
					Caption: song.Caption(),
				}
			}
		}(i)
	}
	waitGroup.Wait()

	const chunkSize = 10
	chunks := chunkAlbumBy(bigAlbum, chunkSize)

	for _, album := range chunks {
		_, err := bot.SendMediaGroup(ctx.EffectiveChat.Id, album, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func chunkAlbumBy(items []gotgbot.InputMedia, chunkSize int) (chunks [][]gotgbot.InputMedia) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}
