package handlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/txt"
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
