package handlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/helpers"
	"strings"
	"sync"
)

func SendDriveFileToUser(h *Handler, c *ext.Context, user *entity.User, driveFileID string) error {

	q := user.State.CallbackData.Query()
	q.Set("driveFileId", driveFileID)
	user.State.CallbackData.RawQuery = q.Encode()

	song, driveFile, err := h.songService.FindOrCreateOneByDriveFileID(driveFileID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: helpers.GetSongInitKeyboard(user, song),
	}

	sendDocumentByReader := func() (*gotgbot.Message, error) {
		reader, err := h.driveFileService.DownloadOneByID(driveFile.Id)
		if err != nil {
			return nil, err
		}

		if c.CallbackQuery != nil {

			message, _, err := h.bot.EditMessageMedia(gotgbot.InputMediaDocument{
				Media:     gotgbot.NamedFile{File: *reader, FileName: fmt.Sprintf("%s.pdf", driveFile.Name)},
				Caption:   helpers.AddCallbackData(song.Caption()+"\n"+strings.Join(song.Tags, ", "), user.State.CallbackData.String()),
				ParseMode: "HTML",
			}, &gotgbot.EditMessageMediaOpts{
				ChatId:      c.CallbackQuery.Message.Chat.Id,
				MessageId:   c.CallbackQuery.Message.MessageId,
				ReplyMarkup: markup,
			})
			return message, err

		} else {

			message, err := h.bot.SendDocument(c.EffectiveChat.Id, gotgbot.NamedFile{
				File:     *reader,
				FileName: fmt.Sprintf("%s.pdf", driveFile.Name),
			}, &gotgbot.SendDocumentOpts{
				Caption:     helpers.AddCallbackData(song.Caption()+"\n"+strings.Join(song.Tags, ", "), user.State.CallbackData.String()),
				ParseMode:   "HTML",
				ReplyMarkup: markup,
			})
			return message, err
		}
	}

	sendDocumentByFileID := func() (*gotgbot.Message, error) {
		if c.CallbackQuery != nil {
			message, _, err := h.bot.EditMessageMedia(gotgbot.InputMediaDocument{
				Media:     song.PDF.TgFileID,
				Caption:   helpers.AddCallbackData(song.Caption()+"\n"+strings.Join(song.Tags, ", "), user.State.CallbackData.String()),
				ParseMode: "HTML",
			}, &gotgbot.EditMessageMediaOpts{
				ChatId:      c.CallbackQuery.Message.Chat.Id,
				MessageId:   c.CallbackQuery.Message.MessageId,
				ReplyMarkup: markup,
			})
			return message, err
		} else {

			message, err := h.bot.SendDocument(c.EffectiveChat.Id, song.PDF.TgFileID, &gotgbot.SendDocumentOpts{
				Caption:     helpers.AddCallbackData(song.Caption()+"\n"+strings.Join(song.Tags, ", "), user.State.CallbackData.String()),
				ParseMode:   "HTML",
				ReplyMarkup: markup,
			})
			return message, err
		}
	}

	var msg *gotgbot.Message
	if song.PDF.TgFileID == "" {
		msg, err = sendDocumentByReader()
	} else {
		msg, err = sendDocumentByFileID()
		if err != nil {
			msg, err = sendDocumentByReader()
		}
	}
	if err != nil {
		return err
	}

	song.PDF.TgFileID = msg.Document.FileId
	//err = SendSongToChannel(h, c, user, song)
	//if err != nil {
	//	return err
	//}

	song, err = h.songService.UpdateOne(*song)

	return err
}

//func SendSongToChannel(h *Handler, c *ext.Context, user *entities.User, song *entities.Song) error {
//	send := func() (*telebot.Message, error) {
//		return h.bot.Send(
//			telebot.ChatID(helpers.FilesChannelID),
//			&telebot.Document{
//				File: telebot.File{FileID: song.PDF.TgFileID},
//			},
//			telebot.Silent)
//	}
//
//	edit := func() (*telebot.Message, error) {
//		return h.bot.EditMedia(
//			&telebot.Message{
//				ID:   song.PDF.TgChannelMessageID,
//				Chat: &telebot.Chat{ID: helpers.FilesChannelID},
//			}, &telebot.Document{
//				File: telebot.File{FileID: song.PDF.TgFileID},
//				MIME: "application/pdf",
//			},
//		)
//	}
//
//	var msg *telebot.Message
//	var err error
//	if song.PDF.TgChannelMessageID == 0 {
//		msg, err = send()
//		if err != nil {
//			return err
//		}
//		song.PDF.TgChannelMessageID = msg.ID
//	} else {
//		msg, err = edit()
//		if err != nil {
//			if fmt.Sprint(err) == "telegram unknown: Bad Request: MESSAGE_ID_INVALID (400)" {
//				msg, err = send()
//				if err != nil {
//					return err
//				}
//				song.PDF.TgChannelMessageID = msg.ID
//			}
//		}
//	}
//
//	return nil
//}

func sendDriveFilesAlbum(h *Handler, ctx *ext.Context, user *entity.User, driveFileIDs []string) error {

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(driveFileIDs))
	bigAlbum := make([]gotgbot.InputMedia, len(driveFileIDs))

	for i := range driveFileIDs {
		go func(i int) {
			defer waitGroup.Done()

			song, driveFile, err := h.songService.FindOrCreateOneByDriveFileID(driveFileIDs[i])
			if err != nil {
				return
			}

			if song.PDF.TgFileID == "" {
				reader, err := h.driveFileService.DownloadOneByID(driveFile.Id)
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
		_, err := h.bot.SendMediaGroup(ctx.EffectiveChat.Id, album, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
