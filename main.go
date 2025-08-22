package main

import (
	"done-hub/cli"
	"done-hub/common"
	"done-hub/common/cache"
	"done-hub/common/config"
	"done-hub/common/logger"
	"done-hub/common/notify"
	"done-hub/common/oidc"
	"done-hub/common/redis"
	"done-hub/common/requester"
	"done-hub/common/search"
	"done-hub/common/storage"
	"done-hub/common/telegram"
	"done-hub/controller"
	"done-hub/cron"
	"done-hub/middleware"
	"done-hub/model"
	"done-hub/relay/task"
	"done-hub/router"
	"done-hub/safty"
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

//go:embed web/build
var buildFS embed.FS

//go:embed web/build/index.html
var indexPage []byte

func main() {
	cli.InitCli()
	config.InitConf()
	if viper.GetString("log_level") == "debug" {
		config.Debug = true
	}

	logger.SetupLogger()
	logger.SysLog("Done Hub " + config.Version + " started")

	// Initialize user token
	err := common.InitUserToken()
	if err != nil {
		logger.FatalLog("failed to initialize user token: " + err.Error())
	}

	// Initialize SQL Database
	model.SetupDB()
	defer model.CloseDB()
	// Initialize Redis
	redis.InitRedisClient()
	cache.InitCacheManager()
	// Initialize invite code lock system
	model.InitInviteCodeLock()
	// Initialize options
	model.InitOptionMap()
	// Initialize oidc
	oidc.InitOIDCConfig()
	model.NewPricing()
	model.HandleOldTokenMaxId()

	initMemoryCache()
	initSync()

	common.InitTokenEncoders()
	requester.InitHttpClient()
	// Initialize Telegram bot
	telegram.InitTelegramBot()

	controller.InitMidjourneyTask()
	task.InitTask()
	notify.InitNotifier()
	cron.InitCron()
	storage.InitStorage()
	search.InitSearcher()
	// 初始化安全检查器
	safty.InitSaftyTools()
	// 初始化账单数据
	if config.UserInvoiceMonth {
		logger.SysLog("Enable User Invoice Monthly Data")
		go model.InsertStatisticsMonth()
	}
	initHttpServer()
}

func initMemoryCache() {
	if viper.GetBool("memory_cache_enabled") {
		config.MemoryCacheEnabled = true
	}

	if !config.MemoryCacheEnabled {
		return
	}

	syncFrequency := viper.GetInt("sync_frequency")
	model.TokenCacheSeconds = syncFrequency

	logger.SysLog("memory cache enabled")
	logger.SysLog(fmt.Sprintf("sync frequency: %d seconds", syncFrequency))
	go model.SyncOptions(syncFrequency)
	go SyncChannelCache(syncFrequency)
}

func initSync() {
	// go controller.AutomaticallyUpdateChannels(viper.GetInt("channel.update_frequency"))
	go controller.AutomaticallyTestChannels(viper.GetInt("channel.test_frequency"))
}

func initHttpServer() {
	if viper.GetString("gin_mode") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	server := gin.New()
	server.Use(gin.Recovery())
	server.Use(middleware.RequestId())
	middleware.SetUpLogger(server)

	trustedHeader := viper.GetString("trusted_header")
	if trustedHeader != "" {
		server.TrustedPlatform = trustedHeader
	}

	store := cookie.NewStore([]byte(config.SessionSecret))

	// 检测是否在 HTTPS 环境下运行
	isHTTPS := viper.GetBool("https") || viper.GetString("trusted_header") == "CF-Connecting-IP"

	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   2592000, // 30 days
		HttpOnly: true,
		Secure:   isHTTPS,                 // 在 HTTPS 环境下启用 Secure
		SameSite: http.SameSiteStrictMode, // 改为 Lax 模式，兼容 CDN 环境
	})

	server.Use(sessions.Sessions("session", store))

	router.SetRouter(server, buildFS, indexPage)
	port := viper.GetString("port")

	err := server.Run(":" + port)
	if err != nil {
		logger.FatalLog("failed to start HTTP server: " + err.Error())
	}
}

func SyncChannelCache(frequency int) {
	// 只有 从 服务器端获取数据的时候才会用到
	if config.IsMasterNode {
		logger.SysLog("master node does't synchronize the channel")
		return
	}
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		logger.SysLog("syncing channels from database")
		model.ChannelGroup.Load()
		model.PricingInstance.Init()
		model.ModelOwnedBysInstance.Load()
	}
}
