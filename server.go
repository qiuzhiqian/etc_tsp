package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"tsp/utils"

	"bufio"
	"strconv"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"

	"net/http"

	"crypto/md5"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var jwtSecKey []byte = []byte("asdsaf452g45aert")

var log = logrus.New()

type jwtCustomClaims struct {
	jwt.StandardClaims

	// 追加自己需要的信息
	Uid   uint `json:"uid"`
	Admin bool `json:"admin"`
}

type Users struct {
	Id       int       `xorm:"pk autoincr notnull id"`
	Name     string    `xorm:"name"`
	Password string    `xorm:"password"`
	IsAdmin  bool      `xorm:"admin"`
	Stamp    time.Time `xorm:"stamp"`
}

type LogFrame struct {
	Id    int       `xorm:"pk autoincr notnull id"`
	Stamp time.Time `xorm:"DateTime notnull 'stamp'"`
	Dir   int       `xorm:"dir"`
	Frame string    `xorm:"Varchar(2048) frame"`
}

type DevPage struct {
	Page int `form:"page" json:"page"  binding:"required"`
}

type GPSData struct {
	Imei      string    `xorm:"pk notnull imei`
	Stamp     time.Time `xorm:"DateTime pk notnull stamp`
	WarnFlag  uint32    `xorm:"warnflag"`
	State     uint32    `xorm:"state"`
	AccState  uint8     `xorm:"accstate"`
	GpsState  uint8     `xorm:"gpsstate"`
	Latitude  uint32    `xorm:"latitude"`
	Longitude uint32    `xorm:"longitude"`
	Altitude  uint16    `xorm:"altitude"`
	Speed     uint16    `xorm:"speed"`
	Direction uint16    `xorm:"direction"`
}

func (d GPSData) TableName() string {
	return "gps_data"
}

type DevInfo struct {
	Authkey    string `xorm:"auth_key"`
	Imei       string `xorm:"imei"`
	Vin        string `xorm:"vin"`
	PhoneNum   string `xorm:"pk notnull phone_num"`
	ProvId     uint16 `xorm:"prov_id"`
	CityId     uint16 `xorm:"city_id"`
	Manuf      string `xorm:"manuf"`
	TermType   string `xorm:"term_type"`
	TermId     string `xorm:"term_id"`
	PlateColor int    `xorm:"plate_color"`
	PlateNum   string `xorm:"plate_num"`
}

//var connList []net.Conn
var connManger map[string]*Terminal

var ipaddress string
var port string

var engine *xorm.Engine

var ch chan int

func checkError(err error) {
	if err != nil {
		log.WithFields(logrus.Fields{"Error:": err.Error()}).Error("check")
		os.Exit(1)
	}
}

func recvConnMsg(conn net.Conn) {
	buf := make([]byte, 0)
	addr := conn.RemoteAddr()
	log.WithFields(logrus.Fields{"network": addr.Network(), "ip": addr.String()}).Info("recv")

	var term *Terminal = &Terminal{
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
			log.WithFields(logrus.Fields{"network": addr.Network(), "ip": addr.String()}).Info("closed")
			return
		}

		buf = append(buf, tempbuf[:n]...)
		var outlog string
		for _, val := range buf {
			outlog += fmt.Sprintf("%02X", val)
		}
		log.WithFields(logrus.Fields{"data": outlog}).Info("<--- ")

		logframe := new(LogFrame)
		logframe.Stamp = time.Now()
		logframe.Dir = 0
		logframe.Frame = outlog
		_, err = engine.Insert(logframe)
		if err != nil {
			log.WithFields(logrus.Fields{"error": err.Error()}).Info("insert")
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
				log.WithFields(logrus.Fields{"data": outlog}).Info("---> ")

				logframe := new(LogFrame)
				logframe.Stamp = time.Now()
				logframe.Dir = 1
				logframe.Frame = outlog
				_, err = engine.Insert(logframe)
				if err != nil {
					log.WithFields(logrus.Fields{"error": err.Error()}).Info("insert")
				}

				conn.Write(sendBuf)
			}

			buf = make([]byte, 0)
		}
	}
}

