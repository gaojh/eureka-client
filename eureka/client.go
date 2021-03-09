package eureka

import (
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Client eureka客户端
type Client struct {
	// for monitor system signal
	signalChan chan os.Signal
	mutex      sync.RWMutex
	Running    bool
	Config     *Config
	// eureka服务中注册的应用
	Applications *Applications

	//解析后eureka服务端地址
	eurekaUrls []string

	once sync.Once
}

// Start 启动时注册客户端，并后台刷新服务列表，以及心跳
func (c *Client) Start() {
	c.once.Do(func() {
		c.eurekaUrls = strings.Split(c.Config.DefaultZone, ",")
	})

	c.mutex.Lock()
	c.Running = true
	c.mutex.Unlock()
	// 注册
	for {
		if err := c.doRegister(); err != nil {
			log.Println("注册失败，等待5秒重试", err.Error())
			time.Sleep(time.Second * 5)
		} else {
			log.Println("注册成功")
			break
		}
	}

	// 刷新服务列表
	go c.refresh()
	// 心跳
	go c.heartbeat()
	// 监听退出信号，自动删除注册信息
	go c.handleSignal()
}

func (c *Client) pickUrl() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	index := 0
	if len(c.eurekaUrls) > 1 {
		index = rand.Intn(len(c.eurekaUrls) - 1)
	}
	return c.eurekaUrls[index]
}

// refresh 刷新服务列表
func (c *Client) refresh() {
	for {
		if c.Running {
			if err := c.doRefresh(); err != nil {
				log.Println(err)
			} /*else {
				log.Println("refresh application instance successful")
			}*/
		} else {
			break
		}
		sleep := time.Duration(c.Config.RegistryFetchIntervalSeconds)
		time.Sleep(sleep * time.Second)
	}
}

// heartbeat 心跳
func (c *Client) heartbeat() {
	for {
		if c.Running {
			if err := c.doHeartbeat(); err != nil {
				if err == ErrNotFound {
					log.Println("心跳丢失，重新注册")
					if err = c.doRegister(); err != nil {
						log.Printf("do register error: %s\n", err)
					}
					continue
				}
				log.Println(err)
			} /*else {
				log.Println("heartbeat application instance successful")
			}*/
		} else {
			break
		}
		sleep := time.Duration(c.Config.RenewalIntervalInSecs)
		time.Sleep(sleep * time.Second)
	}
}

func (c *Client) doRegister() error {
	instance := c.Config.instance
	return Register(c.pickUrl(), c.Config.App, instance)
}

func (c *Client) doUnRegister() error {
	instance := c.Config.instance
	return UnRegister(c.pickUrl(), instance.App, instance.InstanceID)
}

func (c *Client) doHeartbeat() error {
	instance := c.Config.instance
	return Heartbeat(c.pickUrl(), instance.App, instance.InstanceID)
}

func (c *Client) doRefresh() error {
	// todo If the delta is disabled or if it is the first time, get all applications

	// get all applications
	applications, err := Refresh(c.pickUrl())
	if err != nil {
		return err
	}

	// set applications
	c.mutex.Lock()
	c.Applications = applications
	c.mutex.Unlock()
	return nil
}

// handleSignal 监听退出信号，删除注册的实例
func (c *Client) handleSignal() {
	if c.signalChan == nil {
		c.signalChan = make(chan os.Signal)
	}
	signal.Notify(c.signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	for {
		switch <-c.signalChan {
		case syscall.SIGINT:
			fallthrough
		case syscall.SIGKILL:
			fallthrough
		case syscall.SIGTERM:
			log.Println("收到退出信号，解除注册")
			err := c.doUnRegister()
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println("解除注册成功")
			}
			os.Exit(0)
		}
	}
}

// NewClient 创建客户端
func NewClient(config *Config) *Client {
	defaultConfig(config)
	config.instance = NewInstance(getLocalIP(), config)
	return &Client{Config: config}
}

func defaultConfig(config *Config) {
	if config.DefaultZone == "" {
		config.DefaultZone = "http://localhost:8761/eureka/"
	}
	if config.RenewalIntervalInSecs == 0 {
		config.RenewalIntervalInSecs = 30
	}
	if config.RegistryFetchIntervalSeconds == 0 {
		config.RegistryFetchIntervalSeconds = 15
	}
	if config.DurationInSecs == 0 {
		config.DurationInSecs = 90
	}
	if config.App == "" {
		config.App = "server"
	} else {
		config.App = strings.ToLower(config.App)
	}
	if config.Port == 0 {
		config.Port = 80
	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	panic("Unable to get the local IP address")
}
