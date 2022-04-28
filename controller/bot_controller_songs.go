package controller

import (
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/joeyave/scala-bot-v2/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/drive/v3"
	"strings"
)

func (c *BotController) song(bot *gotgbot.Bot, ctx *ext.Context, driveFileID string) error {

	user := ctx.Data["user"].(*entity.User)

	ctx.EffectiveChat.SendAction(bot, "upload_document")

	song, driveFile, err := c.SongService.FindOrCreateOneByDriveFileID(driveFileID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard.SongInit(song, user, ctx.EffectiveUser.LanguageCode),
	}

	sendDocumentByReader := func() (*gotgbot.Message, error) {
		reader, err := c.DriveFileService.DownloadOneByID(driveFile.Id)
		if err != nil {
			return nil, err
		}

		message, err := bot.SendDocument(ctx.EffectiveChat.Id, gotgbot.NamedFile{
			File:     *reader,
			FileName: fmt.Sprintf("%s.pdf", driveFile.Name),
		}, &gotgbot.SendDocumentOpts{
			Caption:     song.Caption() + "\n" + strings.Join(song.Tags, ", "),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		return message, err
	}

	sendDocumentByFileID := func() (*gotgbot.Message, error) {
		message, err := bot.SendDocument(ctx.EffectiveChat.Id, song.PDF.TgFileID, &gotgbot.SendDocumentOpts{
			Caption:     song.Caption() + "\n" + strings.Join(song.Tags, ", "),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		return message, err
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

	// todo
	//err = SendSongToChannel(h, c, user, song)
	//if err != nil {
	//	return err
	//}

	song, err = c.SongService.UpdateOne(*song)
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) GetSongs(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.GetSongs {
			user.State = entity.State{
				Index: index,
				Name:  state.GetSongs,
			}
			user.Cache = entity.Cache{}
		}

		switch index {
		case 0:
			{
				// todo
				if ctx.EffectiveMessage.Text == txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode) {
					user.State = entity.State{
						Name: helpers.CreateSongState,
					}
					return nil
					//return c.OldHandler.Enter(ctx, user)
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				if ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode) && user.Cache.NextPageToken.GetPrevValue() != "" {
					user.Cache.NextPageToken = user.Cache.NextPageToken.Prev.Prev
				}

				driveFiles, nextPageToken, err := c.DriveFileService.FindAllByFolderID(user.Band.DriveFolderID, user.Cache.NextPageToken.GetValue())
				if err != nil {
					return err
				}

				user.Cache.NextPageToken = &entity.NextPageToken{
					Value: nextPageToken,
					Prev:  user.Cache.NextPageToken,
				}

				if len(driveFiles) == 0 {
					markup := &gotgbot.ReplyKeyboardMarkup{
						Keyboard: [][]gotgbot.KeyboardButton{
							{{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode)}},
						},
						ResizeKeyboard: true,
					}
					_, err := ctx.EffectiveChat.SendMessage(bot, "В папке на Google Диске нет документов.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}

				markup.Keyboard = append(markup.Keyboard, keyboard.GetSongsStateFilterButtons(ctx.EffectiveUser.LanguageCode))
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode)}})

				likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

				for _, driveFile := range driveFiles {
					opts := &keyboard.DriveFileButtonOpts{
						ShowLike: true,
					}
					if likedSongErr != nil {
						opts.ShowLike = false
					}
					markup.Keyboard = append(markup.Keyboard, keyboard.DriveFileButton(driveFile, likedSongs, opts))
				}

				markup.Keyboard = append(markup.Keyboard, keyboard.NavigationByToken(user.Cache.NextPageToken, ctx.EffectiveUser.LanguageCode)...)

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseSong", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.Cache.DriveFiles = driveFiles

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {

				// todo
				case txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode):
					user.State = entity.State{
						Name: helpers.CreateSongState,
					}
					return nil
					//return c.OldHandler.Enter(ctx, user)

				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode), txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					return c.GetSongs(0)(bot, ctx)

				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					return c.filterSongs(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				driveFileName := keyboard.ParseDriveFileButton(ctx.EffectiveMessage.Text)

				driveFiles := user.Cache.DriveFiles
				var foundDriveFile *drive.File
				for _, driveFile := range driveFiles {
					if driveFile.Name == driveFileName {
						foundDriveFile = driveFile
						break
					}
				}

				if foundDriveFile != nil {
					return c.song(bot, ctx, foundDriveFile.Id)
				} else {
					return c.search(0)(bot, ctx)
				}
			}
		}
		return nil
	}
}

