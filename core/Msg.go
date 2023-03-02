package core

import "fmt"

// 类型枚举
type MessageType uint8

const (
	SERVICE MessageType = iota + 1
	SOCKET_ACCEPT
	SOCKET_RW
)

// ---------------------------------------------------------------------------------------------------------------------
type Message interface {
	GetType() MessageType
}

// 消息基类
type BaseMsg struct {
	Type MessageType
}

func (m BaseMsg) GetType() MessageType {
	return m.Type
}

// ---------------------------------------------------------------------------------------------------------------------
// 服务间消息
type ServiceMsg struct {
	BaseMsg
	Source uint16 // 消息发送方
	Buff   []byte //消息内容
	Size   uint32 //消息内容大小
}

func NewServiceMsg(source uint16, buff []byte, size uint32) ServiceMsg {
	return ServiceMsg{
		BaseMsg: BaseMsg{Type:SERVICE},
		Source:  source,
		Buff:    buff,
		Size:    size,
	}
}

func (m ServiceMsg) GetType() MessageType {
	return m.Type
}

func (m ServiceMsg) toString() string {
	return fmt.Sprintf("【sources】:%d \n 【size】:%d \n【send】%s", m.Source, m.Size, BytesToString(m.Buff))
}

// ---------------------------------------------------------------------------------------------------------------------
// 有新连接
type SocketAcceptMsg struct {
	BaseMsg
	listenFd int32 // 监听套接字描述符
	clientFd int32 // 客户端的套接字描述符
}

func NewSocketAcceptMsg(listenFd, clientFd int32) SocketAcceptMsg {
	return SocketAcceptMsg{
		BaseMsg:  BaseMsg{Type: SOCKET_ACCEPT},
		listenFd: listenFd,
		clientFd: clientFd,
	}
}

func (m SocketAcceptMsg) GetType() MessageType {
	return m.Type
}

// ---------------------------------------------------------------------------------------------------------------------
// 可读可写
type SocketRWMsg struct {
	BaseMsg
	Fd      int32 // 发生事件的套接字描述符
	IsRead  bool   // true 可读
	IsWrite bool   // true 可写
}

func NewSocketRWMsg(fd int32, isRead, isWrite bool) SocketRWMsg {
	return SocketRWMsg{
		BaseMsg:  BaseMsg{Type: SOCKET_RW},
		Fd: fd,
		IsRead: isRead,
		IsWrite: isWrite,
	}
}

func (m SocketRWMsg) GetType() MessageType {
	return m.Type
}
