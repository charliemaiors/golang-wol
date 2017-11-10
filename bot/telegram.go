package bot

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var allowedUser *tgbotapi.User

func RunBot() {
	allowedUser = &tgbotapi.User{
		FirstName: viper.GetString("bot.telegram.firstname"),
		LastName:  viper.GetString("bot.telegram.lastname"),
		UserName:  viper.GetString("bot.telegram.username"),
		IsBot:     false,
	}

	token := viper.GetString("bot.telegram.token")
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		panic(err)
	}

	log.Debugf("Authorized bot %s", bot.Self.String())
	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	if err != nil {
		log.Errorf("Could not get updates, %v error", err)
		return
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}
	}
}
