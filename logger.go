package logger

import (
	"errors"
	"fmt"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/gogo/protobuf/proto"
	"net"
	"time"
)

var base *slsLoggerBase
var projectMgr map[string]*logStoreCache

func init() {
	base = &slsLoggerBase{
		source: proto.String(ipv4),
	}
	projectMgr = make(map[string]*logStoreCache)
}

func getLogStoreCache() *logStoreCache {
	return &logStoreCache{
		logStoreBufferMap: make(map[string]*Buffer, 10),
		logStoreChanMap:   make(map[string]chan *sls.Log, 10),
	}
}

func WithBufferLength(length int) logOption {
	if base.active {
		return nil
	}
	return funcOption{
		f: func(options *options) {
			options.bufferLength = length
		},
	}
}

func WithInterval(t time.Duration) logOption {
	if base.active {
		return nil
	}
	return funcOption{
		f: func(options *options) {
			options.interval = t
		},
	}
}

func WithMaxLogs(max int) logOption {
	if base.active {
		return nil
	}
	return funcOption{
		f: func(options *options) {
			options.maxLogs = max
		},
	}
}

func WithLevel(level string) logOption {
	fo := funcOption{}
	switch level {
	case "error":
		fo.f = func(options *options) {
			options.level = ErrorLevel
		}
		break
	case "warn":
		fo.f = func(options *options) {
			options.level = WarnLevel
		}
		break
	case "info":
		fo.f = func(options *options) {
			options.level = InfoLevel
		}
		break
	default:
		fo.f = func(options *options) {
			options.level = DebugLevel
		}
		break
	}
	return fo
}

type (
	logStoreCache struct {
		logStoreBufferMap map[string]*Buffer       //Buffer
		logStoreChanMap   map[string]chan *sls.Log //写日志通道
	}

	slsLoggerBase struct {
		client       sls.ClientInterface
		active       bool
		source       *string
		init         bool
		bufferLength int           //环形队列的初始化长度，默认1000
		interval     time.Duration //多长时间异步触发一次写日志，默认一秒
		maxLogs      int           //触发一次最多写多少条日志（每个库），默认50
		chanLength   int           //写日志通道大小
		level        Level         //日志级别
	}

	options struct {
		bufferLength int
		interval     time.Duration
		maxLogs      int
		level        Level
	}

	logOption interface {
		apply(*options)
	}

	funcOption struct {
		f func(*options)
	}
)

func (fo funcOption) apply(option *options) {
	fo.f(option)
}

type Content sls.LogContent

func KV(key, value string) *sls.LogContent {
	content := slsLogContent.Pop()
	content.Key = proto.String(key)
	content.Value = proto.String(value)
	return content
}

func storage(project, logStore string, logContents []*sls.LogContent) {
	log := slsLog.Pop()
	log.Time = proto.Uint32(uint32(time.Now().Unix()))
	log.Contents = logContents
	storeCache := projectMgr[project]
	if storeCache == nil {
		return
	}
	if c, ok := storeCache.logStoreChanMap[logStore]; ok {
		c <- log
	}
}

func readBuffer(project string, buffer *Buffer) {
	for i := 0; i < base.maxLogs; i++ {
		d := buffer.Read()
		if d != nil {
			buffer.logCache = append(buffer.logCache, d)
		} else {
			break
		}
	}
	if len(buffer.logCache) == 0 {
		return
	}
	group := &sls.LogGroup{
		Source: base.source,
		Logs:   buffer.logCache,
	}
	err := base.client.PutLogs(project, buffer.store, group)
	if err != nil {
		fmt.Printf("调用日志服务失败,err=%v", err)
	}
	group.Source = nil
	group.Logs = nil
	for _, j := range buffer.logCache {
		for _, n := range j.Contents {
			slsLogContent.Push(n)
		}
		slsLog.Push(j)
	}
	buffer.logCache = buffer.logCache[0:0]
}

func sendSLS() {
	go func() {
		ticker := time.NewTicker(base.interval)
		for {
			select {
			case <-ticker.C:
				for key := range projectMgr {
					logCache := projectMgr[key]
					for logStore := range logCache.logStoreBufferMap {
						readBuffer(key, logCache.logStoreBufferMap[logStore])
					}
				}
			}
		}
	}()
}

