package controller

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/service"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/joeyave/scala-bot-v2/util"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/drive/v3"
	"strings"
	"sync"
)

type BotController struct {
	UserService       *service.UserService
	DriveFileService  *service.DriveFileService
	SongService       *service.SongService
	VoiceService      *service.VoiceService
	BandService       *service.BandService
	MembershipService *service.MembershipService
	EventService      *service.EventService
	RoleService       *service.RoleService
	//OldHandler        *myhandlers.Handler
}

func (c *BotController) ChooseHandlerOrSearch(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	switch user.State.Name {
	case state.GetEvents:
		return c.GetEvents(user.State.Index)(bot, ctx)
	case state.FilterEvents:
		return c.filterEvents(user.State.Index)(bot, ctx)
	case state.GetSongs:
		return c.GetSongs(user.State.Index)(bot, ctx)
	case state.FilterSongs:
		return c.filterSongs(user.State.Index)(bot, ctx)
	case state.SearchSetlist:
		return c.searchSetlist(user.State.Index)(bot, ctx)
	}

	return c.search(user.State.Index)(bot, ctx)
}

func (c *BotController) RegisterUser(bot *gotgbot.Bot, ctx *ext.Context) error {

	user, err := c.UserService.FindOneOrCreateByID(ctx.EffectiveUser.Id)
	if err != nil {
		return err
	}

	user.Name = strings.TrimSpace(fmt.Sprintf("%s %s", ctx.EffectiveChat.FirstName, ctx.EffectiveChat.LastName))

	// todo
	//if user.BandID == primitive.NilObjectID && user.State.Name != helpers.ChooseBandState && user.State.Name != helpers.CreateBandState {
	//	user.State = entity.State{
	//		Name: helpers.ChooseBandState,
	//	}
	//}

	ctx.Data["user"] = user

	return nil
}

func (c *BotController) UpdateUser(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	_, err := c.UserService.UpdateOne(*user)
	return err
}

func (c *BotController) search(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.Search {
			user.State = entity.State{
				Index: index,
				Name:  state.Search,
			}
			user.Cache = entity.Cache{}
		}

		switch index {
		case 0:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				var query string

				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode):
					user.Cache.Filter = ctx.EffectiveMessage.Text
					query = user.Cache.Query
				case txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					query = user.Cache.Query
					user.Cache.NextPageToken = user.Cache.NextPageToken.Prev.Prev
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					query = user.Cache.Query
				default:
					query = ctx.EffectiveMessage.Text
					// Обнуляем страницы при новом запросе.
					user.Cache.NextPageToken = nil
				}

				query = util.CleanUpText(query)
				songNames := util.SplitTextByNewlines(query)

				if len(songNames) > 1 {
					user.Cache.SongNames = songNames
					return c.searchSetlist(0)(bot, ctx)

				} else if len(songNames) == 1 {
					query = songNames[0]
					user.Cache.Query = query
				} else {
					_, err := ctx.EffectiveChat.SendMessage(bot, "Из запроса удаляются все числа, дефисы и скобки вместе с тем, что в них.", nil)
					return err
				}

				var (
					driveFiles    []*drive.File
					nextPageToken string
					err           error
				)
				switch user.Cache.Filter {
				case txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode):
					driveFiles, nextPageToken, err = c.DriveFileService.FindSomeByFullTextAndFolderID(query, "", user.Cache.NextPageToken.GetValue())
				default:
					driveFiles, nextPageToken, err = c.DriveFileService.FindSomeByFullTextAndFolderID(query, user.Band.DriveFolderID, user.Cache.NextPageToken.GetValue())
				}
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
					_, err := ctx.EffectiveChat.SendMessage(bot, txt.Get("text.nothingFound", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: query,
				}
				if markup.InputFieldPlaceholder == "" {
					markup.InputFieldPlaceholder = txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode)
				}

				likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

				set := make(map[string]*entity.Band)
				for i, driveFile := range driveFiles {

					if user.Cache.Filter == txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode) {

						for _, parentFolderID := range driveFile.Parents {
							_, exists := set[parentFolderID]
							if !exists {
								band, err := c.BandService.FindOneByDriveFolderID(parentFolderID)
								if err == nil {
									set[parentFolderID] = band
									driveFiles[i].Name += fmt.Sprintf(" (%s)", band.Name)
									break
								}
							} else {
								driveFiles[i].Name += fmt.Sprintf(" (%s)", set[parentFolderID].Name)
							}
						}
					}

					opts := &keyboard.DriveFileButtonOpts{
						ShowLike: true,
					}
					if likedSongErr != nil {
						opts.ShowLike = false
					}
					markup.Keyboard = append(markup.Keyboard, keyboard.DriveFileButton(driveFile, likedSongs, opts))
				}

				if ctx.EffectiveMessage.Text != txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode) {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode)}})
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
				case txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode), txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					return c.search(0)(bot, ctx)
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

