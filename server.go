package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"audotsp/term"

	"regexp"
	"strconv"
	"strings"
	"bufio"
)

//var connList []net.Conn
var connManger map[string]term.Terminal

var port string

func checkError(err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func recvConnMsg(conn net.Conn) {
	var term term.Terminal
	buf := make([]byte, 0)
	addr := conn.RemoteAddr()
	fmt.Println(addr.Network())
	fmt.Println(addr.String())

	term.Conn = &conn
	connManger[addr.String()]=term

	defer func(){
		delete(connManger,addr.String())
		conn.Close()
	}()

	for {
		tempbuf := make([]byte, 1024)
		n, err := conn.Read(tempbuf)

		if err != nil {
			fmt.Println(addr.Network() + ":" + addr.String() + " is closed.")
			return
		}

		fmt.Printf("rcv frm[%d].\n", n)
		buf = append(buf, tempbuf[:n]...)
		fmt.Printf("<----")
		for _, val := range buf {
			fmt.Printf("%02X ", val)
		}
		fmt.Printf("\n")
		frmlen := term.DataFilter(buf)
		if frmlen == -2 {
			buf = make([]byte, 0)
		} else if frmlen > 0 {
			sendBuf := term.FrameHandle(buf)
			if sendBuf != nil {
				fmt.Printf("---->")
				for _, val := range sendBuf {
					fmt.Printf("%02X ", val)
				}
				fmt.Printf("\n")
				conn.Write(sendBuf)
			}

			buf = make([]byte, 0)
		}
	}
}

func inputHandler() {
	//var buffer [512]byte
	fmt.Printf("inputHandler.\n")
	
	rTermCtrl, _ := regexp.Compile("^ctrl [0-9]{1,3}")
	rList, _ := regexp.Compile("^ls")

	for {
		fmt.Printf("cmd> ")
		var inputReader *bufio.Reader
		inputReader = bufio.NewReader(os.Stdin)
		str, err := inputReader.ReadString('\n')

		templist := strings.SplitN(str, "\n", 2)
		str = templist[0]

		if err != nil {
			fmt.Println("read error:", err)
			continue
		}

		if rTermCtrl.MatchString(str) {
			parList := strings.SplitN(str, " ", 2)

			for _, val := range parList {
				fmt.Println(val)
			}

			if len(parList) == 2 {
				fmt.Println("len p1 ", len(parList[1]))
				cmdid, _ := strconv.Atoi(parList[1])
				fmt.Println("terminal ctrl:", cmdid)

				apduBuff := make([]byte, 0)
				switch cmdid {
				case 0x01:
					apduBuff = append(apduBuff, 0x01)
					apduBuff = append(apduBuff, 0x01)
					apduBuff = append(apduBuff, 0x01)
					apduBuff = append(apduBuff, 0x01)

					apduBuff = append(apduBuff, 0x02)

					urlbytes := []byte("UF 0,00000103,http://kingdom-tech.f3322.net:4000/TboxApp_Aoduo_20190925.tar.gz")

					apduBuff = append(apduBuff, urlbytes...)

				case 0x02:
					apduBuff = append(apduBuff, 0x02)
				case 0x03:
					apduBuff = append(apduBuff, 0x03)
				case 0x04:
					apduBuff = append(apduBuff, 0x04)
				case 0x05:
					apduBuff = append(apduBuff, 0x05)
				case 0x06:
					apduBuff = append(apduBuff, 0x06)

					apduBuff = append(apduBuff, byte(2))
					apduBuff = append(apduBuff, byte(0))
				case 0x80:
					apduBuff = append(apduBuff, 0x80)
				}

				//sendBuf := tm.MakeFrame(0x82,0,apduBuff)
				//for _, val := range sendBuf {
				//	fmt.Printf("%02X ", val)
				//}
				//fmt.Printf("\n")
				//(*tm.Conn).Write(sendBuf)
			}
		}else if rList.MatchString(str) {
			for key,value:=range connManger {
				fmt.Println("ip:",key," ,imei:",value.GetImei()," iccid:",value.GetIccid())
			}
		}
	}
	//fmt.Println("count:", n, ", msg:", string(buffer[:]))
}

func main() {
	flag.StringVar(&port, "port", "11229", "server port")
	flag.Parse()

	address := ":" + port
	fmt.Println("address port ", address)

	listenSock, err := net.Listen("tcp", address)
	checkError(err)
	defer listenSock.Close()

	connManger = make(map[string]term.Terminal)

	go inputHandler()

	for {
		newConn, err := listenSock.Accept()
		if err != nil {
			continue
		}

		go recvConnMsg(newConn)
	}

}
