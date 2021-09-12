package main

import (
	"encoding/binary"
	"fmt"
)

func main() {
	data := encoder(packet{
		Version:   22,
		Operation: 12,
		Sequence:  13,
		Body:      []byte("hello goim!"),
	})
	p:=decoder(data)
	fmt.Printf("p:%+v body:%s",p,p.Body)
}

/*
goim 协议结构
网络中一般使用大端序,本地机器使用小端序
*/

type packet struct {
	packetLen uint32  // 包长度，在数据流传输过程中，先写入整个包的长度，方便整个包的数据读取。
	headerLen uint16  // 头长度，在处理数据时，会先解析头部，可以知道具体业务操作。
	Version   uint16  // 协议版本号，主要用于上行和下行数据包按版本号进行解析,感觉两个字节太长了,版本号有那么多吗?
	Operation uint32  // 4bytes 业务操作码，可以按操作码进行分发数据包到具体业务当中。
	Sequence  uint32  // 序列号，数据包的唯一标记，可以做具体业务处理，或者数据包去重。
	Body      []byte // 实际业务数据，在业务层中会进行数据解码和编码。
}

func decoder(data []byte)(p packet) {
	if len(data) <= 16 {
		fmt.Println("data len < 16.")
		return
	}

	p.packetLen = binary.BigEndian.Uint32(data[:4])

	p.headerLen = binary.BigEndian.Uint16(data[4:6])

	p.Version = binary.BigEndian.Uint16(data[6:8])

	p.Operation = binary.BigEndian.Uint32(data[8:12])

	p.Sequence = binary.BigEndian.Uint32(data[12:16])

	p.Body = data[16:]
	return p
}

func encoder(p packet) []byte {
	headerLen := 16
	packetLen := len(p.Body) + headerLen
	ret := make([]byte, packetLen)

	binary.BigEndian.PutUint32(ret[:4], uint32(packetLen))
	binary.BigEndian.PutUint16(ret[4:6], uint16(headerLen))

	binary.BigEndian.PutUint16(ret[6:8], p.Version)
	binary.BigEndian.PutUint32(ret[8:12], p.Operation)
	binary.BigEndian.PutUint32(ret[12:16], p.Sequence)

	copy(ret[16:], p.Body)

	return ret
}
