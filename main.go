package main

import (
	"context"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/gin-gonic/gin"
	"github.com/joeyave/scala-bot-v2/controller"
	"github.com/joeyave/scala-bot-v2/entities"
	myhandlers "github.com/joeyave/scala-bot-v2/handlers"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/repositories"
	"github.com/joeyave/scala-bot-v2/services"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"net/http"
	"os"
	"time"
)

func main() {
	// Create bot from environment value.
	bot, err := gotgbot.NewBot(os.Getenv("BOT_TOKEN"), &gotgbot.BotOpts{
		Client:      http.Client{},
		GetTimeout:  gotgbot.DefaultGetTimeout,
		PostTimeout: gotgbot.DefaultPostTimeout,
	})
	if err != nil {
		panic("failed to create new bot: " + err.Error())
	}

	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Fatal().Err(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer mongoClient.Disconnect(ctx)
	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal().Err(err)
	}

	driveRepository, err := drive.NewService(context.TODO(), option.WithCredentialsJSON([]byte(os.Getenv("GOOGLEAPIS_CREDENTIALS"))))
	if err != nil {
		log.Fatal().Msgf("Unable to retrieve Drive client: %v", err)
	}

	docsRepository, err := docs.NewService(context.TODO(), option.WithCredentialsJSON([]byte(os.Getenv("GOOGLEAPIS_CREDENTIALS"))))
	if err != nil {
		log.Fatal().Msgf("Unable to retrieve Docs client: %v", err)
	}

	voiceRepository := repositories.NewVoiceRepository(mongoClient)
	voiceService := services.NewVoiceService(voiceRepository)

	bandRepository := repositories.NewBandRepository(mongoClient)
	bandService := services.NewBandService(bandRepository)

	driveFileService := services.NewDriveFileService(driveRepository, docsRepository)

	songRepository := repositories.NewSongRepository(mongoClient)
	songService := services.NewSongService(songRepository, voiceRepository, bandRepository, driveRepository, driveFileService)

	userRepository := repositories.NewUserRepository(mongoClient)
	userService := services.NewUserService(userRepository)

	membershipRepository := repositories.NewMembershipRepository(mongoClient)
	membershipService := services.NewMembershipService(membershipRepository)

	eventRepository := repositories.NewEventRepository(mongoClient)
	eventService := services.NewEventService(eventRepository, membershipRepository, driveFileService)

	roleRepository := repositories.NewRoleRepository(mongoClient)
	roleService := services.NewRoleService(roleRepository)

	botController := controller.BotController{
		UserService:       userService,
		DriveFileService:  driveFileService,
		SongService:       songService,
		VoiceService:      voiceService,
		BandService:       bandService,
		MembershipService: membershipService,
		EventService:      eventService,
		RoleService:       roleService,
	}
	webAppController := controller.WebAppController{
		EventService: eventService,
		Bot:          bot,
	}

	// Create updater and dispatcher.
	updater := ext.NewUpdater(&ext.UpdaterOpts{
		ErrorLog: nil,
		DispatcherOpts: ext.DispatcherOpts{
			Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
				fmt.Println("an error occurred while handling update:", err.Error())

				_, sendMsgErr := ctx.EffectiveChat.SendMessage(b, "Произошла ошибка. Поправим.", nil)
				if sendMsgErr != nil {
					log.Error().Err(sendMsgErr).Msg("Error!")
					return ext.DispatcherActionEndGroups
				}

				user, findUserErr := userService.FindOneByID(ctx.EffectiveChat.Id)
				if findUserErr != nil {
					log.Error().Err(findUserErr).Msg("Error!")
					return ext.DispatcherActionEndGroups
				}

				// todo: send message to the logs channel
				log.Error().Err(err).Msg("Error!")

				user.State = &entities.State{Name: helpers.MainMenuState}
				_, err = userService.UpdateOne(*user)
				if findUserErr != nil {
					log.Error().Err(findUserErr).Msg("Error!")
					return ext.DispatcherActionEndGroups
				}

				return ext.DispatcherActionEndGroups
			},
			Panic:       nil,
			ErrorLog:    nil,
			MaxRoutines: 0,
		},
	})
	dispatcher := updater.Dispatcher

	handler := myhandlers.NewHandler(
		bot,
		userService,
		driveFileService,
		songService,
		voiceService,
		bandService,
		membershipService,
		eventService,
		roleService,
	)

	// old
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, handler.RegisterUser), 0)

	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Text, handler.OnText), 1)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Voice, handler.OnVoice), 1)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Audio, handler.OnAudio), 1)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.All, handler.OnCallback), 1)

	// new
	dispatcher.AddHandlerToGroup(handlers.NewCommand("start", botController.Menu), 2)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(func(msg *gotgbot.Message) bool {

		if msg.WebAppData != nil && msg.WebAppData.ButtonText == "➕ Добавить собрание" {
			return true
		}

		return false
	}, botController.CreateEvent), 2)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("eventChords:"), botController.EventChords), 2)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(func(msg *gotgbot.Message) bool { return msg.Text == "🗓️ Расписание" }, botController.Events), 2)

	go handler.NotifyUser()

	router := gin.New()
	router.LoadHTMLGlob("tmpl/**/*.tmpl")
	router.Static("/assets", "./assets")

	router.GET("/web-app/create-event", webAppController.CreateEvent)
	router.GET("/web-app/edit-event/:id", webAppController.EditEvent)

	go func() {
		// Start receiving updates.
		err = updater.StartPolling(bot, &ext.PollingOpts{DropPendingUpdates: true})
		if err != nil {
			panic("failed to start polling: " + err.Error())
		}
		fmt.Printf("%s has been started...\n", bot.User.Username)

		// Idle, to keep updates coming in, and avoid bot stopping.
		updater.Idle()
	}()

	err = router.Run()
	if err != nil {
		panic("error starting Gin: " + err.Error())
	}
}
