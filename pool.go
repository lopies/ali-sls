package logger

import (
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"sync"
)

type logContentPool []*sls.LogContent
type logPool []*sls.Log

var logContentPoolLocker sync.Mutex
var logPoolLocker sync.Mutex

var slsLogContent logContentPool = make([]*sls.LogContent, 0, 5000)
var slsLog logPool = make([]*sls.Log, 0, 3000)

func (s *logContentPool) Push(log *sls.LogContent) {
	logContentPoolLocker.Lock()
	defer logContentPoolLocker.Unlock()
	*s = append(*s, log)
}

func (s *logPool) Push(log *sls.Log) {
	logPoolLocker.Lock()
	defer logPoolLocker.Unlock()
	*s = append(*s, log)
}

func (s *logContentPool) Pop() *sls.LogContent {
	logContentPoolLocker.Lock()
	defer logContentPoolLocker.Unlock()
	lenght := len(*s)
	if lenght > 0 {
		last := lenght - 1
		entity := (*s)[last]
		*s = (*s)[:last]
		return entity
	} else {
		logContent := new(sls.LogContent)
		return logContent
	}
}

func (s *logPool) Pop() *sls.Log {
	logPoolLocker.Lock()
	defer logPoolLocker.Unlock()
	lenght := len(*s)
	if lenght > 0 {
		last := lenght - 1
		entity := (*s)[last]
		*s = (*s)[:last]
		return entity
	} else {
		log := new(sls.Log)
		return log
	}
}
