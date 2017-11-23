package bot

import (
	"strings"

	"github.com/charliemaiors/golang-wol/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	deviceChan       chan *types.AliasResponse
	getChan          chan *types.GetDev
	delDevChan       chan *types.DelDev
	passHandlingChan chan *types.PasswordHandling
	updatePassChan   chan *types.PasswordUpdate
	getAliases       chan chan string
	allowedUser      *tgbotapi.User
	bot              *tgbotapi.BotAPI
)

func RunBot(deviceChan chan *types.AliasResponse, getChan chan *types.GetDev, delDevChan chan *types.DelDev, passHandlingChan chan *types.PasswordHandling, updatePassChan chan *types.PasswordUpdate, getAliases chan chan string) {
	initBot(deviceChan, getChan, delDevChan, passHandlingChan, updatePassChan, getAliases)
	var err error
	token := viper.GetString("bot.telegram.token")
	bot, err = tgbotapi.NewBotAPI(token)
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
		go handleUpdate(update)

	}
}

func initBot(deviceChanDef chan *types.AliasResponse, getChanDef chan *types.GetDev, delDevChanDef chan *types.DelDev, passHandlingChanDef chan *types.PasswordHandling, updatePassChanDef chan *types.PasswordUpdate, getAliasesDef chan chan string) {
	deviceChan = deviceChanDef
	getChan = getChanDef
	delDevChan = delDevChanDef
	passHandlingChan = passHandlingChanDef
	updatePassChan = updatePassChanDef
	getAliases = getAliasesDef

	allowedUser = &tgbotapi.User{
		FirstName: viper.GetString("bot.telegram.firstname"),
		LastName:  viper.GetString("bot.telegram.lastname"),
		UserName:  viper.GetString("bot.telegram.username"),
		IsBot:     false,
	}
}

func handleUpdate(update tgbotapi.Update) {
	var msg tgbotapi.MessageConfig
	if !checkValidUser(update.Message.From) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You are not allowed to chat with this bot")
	}
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "list":
			devices := getAllDevices()
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You have registered this devices: "+devices)
		case "add":
			args := update.Message.CommandArguments()
			resp := deviceAdd(args)
		}

	}

	msg.ReplyToMessageID = update.Message.MessageID
	_, err := bot.Send(msg)
	if err != nil {
		log.Errorf("Got error while sending answer %v", err)
	}
}

func deviceAdd(args string) string {
	if args == "" {
		return "Usage: /add <device-alias> <device-ip> <device-mac>"
	}

	params := strings.Split(args, " ")
	if len(params) < 3 {
		return "Usage: /add <device-alias> <device-ip> <device-mac>"
	}

	dev := &types.Device{IP: params[1], Mac: params[2]}
	alias := types.Alias{Device: dev, Name: params[0]}
	resp := make(chan struct{})
	aliasResp := &types.AliasResponse{Alias: alias, Response: resp}
	deviceChan <- aliasResp
	respTmp := <-resp
	if respTmp != nil {
		return aliasResp.String()
	}
	return "Got error adding device"
}

func getAllDevices() string {
	resp := ""
	aliasChannel := make(chan string)
	getAliases <- aliasChannel
	for tmp := range aliasChannel {
		resp += ", " + tmp
	}
	return resp
}

func checkValidUser(sender *tgbotapi.User) bool {
	return sender.UserName == allowedUser.UserName && sender.FirstName == allowedUser.FirstName && sender.LastName == allowedUser.LastName
}