func CreateLogStore(project, logStore string, ttl, shardCnt int, autoSplit bool, maxSplitShard int) error {
	if !base.init {
		return fmt.Errorf("please sls init first")
	}
	if base.active {
		return fmt.Errorf("sls log is runing")
	}
	if _, ok := projectMgr[project]; !ok {
		return errors.New("create project first")
	}
	logCache := projectMgr[project]
	if _, ok := logCache.logStoreBufferMap[logStore]; ok {
		return fmt.Errorf("already create same name log store")
	}
	err := base.client.CreateLogStore(project, logStore, ttl, shardCnt, autoSplit, maxSplitShard)
	if err != nil {
		if e, ok := err.(*sls.Error); ok && e.Code == "LogStoreAlreadyExist" {
		} else {
			return fmt.Errorf(fmt.Sprintf("init sls CreateLogStore error,err=%v", e))
		}
	}

	buffer := &Buffer{
		buf:      make([]*sls.Log, base.bufferLength, base.bufferLength),
		size:     base.bufferLength,
		store:    logStore,
		logCache: make([]*sls.Log, 0, base.maxLogs),
	}
	logCache.logStoreBufferMap[logStore] = buffer
	logChan := make(chan *sls.Log, base.chanLength)
	logCache.logStoreChanMap[logStore] = logChan
	go func(b *Buffer, c chan *sls.Log) {
		for {
			select {
			case log := <-c:
				b.Write(log)
			}
		}
	}(buffer, logChan)
	return nil
}

func CreateProject(project string) error {
	if !base.init {
		return errors.New("init sls client first")
	}
	_, err := base.client.CreateProject(project, "")
	if err != nil {
		if e, ok := err.(*sls.Error); ok && e.Code == "ProjectAlreadyExist" {
			err = nil
		} else {
			return errors.New("init sls createProject error")
		}
	}
	if _, ok := projectMgr[project]; ok {
		return errors.New("the project name already exists")
	}
	projectMgr[project] = getLogStoreCache()
	return err
}

func Init(endpoints, accessKeyId, accessKeySecret, securityToken string, opts ...logOption) error {
	fmt.Println("init sls log")
	if base.init {
		fmt.Println("SLS client already init")
	}
	client := sls.CreateNormalInterface(endpoints, accessKeyId, accessKeySecret, securityToken)
	o := options{
		bufferLength: 1000,
		interval:     time.Second * 2,
		maxLogs:      50,
		level:        DebugLevel,
	}
	for _, opt := range opts {
		opt.apply(&o)
	}
	base.bufferLength = o.bufferLength
	base.interval = o.interval
	base.maxLogs = o.maxLogs
	base.level = o.level
	base.client = client
	base.init = true
	return nil
}

//运行函数
func Run() {
	sendSLS()
	base.active = true
}

func Debug(project, logStore string, logContents ...*sls.LogContent) {
	if !base.active {
		return
	}
	if base.level > DebugLevel {
		return
	}
	storage(project, logStore, logContents)
}

func Info(project, logStore string, logContents ...*sls.LogContent) {
	if !base.active {
		return
	}
	if base.level > InfoLevel {
		return
	}
	storage(project, logStore, logContents)
}

func Warn(project, logStore string, logContents ...*sls.LogContent) {
	if !base.active {
		return
	}
	if base.level > WarnLevel {
		return
	}
	storage(project, logStore, logContents)
}

func Error(project, logStore string, logContents ...*sls.LogContent) {
	if !base.active {
		return
	}
	if base.level > ErrorLevel {
		return
	}
	storage(project, logStore, logContents)
}

var ipv4 = getLocalIPv4Address()

func getLocalIPv4Address() (ipv4Address string) {
	ip := ""
	//获取所有网卡
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ip
	}
	//遍历
	for _, addr := range addrs {
		//取网络地址的网卡的信息
		ipNet, isIpNet := addr.(*net.IPNet)
		//是网卡并且不是本地环回网卡
		if isIpNet && !ipNet.IP.IsLoopback() {
			//能正常转成ipv4
			if ipNet.IP.To4() != nil {
				return ipNet.IP.To4().String()
			}
		}
	}
	return ip
}
