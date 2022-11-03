package logger

import (
	sls "github.com/aliyun/aliyun-log-go-sdk"
)

type Buffer struct {
	buf      []*sls.Log // 队列
	size     int        // 当前容量
	r        int        // 读指针计数
	w        int        // 写指针计数
	store    string
	logCache []*sls.Log
}

// 读写指针相遇为空
func (r *Buffer) IsEmpty() bool {
	return r.r == r.w
}

func (r *Buffer) Read() *sls.Log {
	if r.IsEmpty() {
		return nil
	}
	v := r.buf[r.r]
	r.r++
	if r.r == r.size {
		r.r = 0
	}
	return v
}

func (r *Buffer) Write(v *sls.Log) {
	r.buf[r.w] = v
	r.w++
	if r.w == r.size {
		r.w = 0
	}
	if r.w == r.r {
		r.grow()
	}
}

func (r *Buffer) grow() {
	var size int
	if r.size < 1024 {
		size = r.size * 2
	} else {
		size = r.size + r.size/4
	}

	buf := make([]*sls.Log, size)
	copy(buf[0:], r.buf[r.r:])
	copy(buf[r.size-r.r:], r.buf[0:r.r])
	r.r = 0
	r.w = r.size
	r.size = size
	r.buf = buf
}
