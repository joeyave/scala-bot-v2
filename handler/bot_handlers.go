package handler

import (
	"encoding/json"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"scala-bot-v2/dto"
	"scala-bot-v2/entity"
	"scala-bot-v2/service"
	"strings"
	"sync"
	"time"
)

type BotHandler struct {
	UserService       *service.UserService
	DriveFileService  *service.DriveFileService
	SongService       *service.SongService
	VoiceService      *service.VoiceService
	BandService       *service.BandService
	MembershipService *service.MembershipService
	EventService      *service.EventService
	RoleService       *service.RoleService
}

func (h *BotHandler) Menu(b *gotgbot.Bot, ctx *ext.Context) error {

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

func (h *BotHandler) CreateEvent(b *gotgbot.Bot, ctx *ext.Context) error {

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

	event := entity.Event{
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

func (h *BotHandler) EventChords(b *gotgbot.Bot, ctx *ext.Context) error {

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
func (h *BotHandler) Events(b *gotgbot.Bot, ctx *ext.Context) error {

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

func (h *BotHandler) eventToHTML(event *entity.Event) string {
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

func (h *BotHandler) sendDriveFilesAlbum(bot *gotgbot.Bot, ctx *ext.Context, driveFileIDs []string) error {

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

		//// TODO: check for bugs.
		//if err != nil {
		//	fromIndex := 0
		//	toIndex := 0 + len(album)
		//
		//	if i-1 > 0 && i-1 < len(chunks) {
		//		fromIndex = i * len(chunks[i-1])
		//		toIndex = fromIndex + len(chunks[i])
		//	}
		//
		//	foundDriveFileIDs := driveFileIDs[fromIndex:toIndex]
		//
		//	var waitGroup sync.WaitGroup
		//	waitGroup.Add(len(foundDriveFileIDs))
		//	bigAlbum := make(telebot.Album, len(foundDriveFileIDs))
		//
		//	for i := range foundDriveFileIDs {
		//		go func(i int) {
		//			defer waitGroup.Done()
		//			reader, err := h.DriveFileService.DownloadOneByID(foundDriveFileIDs[i])
		//			if err != nil {
		//				return
		//			}
		//
		//			// driveFile, err := h.driveFileService.FindOneByID(foundDriveFileIDs[i])
		//			// if err != nil {
		//			// 	return
		//			// }
		//
		//			song, driveFile, err := h.SongService.FindOrCreateOneByDriveFileID(driveFileIDs[i])
		//			if err != nil {
		//				return
		//			}
		//
		//			bigAlbum[i] = &telebot.Document{
		//				File:     telebot.FromReader(*reader),
		//				MIME:     "application/pdf",
		//				FileName: fmt.Sprintf("%s.pdf", driveFile.Name),
		//				Caption:  song.Caption(),
		//			}
		//		}(i)
		//	}
		//	waitGroup.Wait()
		//
		//	_, err = h.bot.SendAlbum(c.Recipient(), bigAlbum)
		//	if err != nil {
		//		continue
		//	}
		//}

		// for j := range responses {
		//	foundDriveFileID := driveFileIDs[j+(i*len(album))]
		//
		//	song, err := h.songService.FindOneByDriveFileID(foundDriveFileID)
		//	if err != nil {
		//		continue
		//	}
		//
		//	song.PDF.TgFileID = responses[j].Document.FileID
		//	err = SendSongToChannel(h, c, user, song)
		//	if err != nil {
		//		continue
		//	}
		//
		//	_, _ = h.songService.UpdateOne(*song)
		// }
	}

	return nil
}

func chunkAlbumBy(items []gotgbot.InputMedia, chunkSize int) (chunks [][]gotgbot.InputMedia) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}