func readFull(rd *bufio.Reader, buff []byte) (int, error) {
	var pos int = 0
	var err error
	for {
		var n int = 0
		if n, err = rd.Read(buff[pos:]); err == nil {
			log.Info("The number of bytes read:" + strconv.Itoa(n))
			pos += n
			if pos >= len(buff) {
				break
			}

		} else {
			log.Info("read end")
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
	var tempTerm *Terminal = connManger[ipaddress]
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
					log.Info("The number of bytes read:" + strconv.Itoa(n))
					//这里的buf是一个[]byte，因此如果需要只输出内容，仍然需要将文件内容的换行符替换掉
					log.Info("data[", curCnt, "/", sumcnt, "]:", buf)

					log.Info("phone:", tempTerm.GetPhone())
					retbuf := tempTerm.MakeFrameMult(UpdateReq, 1, tempTerm.GetPhone(), 1, sumcnt, curCnt, buf)
					tempTerm.Conn.Write(retbuf)
					var outlog string = ""
					for _, val := range retbuf {
						outlog += fmt.Sprintf("%02X", val)
					}
					log.Info("---> ", outlog)

					curCnt++
					time.Sleep(5000 * time.Millisecond)

				} else {
					log.Info("read end")
					break
				}
			} else {
				buf = make([]byte, 1023)
				if n, err := readFull(reader, buf); err == nil {
					log.Info("The number of bytes read:" + strconv.Itoa(n))
					//这里的buf是一个[]byte，因此如果需要只输出内容，仍然需要将文件内容的换行符替换掉
					log.Info("data[", curCnt, "/", sumcnt, "]:", buf)

					log.Info("phone:", tempTerm.GetPhone())
					retbuf := tempTerm.MakeFrameMult(UpdateReq, 1, tempTerm.GetPhone(), 1, sumcnt, curCnt, buf)
					tempTerm.Conn.Write(retbuf)
					var outlog string = ""
					for _, val := range retbuf {
						outlog += fmt.Sprintf("%02X", val)
					}
					log.Info("---> ", outlog)

					curCnt++
					//time.Sleep(5000 * time.Millisecond)
					<-ch

				} else {
					log.Info("read end")
					break
				}
			}
		}
	}
}

func main() {
	flag.StringVar(&port, "port", "19902", "server port")
	flag.Parse()

	logInit()

	teststr := "00001234526"
	teststr = strings.TrimLeft(teststr, "0")
	log.Info("test:", teststr)

	var err error
	engine, err = xormInit("postgres", "postgres://pqgotest:pqgotest@localhost/pqgodb?sslmode=require")
	if err != nil {
		log.Info("xorm init error: ", err)
	}

	address := ":" + port
	log.Info("address port ", address)

	listenSock, err := net.Listen("tcp", address)
	checkError(err)
	defer listenSock.Close()

	connManger = make(map[string]*Terminal)

	ch = make(chan int)

	go httpServer()

	for {
		newConn, err := listenSock.Accept()
		if err != nil {
			continue
		}

		go recvConnMsg(newConn)
	}

}

func xormInit(driverName string, dataSourceName string) (*xorm.Engine, error) {
	var err error
	engine, err = xorm.NewEngine(driverName, dataSourceName)
	if err != nil {
		return engine, err
	}

	users := new(Users)
	err = engine.Sync2(users)
	if err != nil {
		return engine, err
	}

	logframe := new(LogFrame)
	err = engine.Sync2(logframe)
	if err != nil {
		return engine, err
	}

	gpsdata := new(GPSData)
	err = engine.Sync2(gpsdata)
	if err != nil {
		return engine, err
	}

	devinfo := new(DevInfo)
	err = engine.Sync2(devinfo)
	if err != nil {
		return engine, err
	}
	return engine, err
}

func logInit() {
	// The API for setting attributes is a little different than the package level
	// exported logger. See Godoc.
	log.Out = os.Stdout

	// You could set this to any `io.Writer` such as a file
	// file, err := os.OpenFile("logrus.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err == nil {
	//  log.Out = file
	// } else {
	//  log.Info("Failed to log to file, using default stderr")
	// }

	log.Formatter = &logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	}
}