func (c *BotController) searchSetlist(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.SearchSetlist {
			user.State = entity.State{
				Index: index,
				Name:  state.SearchSetlist,
			}
			user.Cache = entity.Cache{
				SongNames: user.Cache.SongNames,
			}
		}

		switch index {
		case 0:
			{
				if len(user.Cache.SongNames) < 1 {
					ctx.EffectiveChat.SendAction(bot, "upload_document")

					err := c.songsAlbum(bot, ctx, user.Cache.DriveFileIDs)
					if err != nil {
						return err
					}
					return c.Menu(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				songNames := user.Cache.SongNames

				currentSongName := songNames[0]
				user.Cache.SongNames = songNames[1:]

				driveFiles, _, err := c.DriveFileService.FindSomeByFullTextAndFolderID(currentSongName, user.Band.DriveFolderID, "")
				if err != nil {
					return err
				}

				if len(driveFiles) == 0 {
					markup := &gotgbot.ReplyKeyboardMarkup{
						Keyboard: [][]gotgbot.KeyboardButton{
							{{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.skip", ctx.EffectiveUser.LanguageCode)}},
						},
						ResizeKeyboard: true,
					}

					_, err := ctx.EffectiveChat.SendMessage(bot, txt.Get("text.nothingFoundByQuery", ctx.EffectiveUser.LanguageCode, currentSongName), &gotgbot.SendMessageOpts{
						ReplyMarkup: markup,
					})
					if err != nil {
						return err
					}

					user.State.Index = 1
					return nil
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: currentSongName,
				}

				for _, song := range driveFiles {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: song.Name}})
				}
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{
					{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.skip", ctx.EffectiveUser.LanguageCode)},
				})

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseSongOrTypeAnotherQuery", ctx.EffectiveUser.LanguageCode, currentSongName), &gotgbot.SendMessageOpts{
					ReplyMarkup: markup,
				})
				if err != nil {
					return err
				}

				user.State.Index = 1
				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.skip", ctx.EffectiveUser.LanguageCode):
					return c.searchSetlist(0)(bot, ctx)
				}

				foundDriveFile, err := c.DriveFileService.FindOneByNameAndFolderID(ctx.EffectiveMessage.Text, user.Band.DriveFolderID)
				if err != nil {
					user.Cache.SongNames = append([]string{ctx.EffectiveMessage.Text}, user.Cache.SongNames...)
				} else {
					user.Cache.DriveFileIDs = append(user.Cache.DriveFileIDs, foundDriveFile.Id)
				}

				return c.searchSetlist(0)(bot, ctx)
			}
		}
		return nil
	}
}

func (c *BotController) Menu(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	user.State = entity.State{}

	replyMarkup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       keyboard.Menu(ctx.EffectiveUser.LanguageCode),
		ResizeKeyboard: true,
	}

	_, err := ctx.EffectiveChat.SendMessage(bot, txt.Get("text.menu", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) Error(bot *gotgbot.Bot, ctx *ext.Context, botErr error) ext.DispatcherAction {

	log.Error().Msgf("Error handling update: %v", botErr)

	if ctx.CallbackQuery != nil {
		_, err := ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: "Произошла ошибка. Поправим.",
		})
		if err != nil {
			return 0
		}
	} else {
		_, err := ctx.EffectiveChat.SendMessage(bot, "Произошла ошибка. Поправим.", nil)
		if err != nil {
			log.Error().Err(err).Msg("Error!")
			return ext.DispatcherActionEndGroups
		}
	}

	user, err := c.UserService.FindOneByID(ctx.EffectiveChat.Id)
	if err != nil {
		log.Error().Err(err).Msg("Error!")
		return ext.DispatcherActionEndGroups
	}

	// todo: send message to the logs channel

	user.State = entity.State{}
	_, err = c.UserService.UpdateOne(*user)
	if err != nil {
		log.Error().Err(err).Msg("Error!")
		return ext.DispatcherActionEndGroups
	}

	return ext.DispatcherActionEndGroups
}

func (c *BotController) songsAlbum(bot *gotgbot.Bot, ctx *ext.Context, driveFileIDs []string) error {

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

	chunks := chunkAlbum(bigAlbum, 10)

	for _, album := range chunks {
		_, err := bot.SendMediaGroup(ctx.EffectiveChat.Id, album, nil)
		if err != nil { // todo: try to download all
			return err
		}
	}

	return nil
}

func chunkAlbum(items []gotgbot.InputMedia, chunkSize int) (chunks [][]gotgbot.InputMedia) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}
