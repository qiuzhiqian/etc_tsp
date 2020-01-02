package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tsp/term"
	"tsp/utils"

	"bufio"
	"strconv"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"

	"net/http"

	"crypto/md5"

	"github.com/BurntSushi/toml"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"tsp/proto"
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

type Config struct {
	TcpCfg TcpConfig `toml:"tcp"`
	WebCfg WebConfig `toml:"web"`
	MapCfg MapConfig `toml:"map"`
	PgCfg  PgConfig  `toml:"postgresql"`
}

type TcpConfig struct {
	Ip   string
	Port int
}

type WebConfig struct {
	Ip   string
	Port int
}

type MapConfig struct {
	AppKey string
}

type PgConfig struct {
	Hostname  string
	Tablename string
	User      string
	Password  string
}

//var connList []net.Conn
var connManger map[string]*term.Terminal

var ipaddress string

var engine *xorm.Engine

var config Config

func recvConnMsg(conn net.Conn) {
	buf := make([]byte, 0)
	addr := conn.RemoteAddr()
	log.WithFields(logrus.Fields{"network": addr.Network(), "ip": addr.String()}).Info("recv")

	var t *term.Terminal = &term.Terminal{
		Conn:   conn,
		Engine: engine,
		Ch:     make(chan int),
	}
	connManger[addr.String()] = t
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

		var msg []proto.Message
		var lens int
		msg, lens, err = proto.Filter(buf)
		if err != nil {
			//
		}

		buf = buf[lens:]

		for len(msg) > 0 {
			//处理消息
			sendBuf := t.Handler(msg[0])

			if sendBuf != nil {
				outlog = ""
				for _, val := range sendBuf {
					outlog += fmt.Sprintf("%02X", val)
				}
				log.WithFields(logrus.Fields{"data": outlog}).Info("---> ")

				logframe := &LogFrame{
					Stamp: time.Now(),
					Dir:   1,
					Frame: outlog,
				}
				_, err = engine.Insert(logframe)
				if err != nil {
					log.WithFields(logrus.Fields{"error": err.Error()}).Info("insert")
				}

				conn.Write(sendBuf)

				msg = msg[1:]
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

func main() {
	var err error
	logInit()

	curpath := GetCurrentDirectory()
	_, err = toml.DecodeFile(curpath+"/config.toml", &config)
	if err != nil {
		log.Info("config error: ", err)
		return
	}
	fmt.Println(config)

	connStr := "postgres://" + config.PgCfg.User + ":" + config.PgCfg.Password + "@" + config.PgCfg.Hostname + "/" + config.PgCfg.Tablename + "?sslmode=require"
	engine, err = xormInit("postgres", connStr)
	if err != nil {
		log.Info("xorm init error: ", err)
	}

	address := config.TcpCfg.Ip + ":" + strconv.FormatInt(int64(config.TcpCfg.Port), 10)
	log.Info("address port ", address)

	listenSock, err := net.Listen("tcp", address)
	if err != nil {
		log.WithFields(logrus.Fields{"Error:": err.Error()}).Error("check")
		os.Exit(1)
	}

	defer listenSock.Close()

	connManger = make(map[string]*term.Terminal)

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

	gpsdata := new(term.GPSData)
	err = engine.Sync2(gpsdata)
	if err != nil {
		return engine, err
	}

	devinfo := new(term.DevInfo)
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

	v1 := router.Group("/api/v1")
	{
		v1.POST("list", listHandler)
		v1.POST("data", dataHandler)
		v1.POST("nowgps", nowGpsHandler)
		v1.POST("gpsmap", gpsMapHandler)
		v1.POST("login", loginHandler)
		v1.POST("config", configHandler)
		v1.POST("control", controlHandler)
		v1.POST("userlist", userListHandler)
		v1.POST("useradd", userAddHandler)
	}

	router.StaticFS("/css", http.Dir("frontend/dist/css"))
	router.StaticFS("/fonts", http.Dir("frontend/dist/fonts"))
	router.StaticFS("/img", http.Dir("frontend/dist/img"))
	router.StaticFS("/js", http.Dir("frontend/dist/js"))
	router.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	//router.LoadHTMLGlob("templates/*")
	router.LoadHTMLFiles("frontend/dist/index.html")
	router.GET("/", mainPage)

	router.NoRoute(mainPage)

	address := config.WebCfg.Ip + ":" + strconv.FormatInt(int64(config.WebCfg.Port), 10)
	log.Info("address port ", address)
	router.Run(address)
}

//主页面
func mainPage(c *gin.Context) {
	//log.Info("no route page")
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
	gpsdata := new(term.GPSData)
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

	datas := make([]term.GPSData, 0)
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
	gpsdata := new(term.GPSData)
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
	gpsmap := make([]term.GPSData, 0)
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

	type DataResp struct {
		MapAppKey string `json:"mapAppKey"`
	}
	var resp DataResp
	resp.MapAppKey = config.MapCfg.AppKey

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
			//
		case "factory":
			//
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

func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1) //将\替换成/
}
