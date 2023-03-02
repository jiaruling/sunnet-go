package core

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// DONE: 事件列表
type eventList struct {
	size   int
	events []unix.EpollEvent
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.EpollEvent, size)}
}

type EpollMethods interface {
	onEvent(ev unix.EpollEvent)  // 事件处理
	onAccept(conn *Conn)
	onRW(conn *Conn, read, write bool)
	addEvent(fd int32)
	removeEvent(fd int32)
	modifyEvent(fd int32, epollOut bool)
}

var socketWorker = newEpoll()

type Epoll struct {
	epollFd int // epoll 文件描述符
}

// DONE: 初始化epoll
func newEpoll() *Epoll {
	epollFd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC) // 最大支持的并发连接数 unix.EPOLL_CLOEXEC=0x80000 也即是2^19~=50w
	if err != nil {
		fmt.Println("create epoll err", err.Error())
		_ = unix.Close(epollFd)
		return nil
	}
	return &Epoll{
		epollFd: epollFd,
	}
}

// DONE: 处理事件
func (e *Epoll) onEvent(ev unix.EpollEvent) {
	fd := ev.Fd
	conn := Inst.GetConn(fd)
	if conn == nil {
		fmt.Println("OnEvent error, conn is nil")
		return
	}
	isRead := ev.Events == unix.EPOLLIN
	isWrite := ev.Events == unix.EPOLLOUT
	isError := (ev.Events&unix.EPOLLERR == unix.EPOLLERR) || (ev.Events&unix.EPOLLHUP == unix.EPOLLHUP)
	if conn.Type == LISTEN && isRead {
		// 监听socket
		e.onAccept(conn)
	} else {
		// 普通socket
		if isRead || isWrite {
			e.onRW(conn, isRead, isWrite)
		}
		if isError {
			fmt.Println("OnError fd: ", fd)
		}
	}
}

// DONE: 监听socket
func (e *Epoll) onAccept(conn *Conn) {
	// step1：accept
	clientFd, _, err := unix.Accept(int(conn.Fd))
	if err != nil {
		fmt.Println("accept err:", err.Error())
		return
	}
	// step2：设置非阻塞
	err = unix.SetNonblock(clientFd, true)
	if err != nil {
		fmt.Println("set nonblock err:", err)
		return
	}
	// step3：添加连接对象
	Inst.AddConn(CLIENT, int32(clientFd), conn.ServiceId)
	// step4：添加到epoll监听列表
	e.addEvent(int32(clientFd))
	// step5：通知服务
	msg := NewSocketAcceptMsg(conn.Fd, int32(clientFd))
	Inst.Send(conn.ServiceId, msg)
	return
}

// done: 读写socket
func (e *Epoll) onRW(conn *Conn, read, write bool) {
	msg := NewSocketRWMsg(conn.Fd, read, write)
	Inst.Send(conn.ServiceId, msg)
	return
}

// DONE: epoll添加套接字
func (e *Epoll) addEvent(fd int32) {
	err := unix.EpollCtl(e.epollFd, unix.EPOLL_CTL_ADD, int(fd), &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLET,
		Fd:     fd,
		Pad:    0,
	})
	if err != nil {
		fmt.Println("AddEvent epoll_ctl Fail:", err.Error())
	}
}

// done: 从epoll删除套接字
func (e *Epoll) removeEvent(fd int32) {
	err := unix.EpollCtl(e.epollFd, unix.EPOLL_CTL_DEL, int(fd), nil)
	if err != nil {
		fmt.Println("RemoveEvent epoll_ctl Fail:", err.Error())
	}
}

// done: epoll中修改套接字
func (e *Epoll) modifyEvent(fd int32, epollOut bool) {
	var ev unix.EpollEvent
	ev.Fd = fd
	if epollOut {
		ev.Events = unix.EPOLLIN | unix.EPOLLET | unix.EPOLLOUT
	} else {
		ev.Events = unix.EPOLLIN | unix.EPOLLET
	}
	err := unix.EpollCtl(e.epollFd, unix.EPOLL_CTL_MOD, int(fd), &ev)
	if err != nil {
		fmt.Println("ModifyEvent epoll_ctl Fail:", err.Error())
	}
}

// DONE: 工作线程
func operator() {
	el := newEventList(128)
	for {
		//msec -1,会一直阻塞,直到有事件可以处理才会返回, n 事件个数
		eventCount, err := unix.EpollWait(socketWorker.epollFd, el.events, -1)
		if err != nil {
			fmt.Println("epoll wait error: ", err.Error())
			continue
		} else {
			for i := 0; i < eventCount; i++ {
				socketWorker.onEvent(el.events[i])
			}
		}
	}
}