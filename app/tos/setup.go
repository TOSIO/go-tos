package main

import (
	"fmt"
	"math"
	"runtime"
	godebug "runtime/debug"
	"strconv"
	"time"

	"github.com/TOSIO/go-tos/sdag"
	"github.com/TOSIO/go-tos/services/dashboard"
	"github.com/elastic/gosigar"

	"github.com/TOSIO/go-tos/app/utils"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/metrics"
	"github.com/TOSIO/go-tos/internal/debug"
	"github.com/TOSIO/go-tos/node"
	cli "gopkg.in/urfave/cli.v1"
)

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.HTTPModules = append(cfg.HTTPModules, "sdag", "TOS")
	cfg.WSModules = append(cfg.WSModules, "sdag", "TOS")
	cfg.IPCPath = "geth.ipc"
	return cfg
}

//读取配置文件、各模块应用命令行参数进行配置初始化
//返回节点对象
func makeConfigNode(ctx *cli.Context) (*node.Node, tosConfig) {
	// Load defaults.
	cfg := tosConfig{
		// 各模块config变量初始化
		Sdag:      sdag.DefaultConfig,
		Node:      defaultNodeConfig(),
		Dashboard: dashboard.DefaultConfig,
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.应用命令行传递进来的参数（参数封装在ctx中）
	utils.ApplyNodeFlags(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	utils.ApplySdagFlags(ctx, &cfg.Sdag)
	utils.ApplyDashboardConfig(ctx, &cfg.Dashboard)

	// 其他模块config设置
	log.Info("Warning! Other moduler config is not yet been")
	return stack, cfg
}

func activePPROF(ctx *cli.Context) error {
	runtime.GOMAXPROCS(runtime.NumCPU())

	logdir := (&node.Config{DataDir: utils.MakeDataDir(ctx)}).ResolvePath("logs")
	if ctx.GlobalBool(utils.DashboardEnabledFlag.Name) {
		logdir = (&node.Config{DataDir: utils.MakeDataDir(ctx)}).ResolvePath("logs")
	}
	fmt.Printf("Log dir is %s", logdir)
	if err := debug.Setup(ctx, logdir); err != nil {
		return err
	}
	// Cap the cache allowance and tune the garbage collector
	var mem gosigar.Mem
	if err := mem.Get(); err == nil {
		allowance := int(mem.Total / 1024 / 1024 / 3)
		if cache := ctx.GlobalInt(utils.CacheFlag.Name); cache > allowance {
			log.Warn("Sanitizing cache to Go's GC limits", "provided", cache, "updated", allowance)
			ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(allowance))
		}
	}
	// Ensure Go's GC ignores the database cache for trigger percentage
	cache := ctx.GlobalInt(utils.CacheFlag.Name)
	gogc := math.Max(20, math.Min(100, 100/(float64(cache)/1024)))

	log.Debug("Sanitizing Go's GC trigger", "percent", int(gogc))
	godebug.SetGCPercent(int(gogc))

	// Start metrics export if enabled
	//utils.SetupMetrics(ctx)

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)

	return nil
}

//生成节点对象、注册服务
func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)

	// 服务注册
	utils.RegisterSdagService(stack, &cfg.Sdag)
	if ctx.GlobalBool(utils.DashboardEnabledFlag.Name) {
		utils.RegisterDashboardService(stack, &cfg.Dashboard, gitCommit)
	}
	return stack
}

//启动账户相关服务
func activeAccount(ctx *cli.Context, stack *node.Node) {
	log.Info("Starting account service")
}

//启动辅助服务
func startAuxservice(ctx *cli.Context, stack *node.Node) {
	log.Info("Starting aux service")
}

//钱包循环
func walletloop(stack *node.Node) {
	log.Info("Starting wallet-loop")
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
// 启动节点（本机）
func startNode(ctx *cli.Context, stack *node.Node) {
	debug.Memsize.Add("node", stack)

	// Start up the node itself
	utils.StartNode(stack)

	// 账户相关
	activeAccount(ctx, stack)

	// 钱包相关
	go walletloop(stack)

	// 启动辅助服务（如挖矿、内存交易池等）
	// Start auxiliary services if enabled
	startAuxservice(ctx, stack)
}
