package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/entities"
)

type HandlerFunc = func(h *Handler, c *ext.Context, user *entities.User) error

var handlers = make(map[int][]HandlerFunc, 0)

// Register your handlers here.
func init() {
	registerHandlers(
		mainMenuHandler,
		// addBandAdminHandler,
		createSongHandler,
		copySongHandler,
		// deleteSongHandler,
		createBandHandler,
		chooseBandHandler,
		styleSongHandler,
		addLyricsPageHandler,
		changeSongBPMHandler,
		searchSongHandler,
		setlistHandler,
		songActionsHandler,
		getVoicesHandler,
		uploadVoiceHandler,
		deleteVoiceHandler,
		addSongTagHandler,
		transposeSongHandler,
		getEventsHandler,
		createEventHandler,
		eventActionsHandler,
		settingsHandler,
		createRoleHandler,
		addEventMemberHandler,
		addEventSongHandler,
		changeEventNotesHandler,
		deleteEventHandler,
		changeSongOrderHandler,
		deleteEventMemberHandler,
		deleteEventSongHandler,
		addBandAdminHandler,
		deleteSongHandler,
		getSongsFromMongoHandler,
		changeEventDateHandler,
		editInlineKeyboardHandler,
	)
}

func registerHandlers(funcs ...func() (name int, funcs []HandlerFunc)) {
	for _, f := range funcs {
		name, hf := f()
		handlers[name] = hf
	}
}
