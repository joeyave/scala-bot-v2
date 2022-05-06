package handlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/txt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/drive/v3"
	"regexp"
	"strconv"
	"time"
)

func createRoleHandler() (int, []HandlerFunc) {

	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь название новой роли. Например, лид-вокал, проповедник и т. д.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.Role = &entity.Role{
			Name: c.EffectiveMessage.Text,
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		if len(user.Band.Roles) == 0 {
			user.State.Context.Role.Priority = 1
			user.State.Index++
			return h.Enter(c, user)
		}

		for _, role := range user.Band.Roles {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: role.Name}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err := c.EffectiveChat.SendMessage(h.bot, "После какой роли должна быть эта роль?", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		if user.State.Context.Role.Priority == 0 {

			var foundRole *entity.Role
			for _, role := range user.Band.Roles {
				if c.EffectiveMessage.Text == role.Name {
					foundRole = role
					break
				}
			}

			if foundRole == nil {
				user.State.Index--
				return h.Enter(c, user)
			}

			user.State.Context.Role.Priority = foundRole.Priority + 1

			for _, role := range user.Band.Roles {
				if role.Priority > foundRole.Priority {
					role.Priority++
					h.roleService.UpdateOne(*role)
				}
			}
		}

		role, err := h.roleService.UpdateOne(
			entity.Role{
				Name:     user.State.Context.Role.Name,
				BandID:   user.BandID,
				Priority: user.State.Context.Role.Priority,
			})
		if err != nil {
			return err
		}

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Добавлена новая роль: %s.", role.Name), nil)
		if err != nil {
			return err
		}

		user.State = entity.State{Name: helpers.MainMenuState}
		return h.Enter(c, user)
	})

	return helpers.CreateRoleState, handlerFuncs
}

func deleteEventHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := gotgbot.InlineKeyboardMarkup{}
		markup.InlineKeyboard = helpers.ConfirmDeletingEventKeyboard
		msg := helpers.AddCallbackData(fmt.Sprintf("<b>%s</b>\n\nТы уверен, что хочешь удалить это собрание?", user.State.CallbackData.Query().Get("eventAlias")),
			user.State.CallbackData.String())
		_, _, err := c.EffectiveMessage.EditText(h.bot, msg, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   "HTML",
		})
		return err

	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}
		err = h.eventService.DeleteOneByID(eventID)
		if err != nil {
			return err
		}

		_, _, err = c.EffectiveMessage.EditText(h.bot, "Удаление завершено.", nil)
		if err != nil {
			return err
		}
		return err
	})

	return helpers.DeleteEventState, handlerFuncs
}

func createBandHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Введи название своей группы:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		user.State.Context.Band = &entity.Band{
			Name: c.EffectiveMessage.Text,
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}
		_, err := c.EffectiveChat.SendMessage(h.bot, "Теперь добавь имейл scala-drive@scala-chords-bot.iam.gserviceaccount.com в папку на Гугл Диске как редактора. После этого отправь мне ссылку на эту папку.",
			&gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		re := regexp.MustCompile(`(/folders/|id=)(.*?)(/|\?|$)`)
		matches := re.FindStringSubmatch(c.EffectiveMessage.Text)
		if matches == nil || len(matches) < 3 {
			user.State.Index--
			return h.Enter(c, user)
		}
		user.State.Context.Band.DriveFolderID = matches[2]
		user.Role = helpers.Admin
		band, err := h.bandService.UpdateOne(*user.State.Context.Band)
		if err != nil {
			return err
		}

		user.BandID = band.ID

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Ты добавлен в группу \"%s\" как администратор.", band.Name), nil)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Name: helpers.MainMenuState,
		}
		return h.Enter(c, user)
	})

	return helpers.CreateBandState, handlerFunc
}

func addBandAdminHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		users, err := h.userService.FindMultipleByBandID(user.BandID)
		if err != nil {
			return err
		}

		for _, user := range users {
			buttonText := user.Name
			if user.Role == helpers.Admin {
				buttonText += " (админ)"
			}
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err = c.EffectiveChat.SendMessage(h.bot, "Выбери пользователя, которого ты хочешь сделать администратором:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		regex := regexp.MustCompile(` \(админ\)$`)
		query := regex.ReplaceAllString(c.EffectiveMessage.Text, "")

		chosenUser, err := h.userService.FindOneByName(query)
		if err != nil {
			user.State.Index--
			return h.Enter(c, user)
		}

		chosenUser.Role = helpers.Admin
		_, err = h.userService.UpdateOne(*chosenUser)
		if err != nil {
			return err
		}

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Пользователь '%s' повышен до администратора.", chosenUser.Name), nil)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Name: helpers.MainMenuState,
		}

		return h.Enter(c, user)
	})

	return helpers.AddBandAdminState, handlerFunc
}

