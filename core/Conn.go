package core

// 类型枚举
type ConnType uint8

const (
	LISTEN ConnType = iota + 1
	CLIENT
)

type Conn struct {
	Type      ConnType
	Fd        int32
	ServiceId uint16
}

func NewConn(types ConnType, fd int32, serviceId uint16) *Conn {
	return &Conn{Type:types, Fd:fd, ServiceId:serviceId}
}