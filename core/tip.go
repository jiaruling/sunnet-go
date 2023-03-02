package core

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

// StringToBytes 实现string 转换成 []byte, 不用额外的内存分配
func StringToBytes(str string) (bytes []byte) {
	ss := *(*reflect.StringHeader)(unsafe.Pointer(&str))
	bs := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	bs.Data = ss.Data
	bs.Len = ss.Len
	bs.Cap = ss.Len
	return bytes
}

// BytesToString 实现 []byte 转换成 string, 不需要额外的内存分配
func BytesToString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}


func format(n int64) string {
	// 十进制转十六进制
	s := strconv.FormatInt(n, 16)
	// 前置补零
	return fmt.Sprintf("%06s", s)
}