func httpServer() {
	router := gin.Default()

	router.POST("/api/list", listHandler)
	router.POST("/api/data", dataHandler)
	router.POST("/api/nowgps", nowGpsHandler)
	router.POST("/api/gpsmap", gpsMapHandler)
	router.POST("/api/login", loginHandler)
	router.POST("/api/config", configHandler)
	router.POST("/api/control", controlHandler)
	router.POST("/api/userlist", userListHandler)
	router.POST("/api/useradd", userAddHandler)

	router.StaticFS("/css", http.Dir("frontend/dist/css"))
	router.StaticFS("/fonts", http.Dir("frontend/dist/fonts"))
	router.StaticFS("/img", http.Dir("frontend/dist/img"))
	router.StaticFS("/js", http.Dir("frontend/dist/js"))
	router.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	//router.LoadHTMLGlob("templates/*")
	router.LoadHTMLFiles("frontend/dist/index.html")
	router.GET("/", mainPage)

	router.NoRoute(mainPage)

	router.Run(":8080")
}

//主页面
func mainPage(c *gin.Context) {
	log.Info("no route page")
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Main website",
	})
}

//获取在线设备
func listHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)
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

	log.Info("DevPage post")
	var json DevPage
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info("page:", json)

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
			item.Phone = strings.TrimLeft(utils.HexBuffToString(val.GetPhone()), "0")
			datalist = append(datalist, item)
		}
		index++
	}
	devpagelist.Data = datalist
	c.JSON(http.StatusOK, devpagelist)
}

//获取数据
func dataHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		Imei  string `json:"imei" binding:"required"`
		Start int64  `json:"starttime" binding:"required"`
		End   int64  `json:"endtime" binding:"required"`
		Page  int    `json:"page"`
	}
	var json DataReq
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if json.Page == 0 {
		json.Page = 1
	}

	//查找数据库
	type DataItem struct {
		Imei      string `json:"imei"`
		Stamp     int64  `json:"stamp"`
		WarnFlag  uint32 `json:"warnflag"`
		State     uint32 `json:"state"`
		Latitude  uint32 `json:"latitude"`
		Longitude uint32 `json:"longitude"`
		Altitude  uint16 `json:"altitude"`
		Speed     uint16 `json:"speed"`
		Direction uint16 `json:"direction"`
	}

	type DataResp struct {
		PageCnt   int        `json:"pagecnt"`
		PageSize  int        `json:"pagesize"`
		PageIndex int        `json:"pageindex"`
		Data      []DataItem `json:"data"`
	}

	//获取总数
	gpsdata := new(GPSData)
	total, err := engine.Where("imei = ? AND stamp > ? AND stamp < ?", json.Imei, time.Unix(json.Start, 0), time.Unix(json.End, 0)).Count(gpsdata)
	if err != nil {
		log.Info("where err:", err)
	}
	log.Info("select total:", total)

	var dataresp DataResp
	dataresp.PageSize = 10
	dataresp.PageCnt = ((int)(total) + (dataresp.PageSize - 1)) / dataresp.PageSize
	dataresp.PageIndex = json.Page

	log.Info("page:", json.Page)

	if dataresp.PageIndex > dataresp.PageCnt {
		dataresp.PageIndex = dataresp.PageCnt
	}

	datas := make([]GPSData, 0)
	startindex := (dataresp.PageIndex - 1) * dataresp.PageSize
	log.Info("start:", startindex)
	err = engine.Where("imei = ? AND stamp > ? AND stamp < ?", json.Imei, time.Unix(json.Start, 0), time.Unix(json.End, 0)).Limit(dataresp.PageSize, startindex).Find(&datas)
	if err != nil {
		log.Info("where err:", err)
	}

	datalist := make([]DataItem, 0)
	for _, val := range datas {
		var item DataItem
		item.Stamp = val.Stamp.Unix()
		item.Imei = val.Imei
		item.WarnFlag = val.WarnFlag
		item.State = val.State
		item.Latitude = val.Latitude
		item.Longitude = val.Longitude
		item.Altitude = val.Altitude
		item.Speed = val.Speed
		item.Direction = val.Direction
		datalist = append(datalist, item)
	}
	dataresp.Data = datalist

	c.JSON(http.StatusOK, dataresp)
}

