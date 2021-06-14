package bots

import (
	"database/sql"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	BotToken string
	Socks5   string
	bot      *tb.Bot
	//logger   *log.Logger
)

const (
	dbDriverName = "mysql"
	logo         = `
  ______ _____ _____       _     ____        _   
 |  ____| ____/ ____|     | |   |  _ \      | |  
 | |__  | |__| (___  _   _| |__ | |_) | ___ | |_ 
 |  __| |___ \\___ \| | | | '_ \|  _ < / _ \| __|
 | |____ ___) |___) | |_| | |_) | |_) | (_) | |_ 
 |______|____/_____/ \__,_|_.__/|____/ \___/ \__|
`
)

var dbPath string

func BotStart() {
	MakeHandle()
	TaskLaunch()
	fmt.Println("Bot Start")
	fmt.Println("------------")
	bot.Start()
}
func MakeHandle() {
	fmt.Println("Make Handlers……")
	//所有用户
	bot.Handle("/start", bStart)
	bot.Handle("/my", bMy)
	bot.Handle("/bind", bBind1)
	bot.Handle("/unbind", bUnBind)
	bot.Handle("/export", bExport)
	bot.Handle("/help", bHelp)
	bot.Handle(tb.OnText, bOnText)
	//管理员
	bot.Handle("/task", bTask)
	bot.Handle("/log", bLog)
}
func TaskLaunch() {
	task := cron.New()
	//每三小时执行一次
	task.AddFunc(viper.GetString("cron"), SignTask)
	//log分为每天
	//task.AddFunc(" 0 0 * * *", InitLogger)
	//  */1 * * * *    1 */3 * * *
	fmt.Println("Cron Task Start……")
	task.Start()
}
func init() {
	fmt.Println(logo)

	//read config
	fmt.Println("Read Config……")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		zap.S().Errorw("failed to read config","error",err)
	}
	host := viper.GetString("mysql.host")
	user := viper.GetString("mysql.user")
	port := viper.GetString("mysql.port")
	pwd := viper.GetString("mysql.password")
	database := viper.GetString("mysql.database")
	dbPath = strings.Join([]string{user, ":", pwd, "@tcp(", host, ":", port, ")/", database, "?charset=utf8"}, "")
	//fmt.Println(path)
	db, err := sql.Open(dbDriverName, dbPath)
	if err != nil {
		zap.S().Errorw("failed to connect db","error",err,"path",dbPath)
	}
	fmt.Println("Connect MySQL Success!")
	if ok, err := CreateTB(); !ok {
		zap.S().Errorw("failed to create table","error",err,"path",dbPath)
	}
	defer db.Close()
	BotToken = viper.GetString("bot_token")
	Socks5 = viper.GetString("socks5")
	Poller := &tb.LongPoller{Timeout: 15 * time.Second}
	spamProtected := tb.NewMiddlewarePoller(Poller, func(upd *tb.Update) bool {
		if upd.Message == nil {
			return true
		}
		if !upd.Message.Private() {
			return false
		}
		return true
	})
	botsettings := tb.Settings{
		Token:  BotToken,
		Poller: spamProtected,
	}
	//set socks5
	if Socks5 != "" {
		fmt.Println("Proxy:" + Socks5)
		dialer, err := proxy.SOCKS5("tcp", Socks5, nil, proxy.Direct)
		if err != nil {
			zap.S().Errorw("failed to make dialer","error",err,"socks5",Socks5)
		}
		httpTransport := &http.Transport{}
		httpClient := &http.Client{Transport: httpTransport}
		httpTransport.Dial = dialer.Dial
		botsettings.Client = httpClient
	}
	//create bot
	bot, err = tb.NewBot(botsettings)
	if err != nil {
		zap.S().Errorw("failed to create bot","error",err)
	}
	fmt.Println("Bot: " + strconv.Itoa(bot.Me.ID) + " " + bot.Me.Username)
}