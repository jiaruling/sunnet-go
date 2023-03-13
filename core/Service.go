package core

import (
	"context"
	"fmt"

	"golang.org/x/sys/unix"
)

type ServiceMethods interface {
	// 回调函数(编写服务逻辑)
	onInit()
	OnMsg(msg Message)
	OnExit()
	// 插入消息
	PushMsg(msg Message) bool
	// 收到其它服务消息
	OnServiceMsg(msg ServiceMsg)
	// 新连接
	OnAcceptMsg(msg SocketAcceptMsg)
	// 套接字可读可写
	OnRWMsg(msg SocketRWMsg)
	// 收到客户端数据
	OnSocketData(fd int32, buff []byte, size int)
	// 套接字可写
	OnSocketWritable(fd int32)
	// 关闭连接前
	OnSocketClose(fd int32)
}

type Service struct {
	Id        uint16       // 唯一id
	Type      string       // 类型
	IsExiting bool         // 是否正在退出
	MsgQueue  chan Message // 消息队列
	MsgLen    int          // 消息队列的长度
	ctx       context.Context
	cancel    context.CancelFunc
}

// 构造函数
func newService(types string, msgLen int) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		Id:        0,
		Type:      types,
		IsExiting: false,
		MsgQueue:  make(chan Message, msgLen),
		MsgLen:    msgLen,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *Service) onInit() {
	fmt.Printf("[%s]:[%s] Service OnInit\n", format(int64(s.Id)), s.Type)
	go run(s)
}

func (s *Service) OnMsg(msg Message) {
	switch msg.GetType() {
	case SERVICE:
		m, ok := msg.(ServiceMsg)
		if ok {
			s.OnServiceMsg(m)
		}
	case SOCKET_ACCEPT:
		m, ok := msg.(SocketAcceptMsg)
		if ok {
			s.OnAcceptMsg(m)
		}
	case SOCKET_RW:
		m, ok := msg.(SocketRWMsg)
		if ok {
			s.OnRWMsg(m)
		}
	}
}

func (s *Service) OnExit() {
	fmt.Printf("Service [%d] OnExit", s.Id)
	// 设置为退出状态
	s.IsExiting = true
	// 关闭管道
	close(s.MsgQueue)
	// 关闭协程
	s.cancel()
}

func (s *Service) PushMsg(msg Message) bool {
	// 正在退出或管道已满
	if s.IsExiting || len(s.MsgQueue) == s.MsgLen {
		return false
	} else {
		s.MsgQueue <- msg
		return true
	}
}

// 收到其它服务消息
func (s *Service) OnServiceMsg(msg ServiceMsg) {
	fmt.Println(msg.toString())
	return
}

// Done: 新连接
func (s *Service) OnAcceptMsg(msg SocketAcceptMsg) {
	fmt.Println("OnAcceptMsg clientFd:", msg.clientFd)
}

// done: 套接字可读可写
func (s *Service) OnRWMsg(msg SocketRWMsg) {
	if msg.IsRead {
		buffSize := 512
		buf := make([]byte, buffSize)
		for {
			n, err := unix.Read(int(msg.Fd), buf)
			if n <= 0 && err != unix.EAGAIN  {
				if Inst.GetConn(msg.Fd) != nil {
					s.OnSocketClose(msg.Fd)
					Inst.CloseConn(msg.Fd)
				}
				break
			}
			if n > 0 {
				s.OnSocketData(msg.Fd, buf, n)
			}
			// 可能还有数据没有读取完
			if n == buffSize {
				continue
			}
			if n <= 0 {
				break
			}
		}
	}
	if msg.IsWrite {
		if Inst.GetConn(msg.Fd) != nil {
			s.OnSocketWritable(msg.Fd)
		}
	}
}

// done: 收到客户端数据
func (s *Service) OnSocketData(fd int32, buff []byte, size int) {
	fmt.Println("OnSocketData: ", fd, " buff: ", BytesToString(buff))
	// echo
	toClient := "lpy\n"
	_ , err:= unix.Write(int(fd), StringToBytes(toClient))
	if err != nil {
		fmt.Println("write to client err:", err.Error())
	}
}

// done: 套接字可写
func (s *Service) OnSocketWritable(fd int32) {
	fmt.Println("OnSocketWirtable ", fd)
}

// done: 关闭连接前
func (s *Service) OnSocketClose(fd int32) {
	fmt.Println("OnSocketClose ", fd)
}

// DONE: 完成
func run(srv *Service) {
LOOP:
	for {
		select {
		case <-srv.ctx.Done(): // 等待上级通知
			break LOOP
		case msg := <-srv.MsgQueue: // 获取消息
			srv.OnMsg(msg)
		}
	}
}
