package controller

import (
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/entities"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/drive/v3"
	"regexp"
	"strings"
)

func (c *BotController) song(bot *gotgbot.Bot, ctx *ext.Context, driveFileID string) error {

	user := ctx.Data["user"].(*entities.User)

	ctx.EffectiveChat.SendAction(bot, "upload_document")

	song, driveFile, err := c.SongService.FindOrCreateOneByDriveFileID(driveFileID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: helpers.GetSongInitKeyboard(user, song),
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

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.GetSongs {
			user.State = entities.State{
				Index: index,
				Name:  state.GetSongs,
			}
			user.Cache = entities.Cache{}
		}

		switch index {
		case 0:
			{
				// todo
				if ctx.EffectiveMessage.Text == txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode) {
					user.State = entities.State{
						Name: helpers.CreateSongState,
					}
					return c.OldHandler.Enter(ctx, user)
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				if ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode) && user.Cache.NextPageToken.GetPrevValue() != "" {
					user.Cache.NextPageToken = user.Cache.NextPageToken.Prev.Prev
				}

				driveFiles, nextPageToken, err := c.DriveFileService.FindAllByFolderID(user.Band.DriveFolderID, user.Cache.NextPageToken.GetValue())
				if err != nil {
					return err
				}

				user.Cache.NextPageToken = &entities.NextPageToken{
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
					InputFieldPlaceholder: "Фраза из песни или список",
				}

				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)}},
				}
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode)}})

				likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

				for _, driveFile := range driveFiles {
					driveFileName := driveFile.Name

					if likedSongErr == nil {
						for _, likedSong := range likedSongs {
							if likedSong.DriveFileID == driveFile.Id {
								driveFileName += " " + txt.Get("button.like", ctx.EffectiveUser.LanguageCode)
							}
						}
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFileName}})
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
					user.State = entities.State{
						Name: helpers.CreateSongState,
					}
					return c.OldHandler.Enter(ctx, user)

				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode), txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					return c.GetSongs(0)(bot, ctx)

				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					return c.filterSongs(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				driveFiles := user.Cache.DriveFiles
				var foundDriveFile *drive.File
				for _, driveFile := range driveFiles {
					if driveFile.Name == strings.ReplaceAll(ctx.EffectiveMessage.Text, " "+txt.Get("button.like", ctx.EffectiveUser.LanguageCode), "") {
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

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.FilterSongs {
			user.State = entities.State{
				Index: index,
				Name:  state.FilterSongs,
			}
			user.Cache = entities.Cache{}
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
					songs []*entities.SongExtra
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
					if strings.Contains(ctx.EffectiveMessage.Text, "〔") {
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
					InputFieldPlaceholder: "Фраза из песни или список",
				}
				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{
						{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)},
					},
				}

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.Cache.Filter {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
						break
					}
				}

				for _, songExtra := range songs {
					buttonText := songExtra.Song.PDF.Name
					if songExtra.Caption() != "" {
						buttonText += fmt.Sprintf(" (%s)", songExtra.Caption())
					}

					if user.Cache.Filter != txt.Get("button.like", ctx.EffectiveUser.LanguageCode) {
						for _, userID := range songExtra.Song.Likes {
							if user.ID == userID {
								buttonText += " " + txt.Get("button.like", ctx.EffectiveUser.LanguageCode)
								break
							}
						}
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
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

				if strings.Contains(ctx.EffectiveMessage.Text, "〔") && strings.Contains(ctx.EffectiveMessage.Text, "〕") {
					return c.GetSongs(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				var songName string
				regex := regexp.MustCompile(`\s*\(.*\)\s*(` + txt.Get("button.like", ctx.EffectiveUser.LanguageCode) + `)?\s*`)
				songName = regex.ReplaceAllString(ctx.EffectiveMessage.Text, "")

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
				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{
						{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)},
					},
				}

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.Cache.Filter {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
						break
					}
				}

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
