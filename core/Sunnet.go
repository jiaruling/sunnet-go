package core

import (
	"fmt"
	"net"
	"sync"

	"golang.org/x/sys/unix"
)

var Inst = NewSunnet(10)

type SunnetMethods interface {
	Start()
	NewService(types string, msgLen int) uint16      // 增加服务
	KillService(id uint16)               // 删除服务，仅限服务自己调用
	getService(id uint16) (srv *Service) // 获取服务
	Send(toId uint16, msg Message)       // 发送消息
	AddConn(types ConnType, fd int32, serviceId uint16)   // 增加连接
	GetConn(fd int32) (conn *Conn)                        // 获取连接
	RemoveConn(fd int32)                         // 删除连接
	Listen(ipAddr string, port, serviceId uint16)
	CloseConn(fd int32)
}

type Sunnet struct {
	Services     map[uint16]*Service // 服务列表
	ServiceId    uint16              // 最大id值
	ServicesLock sync.RWMutex        // 读写锁
	Conns        map[int32]*Conn    // 连接列表
	ConnsLock    sync.RWMutex        // 读写锁
}

func NewSunnet(cap int) *Sunnet {
	return &Sunnet{
		ServiceId: 1,
		Services:  make(map[uint16]*Service, cap),
		Conns:     make(map[int32]*Conn, cap),
	}
}

func (s *Sunnet) Start() {
	fmt.Println("socket worker start...")
	go operator()
}

func (s *Sunnet) NewService(types string, msgLen int) uint16 {
	srv := newService(types, msgLen)
	s.ServicesLock.Lock()
	{
		srv.Id = s.ServiceId
		s.ServiceId++
		s.Services[srv.Id] = srv
	}
	s.ServicesLock.Unlock()
	srv.onInit()
	return srv.Id
}

func (s *Sunnet) KillService(id uint16) {
	srv := s.getService(id)
	if srv == nil {
		return
	}
	// 退出前
	srv.OnExit()
	srv.IsExiting = true
	// 删除服务
	s.ServicesLock.Lock()
	{
		delete(s.Services, id)
	}
	s.ServicesLock.Unlock()
	return
}

func (s *Sunnet) getService(id uint16) (srv *Service) {
	s.ServicesLock.RLock()
	{
		srv = s.Services[id]
	}
	s.ServicesLock.RUnlock()
	return
}

// DONE: 向服务发送消息
func (s *Sunnet) Send(toId uint16, msg Message) {
	srv := s.getService(toId)
	srv.PushMsg(msg)
}

func (s *Sunnet) AddConn(types ConnType, fd int32, serviceId uint16) {
	conn := NewConn(types, fd, serviceId)
	s.ConnsLock.Lock()
	{
		s.Conns[fd] = conn
	}
	s.ConnsLock.Unlock()
	return
}

func (s *Sunnet) GetConn(fd int32) (conn *Conn) {
	s.ConnsLock.RLock()
	{
		conn = s.Conns[fd]
	}
	s.ConnsLock.RUnlock()
	return
}

func (s *Sunnet) RemoveConn(fd int32) {
	s.ConnsLock.Lock()
	{
		delete(s.Conns, fd)
	}
	s.ConnsLock.Unlock()
	return
}

// DONE: 启动网络服务
func (s *Sunnet) Listen(ipAddr string, port, serviceId uint16) {
	// step1: 创建socket
	listenFd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		fmt.Println("created socket failed, err:", err.Error())
		return
	}
	// step2: 设置为非阻塞
	if err := unix.SetNonblock(listenFd, true); err != nil {
		fmt.Println("set nonblock err:", err.Error())
		return
	}
	// step3: bind
	//ip地址转换
	var addr [4]byte
	copy(addr[:], net.ParseIP(ipAddr).To4())
	net.ParseIP(ipAddr).To4()
	if err := unix.Bind(listenFd, &unix.SockaddrInet4{Addr: addr, Port: int(port)}); err != nil {
		fmt.Println("socket bind err", err.Error())
		return
	}
	// step4: listen
	if err := unix.Listen(listenFd, 64); err != nil {
		fmt.Println("listen err", err.Error())
		return
	}
	// step5: 添加到Conns
	Inst.AddConn(LISTEN, int32(listenFd), serviceId)
	// step6: 添加到epoll
	socketWorker.addEvent(int32(listenFd))
	return
}

// DONE: 关闭Conn对象
func (s *Sunnet) CloseConn(fd int32) {
	// 删除conn对象
	s.RemoveConn(fd)
	// 删除epoll对象对套接字的监听
	socketWorker.removeEvent(fd)
	// 关闭套接字
	err := unix.Close(int(fd))
	if err != nil {
		fmt.Println("close socket failed:", err.Error())
	}
	return
}