//获取数据
func nowGpsHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		Imei string `json:"imei" binding:"required"`
	}
	var json DataReq
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//查找数据库
	type DataItem struct {
		Imei      string `json:"imei"`
		Stamp     int64  `json:"stamp"`
		WarnFlag  uint32 `json:"warnflag"`
		State     uint32 `json:"state"`
		Latitude  uint32 `json:"latitude"`
		Longitude uint32 `json:"longitude"`
		Altitude  uint16 `json:"altitude"`
		Speed     uint16 `json:"speed"`
		Direction uint16 `json:"direction"`
	}

	//获取总数
	gpsdata := new(GPSData)
	has, err := engine.Where("imei = ? AND state > 0", json.Imei).Desc("stamp").Limit(1).Get(gpsdata)
	if err != nil {
		log.Info("where err:", err)
	}
	log.Info("has:", has)

	var item DataItem
	item.Stamp = gpsdata.Stamp.Unix()
	item.Imei = gpsdata.Imei
	item.WarnFlag = gpsdata.WarnFlag
	item.State = gpsdata.State
	item.Latitude = gpsdata.Latitude
	item.Longitude = gpsdata.Longitude
	item.Altitude = gpsdata.Altitude
	item.Speed = gpsdata.Speed
	item.Direction = gpsdata.Direction

	c.JSON(http.StatusOK, item)
}

func gpsMapHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		Imei  string `json:"imei" binding:"required"`
		Start int64  `json:"starttime" binding:"required"`
		End   int64  `json:"endtime" binding:"required"`
	}
	var json DataReq
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//查找数据库
	type DataItem struct {
		Imei      string `json:"imei"`
		Stamp     int64  `json:"stamp"`
		Latitude  uint32 `json:"latitude"`
		Longitude uint32 `json:"longitude"`
		Altitude  uint16 `json:"altitude"`
		Speed     uint16 `json:"speed"`
		Direction uint16 `json:"direction"`
	}

	//获取总数
	gpsmap := make([]GPSData, 0)
	err = engine.Where("imei = ? AND stamp > ? AND stamp < ? AND gpsstate = ?", json.Imei, time.Unix(json.Start, 0), time.Unix(json.End, 0), 1).Asc("stamp").Find(&gpsmap)
	if err != nil {
		log.Info("where err:", err)
	}

	datalist := make([]DataItem, 0)
	for _, val := range gpsmap {
		var item DataItem
		item.Stamp = val.Stamp.Unix()
		item.Imei = val.Imei
		item.Latitude = val.Latitude
		item.Longitude = val.Longitude
		item.Altitude = val.Altitude
		item.Speed = val.Speed
		item.Direction = val.Direction
		datalist = append(datalist, item)
	}

	c.JSON(http.StatusOK, datalist)
}

func loginHandler(c *gin.Context) {
	log.Info("login post")
	type DataReq struct {
		User     string `json:"user" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var json DataReq
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//查找数据库
	//该用户存在
	if json.User != "admin" || md5.Sum([]byte(json.Password)) != md5.Sum([]byte("admin123456")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usrname or password is error."})
		return
	}
	//后面集成jwt
	type DataResp struct {
		Token string `json:"token"`
	}
	var resp DataResp
	var tokenstr string
	var err error
	tokenstr, err = CreateToken(jwtSecKey, json.User, 3321231, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp.Token = tokenstr

	c.JSON(http.StatusOK, resp)
}

func configHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		User     string `json:"user" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var json DataReq
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//查找数据库
	//该用户存在
	//后面集成jwt
	type DataResp struct {
		Token string `json:"token"`
	}
	var resp DataResp
	resp.Token = "xdfasZsdfa2DsJsfa2"

	c.JSON(http.StatusOK, resp)
}

func controlHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		Imei  string `json:"imei" binding:"required"`
		Cmd   string `json:"cmd" binding:"required"`
		Param string `json:"param"`
	}
	var json DataReq
	if err = c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	termip := ""
	for key, val := range connManger {
		tempimei := val.GetImei()
		if tempimei == json.Imei {
			termip = key
		}
	}

	log.Info("ip:", termip)

	if termip != "" {

		switch json.Cmd {
		case "reset":
			buf := connManger[termip].makeApduCtrl(4, "")
			retbuf := connManger[termip].MakeFrame(CtrlReq, 1, connManger[termip].GetPhone(), 1, buf)
			connManger[termip].Conn.Write(retbuf)
		case "factory":
			buf := connManger[termip].makeApduCtrl(5, "")
			retbuf := connManger[termip].MakeFrame(CtrlReq, 1, connManger[termip].GetPhone(), 1, buf)
			connManger[termip].Conn.Write(retbuf)
		}

	}

	c.JSON(http.StatusOK, gin.H{"status": 0})
}

func userListHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		Page int `json:"page" binding:"required"`
	}
	var json DataReq
	if err = c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if json.Page == 0 {
		json.Page = 1
	}

	//查找数据库
	type DataItem struct {
		User       string `json:"user"`
		Role       string `json:"role"`
		CreateTime int64  `json:"createTime"`
	}

	type DataResp struct {
		PageCnt   int        `json:"pagecnt"`
		PageSize  int        `json:"pagesize"`
		PageIndex int        `json:"pageindex"`
		Data      []DataItem `json:"data"`
	}

	//获取总数
	usersTemp := new(Users)
	total, err := engine.Count(usersTemp)
	if err != nil {
		log.Info("where err:", err)
	}
	log.Info("select total:", total)

	var dataresp DataResp
	dataresp.PageSize = 10
	dataresp.PageCnt = ((int)(total) + (dataresp.PageSize - 1)) / dataresp.PageSize
	dataresp.PageIndex = json.Page

	log.Info("page:", json.Page)

	if dataresp.PageIndex > dataresp.PageCnt {
		dataresp.PageIndex = dataresp.PageCnt
	}

	datas := make([]Users, 0)
	startindex := (dataresp.PageIndex - 1) * dataresp.PageSize
	log.Info("start:", startindex)
	err = engine.Find(&datas)
	if err != nil {
		log.Info("where err:", err)
	}

	datalist := make([]DataItem, 0)
	for _, val := range datas {
		var rolestr string = ""
		if val.IsAdmin {
			rolestr = "Admin"
		} else {
			rolestr = "User"
		}
		var item DataItem
		item.User = val.Name
		item.Role = rolestr
		item.CreateTime = val.Stamp.Unix()
		log.WithFields(logrus.Fields{"item.Role": item.Role}).Info("userlist")
		datalist = append(datalist, item)
	}
	dataresp.Data = datalist

	c.JSON(http.StatusOK, dataresp)
}

func userAddHandler(c *gin.Context) {
	tokenstr := c.GetHeader("Authorization")
	if tokenstr == "" {
		//说明没有token
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	cliams, err := ParseToken(tokenstr, jwtSecKey)
	if err != nil {
		//返回401
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	log.Info("cliams:", cliams)

	type DataReq struct {
		User     string `json:"user"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"admin"`
	}
	var json DataReq
	if err = c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	md5Array := md5.Sum([]byte(json.Password))
	var user Users = Users{
		Name:     json.User,
		Password: utils.HexBuffToString(md5Array[:]),
		IsAdmin:  json.IsAdmin,
		Stamp:    time.Now(),
	}
	var isexist bool
	isexist, err = engine.Exist(&Users{
		Name: json.User,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	if isexist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user is exist"})
	}
	engine.Insert(&user)

	c.JSON(http.StatusOK, gin.H{"status": 0})
}

func CreateToken(SecretKey []byte, issuer string, Uid uint, isAdmin bool) (tokenString string, err error) {
	claims := &jwtCustomClaims{
		jwt.StandardClaims{
			ExpiresAt: int64(time.Now().Add(time.Hour * 72).Unix()),
			Issuer:    issuer,
		},
		Uid,
		isAdmin,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(SecretKey)
	return
}

func ParseToken(tokenSrt string, SecretKey []byte) (claims jwt.Claims, err error) {
	var token *jwt.Token
	token, err = jwt.Parse(tokenSrt, func(*jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	claims = token.Claims
	return
}
