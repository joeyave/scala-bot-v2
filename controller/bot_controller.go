package controller

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/entities"
	myhandlers "github.com/joeyave/scala-bot-v2/handlers"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/services"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/drive/v3"
	"strings"
	"sync"
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
	OldHandler        *myhandlers.Handler
}

func (c *BotController) ChooseHandlerOrSearch(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

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

	user, err := c.UserService.FindOneOrCreateByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	user.Name = strings.TrimSpace(fmt.Sprintf("%s %s", ctx.EffectiveChat.FirstName, ctx.EffectiveChat.LastName))

	if user.BandID == primitive.NilObjectID && user.State.Name != helpers.ChooseBandState && user.State.Name != helpers.CreateBandState {
		user.State = entities.State{
			Name: helpers.ChooseBandState,
		}
	}

	ctx.Data["user"] = user

	return nil
}

func (c *BotController) UpdateUser(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	_, err := c.UserService.UpdateOne(*user)
	return err
}

func (c *BotController) search(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.Search {
			user.State = entities.State{
				Index: index,
				Name:  state.Search,
			}
			user.Cache = entities.Cache{}
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

				query = helpers.CleanUpQuery(query)
				songNames := helpers.SplitQueryByNewlines(query)

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

				//if user.Cache.nextPageToken == nil {
				//	user.Cache.nextPageToken = &entities.nextPageToken{}
				//}

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

				set := make(map[string]*entities.Band)
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

				if ctx.EffectiveMessage.Text != txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode) {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode)}})
				}

				markup.Keyboard = append(markup.Keyboard, keyboard.Navigation(user.Cache.NextPageToken, ctx.EffectiveUser.LanguageCode)...)

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

func (c *BotController) searchSetlist(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.SearchSetlist {
			user.State = entities.State{
				Index: index,
				Name:  state.SearchSetlist,
			}
			user.Cache = entities.Cache{
				SongNames: user.Cache.SongNames,
			}
		}

		switch index {
		case 0:
			{
				if len(user.Cache.SongNames) < 1 {
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
						Keyboard:       helpers.CancelOrSkipKeyboard,
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
				case helpers.Skip:
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

func (c *BotController) Menu(b *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	user.State = entities.State{}

	replyMarkup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       keyboard.Menu(ctx.EffectiveUser.LanguageCode),
		ResizeKeyboard: true,
	}

	_, err := ctx.EffectiveChat.SendMessage(b, txt.Get("text.menu", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return err
	}

	return nil
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

	chunks := chunkAlbumBy(bigAlbum, 10)

	for _, album := range chunks {
		_, err := bot.SendMediaGroup(ctx.EffectiveChat.Id, album, nil)
		if err != nil { // todo: try to download all
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
