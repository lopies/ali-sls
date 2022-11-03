# 📖 Introduction

Lightweight library encapsulates Aliyun log service SDK, uses high-performance data structure of object pool, ring
queue, uses asynchronous read and write logs, batch write logs, users only need simple configuration.

# 🎬 Getting started

```text
go get -u github.com/aliyun/aliyun-log-go-sdk
go get -u go.versetech.cn/sls
```

# 🎡 Use
## Demo

```go
    err := sls.Init("Endpoints", "AccessKeyId", "AccessKeySecret", "SecurityToken", mysls.WithInterval(2*time.Second), mysls.WithBufferLength(1000), mysls.WithMaxLogs(50), mysls.WithLevel("debug"))
    if err != nil {
        panic("init error")
    }
    err = sls.CreateProject("Project")
    if err != nil {
        panic("init sls createProject error")
    }
    err = sls.CreateLogStore("Project,","LogStore", 30, 2, true, 6)
    if err != nil {
        panic("create logStore error")
        return
    }
    sls.Run()
    sls.Debug("Project","LogStore",sls.KV("key","value"))
```
## Option
```go
func WithBufferLength(length int) logOption //Set the size of the write log buffer. The default size is 1000. If the buffer is full, the capacity will be automatically expanded
```
```go
func WithInterval(t time.Duration) logOption //Set the interval for triggering write to Ali cloud SLS (default: 2S)
```
```go
func WithMaxLogs(max int) logOption //Set the maximum number of group logs that trigger a write to Aliyun logs
```
```go
func WithLevel(level string) logOption //Set the log service level，debug/info/warn/error
```