func transposeSongHandler() (int, []HandlerFunc) {

	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{Text: "C (Am)", CallbackData: helpers.AggregateCallbackData(state, index+1, "C")},
					{Text: "C# (A#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "C#")},
					{Text: "Db (Bbm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Db")},
				},
				{
					{Text: "D (Bm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "D")},
					{Text: "D# (Cm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "D#")},
					{Text: "Eb (Cm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Eb")},
				},
				{
					{Text: "E (C#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "E")},
				},
				{
					{Text: "F (Dm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "F")},
					{Text: "F# (D#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "F#")},
					{Text: "Gb (Ebm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Gb")},
				},
				{
					{Text: "G (Em)", CallbackData: helpers.AggregateCallbackData(state, index+1, "G")},
					{Text: "G# (Fm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "G#")},
					{Text: "Ab (Fm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Ab")},
				},
				{
					{Text: "A (F#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "A")},
					{Text: "A# (Gm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "A#")},
					{Text: "Bb (Gm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Bb")},
				},
				{
					{Text: "B (G#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "B")},
				},
				{
					{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")},
				},
			},
		}

		_, _, err := c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
			Caption:     helpers.AddCallbackData("Выбери новую тональность:", user.State.CallbackData.String()),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		if err != nil {
			return err
		}

		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, key := helpers.ParseCallbackData(c.CallbackQuery.Data)

		q := user.State.CallbackData.Query()
		q.Set("key", key)
		user.State.CallbackData.RawQuery = q.Encode()

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{Text: helpers.AppendSection, CallbackData: helpers.AggregateCallbackData(state, index+1, "-1")},
				},
			},
		}

		sectionsNumber, err := h.driveFileService.GetSectionsNumber(user.State.CallbackData.Query().Get("driveFileId"))
		if err != nil {
			return err
		}

		for i := 0; i < sectionsNumber; i++ {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
				{Text: fmt.Sprintf("Вместо %d-й секции", i+1), CallbackData: helpers.AggregateCallbackData(state, index+1, fmt.Sprintf("%d", i))},
			})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
			{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")},
		})

		c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
			Caption:     helpers.AddCallbackData("Куда ты хочешь вставить новую тональность?", user.State.CallbackData.String()),
			ReplyMarkup: markup,
			ParseMode:   "HTML",
		})

		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		_, _, sectionIndexStr := helpers.ParseCallbackData(c.CallbackQuery.Data)

		sectionIndex, _ := strconv.Atoi(sectionIndexStr)

		driveFile, err := h.driveFileService.TransposeOne(
			user.State.CallbackData.Query().Get("driveFileId"),
			user.State.CallbackData.Query().Get("key"),
			sectionIndex)
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(driveFile.Id)
		if err != nil {
			return err
		}

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		_, err = h.songService.UpdateOne(*song)

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")
		return h.enterInlineHandler(c, user)
	})

	return helpers.TransposeSongState, handlerFunc
}

func styleSongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	// Print list of found songs.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		driveFile, err := h.driveFileService.StyleOne(driveFileID)
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(driveFile.Id)
		if err != nil {
			return err
		}

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		_, err = h.songService.UpdateOne(*song)
		if err != nil {
			return err
		}

		// c.CallbackQuery.Answer(h.bot, nil)
		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")
		return h.enterInlineHandler(c, user)
	})
	return helpers.StyleSongState, handlerFunc
}

func addLyricsPageHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	// Print list of found songs.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		driveFile, err := h.driveFileService.AddLyricsPage(driveFileID)
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(driveFile.Id)
		if err != nil {
			return err
		}

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		_, err = h.songService.UpdateOne(*song)
		if err != nil {
			return err
		}

		// c.CallbackQuery.Answer(h.bot, nil)
		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")
		return h.enterInlineHandler(c, user)
	})
	return helpers.AddLyricsPageState, handlerFunc
}

func copySongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		c.EffectiveChat.SendAction(h.bot, "typing")

		file, err := h.driveFileService.FindOneByID(driveFileID)
		if err != nil {
			return err
		}

		file = &drive.File{
			Name:    file.Name,
			Parents: []string{user.Band.DriveFolderID},
		}

		copiedSong, err := h.driveFileService.CloneOne(driveFileID, file)
		if err != nil {
			return err
		}

		song, _, err := h.songService.FindOrCreateOneByDriveFileID(copiedSong.Id)
		if err != nil {
			return err
		}

		q := user.State.CallbackData.Query()
		q.Set("driveFileId", copiedSong.Id)
		user.State.CallbackData.RawQuery = q.Encode()

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: helpers.GetSongInitKeyboard(user, song),
		}
		c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
			Caption:     helpers.AddCallbackData("Скопировано", user.State.CallbackData.String()),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	return helpers.CopySongState, handlerFunc
}

func deleteSongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		if user.Role == helpers.Admin {
			err := h.songService.DeleteOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
			if err != nil {
				return err
			}

			c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{Caption: "Удалено"})
		}

		return nil
	})

	return helpers.DeleteSongState, handlerFunc
}

func uploadVoiceHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	markup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
		ResizeKeyboard: true,
	}

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, err := c.EffectiveChat.SendMessage(h.bot, "Введи название песни, к которой ты хочешь прикрепить эту партию:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		driveFiles, _, err := h.driveFileService.FindSomeByFullTextAndFolderID(c.EffectiveMessage.Text, user.Band.DriveFolderID, "")
		if err != nil {
			return err
		}

		if len(driveFiles) == 0 {
			_, err := c.EffectiveChat.SendMessage(h.bot, "Ничего не найдено. Попробуй другое название.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		// TODO: some sort of pagination.
		for _, driveFile := range driveFiles {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFile.Name}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err = c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseSong", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		foundDriveFile, err := h.driveFileService.FindOneByNameAndFolderID(c.EffectiveMessage.Text, user.Band.DriveFolderID)
		if err != nil {
			user.State.Index--
			return h.Enter(c, user)
		}

		song, _, err := h.songService.FindOrCreateOneByDriveFileID(foundDriveFile.Id)
		if err != nil {
			return err
		}

		user.State.Context.DriveFileID = song.DriveFileID

		markup := markup
		_, err = c.EffectiveChat.SendMessage(h.bot, "Отправь мне название этой партии:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.Voice.Name = c.EffectiveMessage.Text

		song, err := h.songService.FindOneByDriveFileID(user.State.Context.DriveFileID)
		if err != nil {
			return err
		}

		user.State.Context.Voice.SongID = song.ID

		_, err = h.voiceService.UpdateOne(*user.State.Context.Voice)
		if err != nil {
			return err
		}

		c.EffectiveChat.SendMessage(h.bot, "Добавление завершено.", nil)

		user.State = entity.State{
			Name: helpers.SongActionsState,
			Context: entity.Context{
				DriveFileID: user.State.Context.DriveFileID,
			},
		}
		return h.Enter(c, user)
	})

	// Upload voice from song menu.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State = entity.State{
			Name:    helpers.UploadVoiceState,
			Index:   4,
			Context: entity.Context{DriveFileID: user.State.CallbackData.Query().Get("driveFileId")},
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь мне аудио или голосовое сообщение:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		fileID := c.EffectiveMessage.Voice.FileId
		if fileID == "" {
			fileID = c.EffectiveMessage.Audio.FileId
		}
		user.State.Context.Voice = &entity.Voice{FileID: fileID}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь мне название этой партии:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.Voice.Name = c.EffectiveMessage.Text

		song, err := h.songService.FindOneByDriveFileID(user.State.Context.DriveFileID)
		if err != nil {
			return err
		}

		user.State.Context.Voice.SongID = song.ID

		_, err = h.voiceService.UpdateOne(*user.State.Context.Voice)
		if err != nil {
			return err
		}

		c.EffectiveChat.SendMessage(h.bot, "Добавление завершено.", nil)

		user.State = entity.State{
			Name: helpers.SongActionsState,
			Context: entity.Context{
				DriveFileID: user.State.Context.DriveFileID,
			},
			Next: &entity.State{Name: helpers.MainMenuState},
		}
		return h.Enter(c, user)
	})

	return helpers.UploadVoiceState, handlerFunc
}