func (c *BotController) filterSongs(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.FilterSongs {
			user.State = entity.State{
				Index: index,
				Name:  state.FilterSongs,
			}
			user.Cache = entity.Cache{}
		}

		switch index {
		case 0:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode):
					user.Cache.Filter = ctx.EffectiveMessage.Text

				case txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					user.Cache.Filter = ctx.EffectiveMessage.Text
					return c.filterSongs(2)(bot, ctx)
				}

				var (
					songs []*entity.SongWithEvents
					err   error
				)

				switch user.Cache.Filter {
				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode):
					songs, err = c.SongService.FindManyExtraLiked(user.ID, user.Cache.PageIndex)
				case txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode):
					songs, err = c.SongService.FindAllExtraByPageNumberSortedByLatestEventDate(user.BandID, user.Cache.PageIndex)
				case txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode):
					songs, err = c.SongService.FindAllExtraByPageNumberSortedByEventsNumber(user.BandID, user.Cache.PageIndex)
				case txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					if keyboard.IsSelectedButton(ctx.EffectiveMessage.Text) {
						return c.GetSongs(0)(bot, ctx)
					}
					if user.Cache.Query == "" {
						user.Cache.Query = ctx.EffectiveMessage.Text
					}
					songs, err = c.SongService.FindManyExtraByTag(user.Cache.Query, user.BandID, user.Cache.PageIndex)
				}
				if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}

				filterButtons := keyboard.GetSongsStateFilterButtons(ctx.EffectiveUser.LanguageCode)
				for i := range filterButtons {
					if filterButtons[i].Text == user.Cache.Filter {
						filterButtons[i] = keyboard.SelectedButton(filterButtons[i].Text)
						break
					}
				}
				markup.Keyboard = append(markup.Keyboard, filterButtons)

				for _, song := range songs {

					songButtonOpts := &keyboard.SongButtonOpts{
						ShowLike:  false,
						ShowStats: true,
					}

					if user.Cache.Filter != txt.Get("button.like", ctx.EffectiveUser.LanguageCode) {
						songButtonOpts.ShowLike = true
					}

					markup.Keyboard = append(markup.Keyboard, keyboard.SongButton(song, user, ctx.EffectiveUser.LanguageCode, songButtonOpts))
				}

				if user.Cache.PageIndex != 0 {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.next", ctx.EffectiveUser.LanguageCode)}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.next", ctx.EffectiveUser.LanguageCode)}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseSong", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex = 0
					return c.filterSongs(0)(bot, ctx)
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex++
					return c.filterSongs(0)(bot, ctx)
				case txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex--
					return c.filterSongs(0)(bot, ctx)
				}

				if keyboard.IsSelectedButton(ctx.EffectiveMessage.Text) {
					return c.GetSongs(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				songName := keyboard.ParseSongButton(ctx.EffectiveMessage.Text)

				song, err := c.SongService.FindOneByName(strings.TrimSpace(songName))
				if err != nil {
					return c.search(0)(bot, ctx)
				}

				return c.song(bot, ctx, song.DriveFileID)
			}
		case 2:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				tags, err := c.SongService.GetTags()
				if err != nil {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}

				filterButtons := keyboard.GetSongsStateFilterButtons(ctx.EffectiveUser.LanguageCode)
				for i := range filterButtons {
					if filterButtons[i].Text == user.Cache.Filter {
						filterButtons[i] = keyboard.SelectedButton(filterButtons[i].Text)
						break
					}
				}
				markup.Keyboard = append(markup.Keyboard, filterButtons)

				for _, tag := range tags {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: tag}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseTag", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.State.Index = 0
				return nil
			}
		}

		return nil
	}
}

func (c *BotController) SongCB(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	payload := util.ParseCallbackPayload(ctx.CallbackQuery.Data)
	split := strings.Split(payload, ":")

	hex := split[0]
	songID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	song, err := c.SongService.FindOneByID(songID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{}

	if len(split) > 1 {
		switch split[1] {
		case "edit":
			markup.InlineKeyboard = keyboard.SongEdit(song, user, ctx.EffectiveUser.LanguageCode)
		default:
			markup.InlineKeyboard = keyboard.SongInit(song, user, ctx.EffectiveUser.LanguageCode)
		}
	}

	_, _, err = ctx.EffectiveMessage.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: markup,
	})

	ctx.CallbackQuery.Answer(bot, nil)

	return err
}

func (c BotController) SongLike(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	payload := util.ParseCallbackPayload(ctx.CallbackQuery.Data)
	split := strings.Split(payload, ":")

	songID, err := primitive.ObjectIDFromHex(split[0])
	if err != nil {
		return err
	}

	switch split[1] {
	case "like":
		err := c.SongService.Like(songID, user.ID)
		if err != nil {
			return err
		}
	case "dislike":
		err := c.SongService.Dislike(songID, user.ID)
		if err != nil {
			return err
		}
	}

	song, err := c.SongService.FindOneByID(songID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{}
	markup.InlineKeyboard = keyboard.SongInit(song, user, ctx.EffectiveUser.LanguageCode)

	_, _, err = ctx.EffectiveMessage.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: markup,
	})
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)
	return nil
}
