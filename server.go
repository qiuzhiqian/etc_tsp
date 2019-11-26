package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"audotsp/term"

	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
)

type Users struct {
	Id   int    `xorm:"pk autoincr notnull id"`
	Name string `xorm:"unique name"`
	Age  int    `xorm:"age"`
}

type LogFrame struct {
	Id        int    `xorm:"pk autoincr notnull id"`
	Timestamp int64  `xorm:"BigInt notnull 'timestamp'"`
	Dir       int    `xorm:"dir"`
	Frame     string `xorm:"Varchar(2048) frame"`
}

type DevInfo struct {
	Authkey    string
	Imei       string
	Iccid      string
	Vin        string
	ProvId     uint16
	CityId     uint16
	Manuf      string
	TermType   string
	TermId     string
	PlateColor int
	PlateNum   string
}

func (d DevInfo) TableName() string {
	return "dev_info"
}

//var connList []net.Conn
var connManger map[string]*term.Terminal

var port string

var engine *xorm.Engine

func checkError(err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func recvConnMsg(conn net.Conn) {
	buf := make([]byte, 0)
	addr := conn.RemoteAddr()
	fmt.Println(addr.Network())
	fmt.Println(addr.String())

	var term *term.Terminal = &term.Terminal{
		Conn: conn,
	}
	term.Conn = conn
	connManger[addr.String()] = term

	defer func() {
		delete(connManger, addr.String())
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
		var outlog string
		for _, val := range buf {
			outlog += fmt.Sprintf("%02X", val)
		}
		fmt.Println("<--- ", outlog)

		logframe := new(LogFrame)
		engine.Sync2(logframe)
		logframe.Timestamp = time.Now().Unix()
		logframe.Dir = 0
		logframe.Frame = outlog
		_, err = engine.Insert(logframe)
		if err != nil {
			fmt.Println(err)
		}
		frmlen := term.DataFilter(buf)
		if frmlen == -2 {
			buf = make([]byte, 0)
		} else if frmlen > 0 {
			sendBuf := term.FrameHandle(buf)
			if sendBuf != nil {
				outlog = ""
				for _, val := range sendBuf {
					outlog += fmt.Sprintf("%02X", val)
				}
				fmt.Println("---> ", outlog)

				logframe := new(LogFrame)
				engine.Sync2(logframe)
				logframe.Timestamp = time.Now().Unix()
				logframe.Dir = 1
				logframe.Frame = outlog
				_, err = engine.Insert(logframe)
				if err != nil {
					fmt.Println(err)
				}

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
		} else if rList.MatchString(str) {
			for key, value := range connManger {
				fmt.Println("ip:", key, " ,imei:", value.GetImei(), " iccid:", value.GetIccid())
			}
		}
	}
	//fmt.Println("count:", n, ", msg:", string(buffer[:]))
}

func main() {
	flag.StringVar(&port, "port", "19901", "server port")
	flag.Parse()

	var err error
	engine, err = xorm.NewEngine("postgres", "postgres://pqgotest:pqgotest@localhost/pqgodb?sslmode=require")
	if err != nil {
		fmt.Println("new engine ", err)
	}

	users := new(Users)
	err = engine.Sync(users)
	if err != nil {
		fmt.Println("sync users ", err)
	}

	//users.Name = "xml"
	//users.Age = 29
	//_, err = engine.Insert(users)
	//if err != nil {
	//	fmt.Println(err)
	//}

	address := ":" + port
	fmt.Println("address port ", address)

	listenSock, err := net.Listen("tcp", address)
	checkError(err)
	defer listenSock.Close()

	connManger = make(map[string]*term.Terminal)

	go inputHandler()

	for {
		newConn, err := listenSock.Accept()
		if err != nil {
			continue
		}

		go recvConnMsg(newConn)
	}

}
