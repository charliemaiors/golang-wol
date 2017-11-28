package bot

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/charliemaiors/golang-wol/types"
	"github.com/charliemaiors/golang-wol/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const (
	delims = ":-"
)

var (
	deviceChan     chan *types.AliasResponse
	getChan        chan *types.GetDev
	delDevChan     chan *types.DelDev
	getAliases     chan chan string
	allowedUser    *tgbotapi.User
	bot            *tgbotapi.BotAPI
	turnOffPort    string
	turnOffCommand string
	reMAC          = regexp.MustCompile(`^([0-9a-fA-F]{2}[` + delims + `]){5}([0-9a-fA-F]{2})$`)
)

//RunBot starts the telegram bot based on configuration file, this bot will not use password checking because auth is made using allowed user
func RunBot(deviceChan chan *types.AliasResponse, getChan chan *types.GetDev, delDevChan chan *types.DelDev, passHandlingChan chan *types.PasswordHandling, updatePassChan chan *types.PasswordUpdate, getAliases chan chan string) {
	initBot(deviceChan, getChan, delDevChan, getAliases)
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

func initBot(deviceChanDef chan *types.AliasResponse, getChanDef chan *types.GetDev, delDevChanDef chan *types.DelDev, getAliasesDef chan chan string) {
	deviceChan = deviceChanDef
	getChan = getChanDef
	delDevChan = delDevChanDef
	getAliases = getAliasesDef

	turnOffPort = viper.GetString("server.command.port")
	turnOffCommand = viper.GetString("server.command.option")

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
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "You are not allowed to chat with this bot")
	}
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			startMessage := produceStartMessage()
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, startMessage)
		case "help":
			helpMessage := produceStartMessage()
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, helpMessage)
		case "list":
			devices := getAllDevices()
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "You have registered this devices: "+devices)
		case "add":
			args := update.Message.CommandArguments()
			resp := deviceAdd(args)
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, resp)
		case "wakeup":
			args := update.Message.CommandArguments()
			resp := deviceWake(args) //TODO test it, if is to slow try to add concurrency
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, resp)
		case "turnoff":
			args := update.Message.CommandArguments()
			resp := turnOffDev(args)
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, resp)
		case "check":
			args := update.Message.CommandArguments()
			resp := getDeviceStatus(args)
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, resp)
		}

	}

	msg.ReplyToMessageID = update.Message.MessageID
	_, err := bot.Send(msg)
	if err != nil {
		log.Errorf("Got error while sending answer %v", err)
	}

}

func deviceWake(args string) string {
	if args == "" {
		return "Usage: /wakeup <device-alias-1> <device-alias-2> ..."
	}
	aliases := strings.Split(args, " ")
	status := make(map[string]string)

	devChannel := make(chan *types.Device)
	for _, alias := range aliases {
		getCurrentDev := &types.GetDev{Alias: alias, Response: devChannel}
		getChan <- getCurrentDev
		devTmp := <-devChannel

		if devTmp == nil {
			status[alias] = "No device found with alias " + alias
			continue
		}

		alreadyAlive := utils.CheckHealt(devTmp.IP)
		if alreadyAlive {
			status[alias] = "Device already awake"
		}

		err := utils.SendPacket(devTmp.Mac, devTmp.IP)
		if err != nil {
			status[alias] = err.Error()
			continue
		}

		report, err := utils.PingHost(devTmp.IP, alreadyAlive)
		if err != nil {
			status[alias] = err.Error()
			continue
		}

		time, err := isDeviceAlive(report)
		if err != nil {
			status[alias] = err.Error()
			continue
		}
		status[alias] = "Device alive at " + time.String()
	}
	close(devChannel)
	return getReport(status)
}

func turnOffDev(args string) string {
	if args == "" {
		return "usage /turnoff <device-alias>"
	}
	devChannel := make(chan *types.Device)
	getCurrentDev := &types.GetDev{Alias: args, Response: devChannel}
	getChan <- getCurrentDev
	dev := <-devChannel
	if dev == nil {
		return "Device not found"
	}
	err := utils.TurnOffDev(dev.IP, turnOffPort, turnOffCommand)
	if err != nil {
		return err.Error()
	}
	return "Device " + args + " with " + dev.String() + " is asleep"
}

func getDeviceStatus(args string) string {
	if args == "" {
		return "usage /check <device-alias>"
	}

	devChannel := make(chan *types.Device)
	getCurrentDev := &types.GetDev{Alias: args, Response: devChannel}
	getChan <- getCurrentDev
	dev := <-devChannel
	if dev == nil {
		return "Device not found"
	}
	close(devChannel)

	alive := utils.CheckHealt(dev.IP)

	if alive {
		return "Device " + args + " with " + dev.String() + " is awake"
	}

	return "Device " + args + " with " + dev.String() + " is asleep"
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
	_, ok := <-resp
	if ok {
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

func produceStartMessage() string {
	return "This bot could help you in order to handle your devices, the available commands are:\n/list - list all devices under server control" +
		"\n/add <device-alias> <device-ip> <device-mac> - add a single device under server control\n/wakeup <device-alias-1> <device-alias-2> ... - wake up devices with given aliases if they are registered to current server" +
		"\n/check <device-alias> - check if given device is alive\n/turnoff <device-alias> - will turn off your device\n/help - will print this output"
}

func getReport(status map[string]string) string {
	final := ""
	for k, v := range status {
		final += k + ":" + v + "\n"
	}
	return final
}

func isDeviceAlive(report map[time.Time]bool) (time.Time, error) {
	for k, v := range report {
		if v {
			return k, nil
		}
	}
	return time.Now(), errors.New("Device is sleeping") //time.Now() because nil is not valid for time.Time
}

func checkValidUser(sender *tgbotapi.User) bool {
	return sender.UserName == allowedUser.UserName && sender.FirstName == allowedUser.FirstName && sender.LastName == allowedUser.LastName
}
