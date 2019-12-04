package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"audotsp/term"
	"audotsp/utils"

	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"net/http"
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

type DevPage struct {
	Page int `form:"page" json:"page"  binding:"required"`
}

type DevPageItem struct {
	Ip    string `json:"ip"`
	Imei  string `json:"imei"`
	Phone string `json:"phone"`
}

type DevPageList struct {
	PageCnt   int           `json:"pagecnt"`
	PageSize  int           `json:"pagesize"`
	PageIndex int           `json:"pageindex"`
	Data      []DevPageItem `json:"data"`
}

//var connList []net.Conn
var connManger map[string]*term.Terminal

var ipaddress string
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
		Conn:   conn,
		Engine: engine,
	}
	term.Conn = conn
	connManger[addr.String()] = term
	ipaddress = addr.String()

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
	fmt.Printf("inputHandler.\n")

	rTermCtrl, _ := regexp.Compile("^ctrl [0-9]{1,3}")
	rList, _ := regexp.Compile("^ls")
	rUpdate, _ := regexp.Compile("^update ")

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

		} else if rList.MatchString(str) {
			for key, value := range connManger {
				fmt.Println("ip:", key, " ,imei:", value.GetImei(), " iccid:", value.GetIccid())
			}
		} else if rUpdate.MatchString(str) {
			parList := strings.SplitN(str, " ", 2)

			for _, val := range parList {
				fmt.Println(val)
			}

			if len(parList) == 2 {
				updateHandler(parList[1])
			}
		}
	}
}

func readFull(rd *bufio.Reader, buff []byte) (int, error) {
	var pos int = 0
	var err error
	for {
		var n int = 0
		if n, err = rd.Read(buff[pos:]); err == nil {
			fmt.Println("The number of bytes read:" + strconv.Itoa(n))
			pos += n
			if pos >= len(buff) {
				break
			}

		} else {
			fmt.Println("read end")
			break
		}
	}

	if pos > 0 {
		err = nil
	}
	return pos, err
}

func updateHandler(name string) {
	var verstr string = "v1.0.0"
	var tempTerm *term.Terminal = connManger[ipaddress]
	//vbyte := []byte{verstr}
	if fileObj, err := os.Open(name); err == nil {
		defer fileObj.Close()

		info, _ := fileObj.Stat()
		var filesize uint32 = uint32(info.Size())

		var sumcnt uint16 = 0
		//var reqlen uint32 = uint32(1 + 5 + 1 + len(verstr) + 4)
		//if (uint32(reqlen)+filesize)%1024 == 0 {
		//	sumcnt = (reqlen + filesize) / 1024
		//} else {
		//	sumcnt = (reqlen+filesize)/1024 + 1
		//}

		var curCnt uint16 = 1

		//一个文件对象本身是实现了io.Reader的 使用bufio.NewReader去初始化一个Reader对象，存在buffer中的，读取一次就会被清空
		reader := bufio.NewReader(fileObj)
		//读取Reader对象中的内容到[]byte类型的buf中

		var buf []byte
		for {
			if curCnt == 1 {
				buf = append(buf, 0)
				buf = append(buf, make([]byte, 5)...)
				buf = append(buf, byte(len(verstr)))
				buf = append(buf, verstr...)
				buf = append(buf, utils.Dword2Bytes(filesize)...)

				templen := len(buf)
				buf = append(buf, make([]byte, 1023-templen)...)

				if (uint32(templen)+filesize)%1023 == 0 {
					sumcnt = uint16((uint32(templen) + filesize) / 1023)
				} else {
					sumcnt = uint16((uint32(templen)+filesize)/1023 + 1)
				}

				if n, err := readFull(reader, buf[templen:]); err == nil {
					fmt.Println("The number of bytes read:" + strconv.Itoa(n))
					//这里的buf是一个[]byte，因此如果需要只输出内容，仍然需要将文件内容的换行符替换掉
					fmt.Println("data[", curCnt, "/", sumcnt, "]:", buf)

					fmt.Println("phone:", tempTerm.GetPhone())
					retbuf := tempTerm.MakeFrameMult(term.UpdateReq, 1, tempTerm.GetPhone(), 1, sumcnt, curCnt, buf)
					tempTerm.Conn.Write(retbuf)
					var outlog string = ""
					for _, val := range retbuf {
						outlog += fmt.Sprintf("%02X", val)
					}
					fmt.Println("---> ", outlog)

					curCnt++
					time.Sleep(5000 * time.Millisecond)

				} else {
					fmt.Println("read end")
					break
				}
			} else {
				buf = make([]byte, 1023)
				if n, err := readFull(reader, buf); err == nil {
					fmt.Println("The number of bytes read:" + strconv.Itoa(n))
					//这里的buf是一个[]byte，因此如果需要只输出内容，仍然需要将文件内容的换行符替换掉
					fmt.Println("data[", curCnt, "/", sumcnt, "]:", buf)

					fmt.Println("phone:", tempTerm.GetPhone())
					retbuf := tempTerm.MakeFrameMult(term.UpdateReq, 1, tempTerm.GetPhone(), 1, sumcnt, curCnt, buf)
					tempTerm.Conn.Write(retbuf)
					var outlog string = ""
					for _, val := range retbuf {
						outlog += fmt.Sprintf("%02X", val)
					}
					fmt.Println("---> ", outlog)

					curCnt++
					time.Sleep(5000 * time.Millisecond)

				} else {
					fmt.Println("read end")
					break
				}
			}
		}
	}
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

	address := ":" + port
	fmt.Println("address port ", address)

	listenSock, err := net.Listen("tcp", address)
	checkError(err)
	defer listenSock.Close()

	connManger = make(map[string]*term.Terminal)

	go inputHandler()

	go httpServer()

	for {
		newConn, err := listenSock.Accept()
		if err != nil {
			continue
		}

		go recvConnMsg(newConn)
	}

}

func httpServer() {
	router := gin.Default()

	router.POST("/api/list", listHandler)

	router.Run(":8080")
}

//获取在线设备
func listHandler(c *gin.Context) {
	fmt.Println("DevPage post")
	var json DevPage
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("page:", json)

	var devpagelist DevPageList
	devpagelist.PageSize = 10
	devpagelist.PageCnt = (len(connManger) + (devpagelist.PageSize - 1)) / devpagelist.PageSize
	devpagelist.PageIndex = json.Page

	if devpagelist.PageIndex > devpagelist.PageSize {
		devpagelist.PageIndex = devpagelist.PageSize
	}

	datalist := make([]DevPageItem, 0)
	var index int = 0
	for _, val := range connManger {
		if index >= (devpagelist.PageIndex-1)*devpagelist.PageSize && index < devpagelist.PageIndex*devpagelist.PageSize {
			var item DevPageItem
			item.Ip = val.Conn.RemoteAddr().String()
			item.Imei = val.GetImei()
			item.Phone = utils.HexBuffToString(val.GetPhone())
			datalist = append(datalist, item)
		}
		index++
	}
	devpagelist.Data = datalist
	c.JSON(http.StatusOK, devpagelist)
}
