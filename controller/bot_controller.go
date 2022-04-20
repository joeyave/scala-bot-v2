package controller

import (
	"encoding/json"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/dto"
	"github.com/joeyave/scala-bot-v2/entities"
	"github.com/joeyave/scala-bot-v2/helpers"
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

func (c *BotController) Event(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := c.UserService.FindOneByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	event := ctx.Data["event"].(*entities.Event)

	html := c.EventService.ToHtmlStringByEvent(*event)

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: helpers.GetEventActionsKeyboard(*user, *event),
	}

	if ctx.CallbackQuery != nil {
		_, _, err := ctx.EffectiveMessage.EditText(b, html, &gotgbot.EditMessageTextOpts{
			ReplyMarkup:           markup,
			DisableWebPagePreview: true,
			ParseMode:             "HTML",
		})
		if err != nil {
			return err
		}
		ctx.CallbackQuery.Answer(b, nil)
		return nil
	} else {
		_, err := ctx.EffectiveChat.SendMessage(b, html, &gotgbot.SendMessageOpts{
			ReplyMarkup:           markup,
			DisableWebPagePreview: true,
			ParseMode:             "HTML",
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *BotController) CreateEvent(b *gotgbot.Bot, ctx *ext.Context) error {

	var data *dto.CreateEventData
	err := json.Unmarshal([]byte(ctx.EffectiveMessage.WebAppData.Data), &data)
	if err != nil {
		return err
	}

	user, err := c.UserService.FindOneByID(ctx.EffectiveUser.Id)
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
	createdEvent, err := c.EventService.UpdateOne(event)
	if err != nil {
		return err
	}

	ctx.Data["event"] = createdEvent
	return c.Event(b, ctx)
}

// work in progress -----------------------------

func (c *BotController) EventChords(b *gotgbot.Bot, ctx *ext.Context) error {

	hex := strings.TrimPrefix(ctx.CallbackQuery.Data, "eventChords:")
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	event, err := c.EventService.FindOneByID(eventID)
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

	err = c.sendDriveFilesAlbum(b, ctx, driveFileIDs)
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
func (c *BotController) Events(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := c.UserService.FindOneByID(ctx.EffectiveUser.Id)
	if err != nil {
		return err
	}

	_, err = c.EventService.FindManyFromTodayByBandID(user.BandID)
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) sendDriveFilesAlbum(bot *gotgbot.Bot, ctx *ext.Context, driveFileIDs []string) error {

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(driveFileIDs))
	bigAlbum := make([]gotgbot.InputMedia, len(driveFileIDs))

	for i := range driveFileIDs {
		go func(i int) {
			defer waitGroup.Done()

			song, driveFile, err := c.SongService.FindOrCreateOneByDriveFileID(driveFileIDs[i])
			if err != nil {
				return
			}

			if song.PDF.TgFileID == "" {
				reader, err := c.DriveFileService.DownloadOneByID(driveFile.Id)
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

func (c *BotController) Menu(b *gotgbot.Bot, ctx *ext.Context) error {

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
