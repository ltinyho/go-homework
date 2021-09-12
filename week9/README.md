1. 总结几种 socket 粘包的解包方式: fix length/delimiter based/length field based frame decoder。尝试举例其应用
2. 实现一个从 socket connection 中解码出 goim 协议的解码器。




socket 的粘包只在TCP中,对于 UDP 是一个整包的概念.
TCP 是一个数据流的概念,没有包的分界点,需要上层应用自己去解析一个完整的包.
像对于 HTTP 协议来说,使用 Content-Length 来识别一个包的完整大小.

粘包的解包方式有以下几种:
- fix length : 发送方，每次发送固定长度的数据，并且不超过缓冲区，接受方每次按固定长度区接受数据
- delimiter based : 使用固定的标识符去标识包的结束,一般选择不太可能在包中发送的内容作为结束符. FTP协议，发邮件的 SMTP 
  协议，一个命令或者一段数据后面加上"\r\n"表示一个包的结束.一般用于一些包含各种命令控制的应用中
- length field based: 类似于 HTTP,使用一个packet length标识一个包的大小,发送端根据发送数据的大小写入到packet length 字段中



# goim解码器
[解码器代码](./main.go)
