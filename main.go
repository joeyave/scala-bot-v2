package main

import (
	"context"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"net/http"
	"os"
	"scala-bot-v2/handler"
	"scala-bot-v2/repository"
	"scala-bot-v2/service"
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

	voiceRepository := repository.NewVoiceRepository(mongoClient)
	voiceService := service.NewVoiceService(voiceRepository)

	bandRepository := repository.NewBandRepository(mongoClient)
	bandService := service.NewBandService(bandRepository)

	driveFileService := service.NewDriveFileService(driveRepository, docsRepository)

	songRepository := repository.NewSongRepository(mongoClient)
	songService := service.NewSongService(songRepository, voiceRepository, bandRepository, driveRepository, driveFileService)

	userRepository := repository.NewUserRepository(mongoClient)
	userService := service.NewUserService(userRepository)

	membershipRepository := repository.NewMembershipRepository(mongoClient)
	membershipService := service.NewMembershipService(membershipRepository)

	eventRepository := repository.NewEventRepository(mongoClient)
	eventService := service.NewEventService(eventRepository, membershipRepository, driveFileService)

	roleRepository := repository.NewRoleRepository(mongoClient)
	roleService := service.NewRoleService(roleRepository)

	botHandler := handler.BotHandler{
		UserService:       userService,
		DriveFileService:  driveFileService,
		SongService:       songService,
		VoiceService:      voiceService,
		BandService:       bandService,
		MembershipService: membershipService,
		EventService:      eventService,
		RoleService:       roleService,
	}
	webAppHandler := handler.WebAppHandler{
		EventService: eventService,
		Bot:          bot,
	}

	// Create updater and dispatcher.
	updater := ext.NewUpdater(&ext.UpdaterOpts{
		ErrorLog: nil,
		DispatcherOpts: ext.DispatcherOpts{
			Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
				fmt.Println("an error occurred while handling update:", err.Error())
				return ext.DispatcherActionNoop
			},
			Panic:       nil,
			ErrorLog:    nil,
			MaxRoutines: 0,
		},
	})
	dispatcher := updater.Dispatcher

	dispatcher.AddHandler(handlers.NewCommand("start", botHandler.Menu))
	dispatcher.AddHandler(handlers.NewMessage(func(msg *gotgbot.Message) bool {

		if msg.WebAppData != nil && msg.WebAppData.ButtonText == "➕ Добавить собрание" {
			return true
		}

		return false
	}, botHandler.CreateEvent))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("eventChords:"), botHandler.EventChords))
	dispatcher.AddHandler(handlers.NewMessage(func(msg *gotgbot.Message) bool { return msg.Text == "🗓️ Расписание" }, botHandler.Events))

	router := gin.New()
	router.LoadHTMLGlob("tmpl/**/*.tmpl")
	router.Static("/assets", "./assets")

	router.GET("/web-app/create-event", webAppHandler.CreateEvent)
	router.GET("/web-app/edit-event/:id", webAppHandler.EditEvent)

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
