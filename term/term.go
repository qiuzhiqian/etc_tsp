package term

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"tsp/codec"
	"tsp/proto"
	"tsp/utils"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
)

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

func (d DevInfo) TableName() string {
	return "dev_info"
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
	DataStamp time.Time `xorm:"DateTime pk notnull datastamp`
}

func (d GPSData) TableName() string {
	return "gps_data"
}

type Terminal struct {
	authkey   string
	imei      string
	iccid     string
	vin       string
	tboxver   string
	loginTime time.Time
	seqNum    uint16
	phoneNum  []byte
	Conn      net.Conn
	Engine    *xorm.Engine
	Ch        chan int
}

func (t *Terminal) NewTerminal() {
	t.Ch = make(chan int)
}

type TermAckBody struct {
	AckSeqNum uint16
	AckID     uint16
	AckResult uint8
}

type PlatAckBody struct {
	AckSeqNum uint16
	AckID     uint16
	AckResult uint8
}

type RegisterBody struct {
	ProID         uint16
	CityID        uint16
	ManufID       []byte `len:"11"`
	TermType      []byte `len:"30"`
	TermID        []byte `len:"30"`
	LicPlateColor uint8
	LicPlate      string
}

type RegisterAckBody struct {
	AckSeqNum uint16
	AckResult uint8
	AuthKey   string
}

type AuthBody struct {
	AuthKeyLen uint8
	AuthKey    string
	Imei       []byte `len:"15"`
	Version    []byte `len:"20"`
}

type GPSInfoBody struct {
	WarnFlag uint32
	State    uint32
	Lat      uint32
	Lng      uint32
	Alt      uint16
	Speed    uint16
	Dir      uint16
	Time     []byte `len:"6"`
}

type CtrlBody struct {
	Cmd   uint8
	Param string
}

func (t *Terminal) SendCtrl(cmdid uint8, param string) error {
	var err error
	var body []byte
	body, err = codec.Marshal(&CtrlBody{
		Cmd:   cmdid,
		Param: param,
	})
	if err != nil {
		fmt.Println("err:", err)
	}

	msg := proto.Message{
		HEADER: proto.Header{
			MID:      proto.CtrlReq,
			Attr:     proto.MakeAttr(1, false, 0, uint16(len(body))),
			Version:  1,
			PhoneNum: string(t.phoneNum),
			SeqNum:   t.seqNum,
		},
		BODY: body,
	}
	sendbuff := proto.Packer(msg)
	t.Conn.Write(sendbuff)

	select {
	case num := <-t.Ch:
		fmt.Println("num:", num)
	case <-time.After(3 * time.Second):
		fmt.Println("timeout")
	}

	return nil
}

func (t *Terminal) GetImei() string {
	return t.imei
}

func (t *Terminal) GetIccid() string {
	return t.iccid
}

func (t *Terminal) GetPhone() []byte {
	return t.phoneNum
}

//Handler is proto Handler api
func (t *Terminal) Handler(msg proto.Message) []byte {
	if t.phoneNum == nil {
		t.phoneNum = make([]byte, 10)
	}

	copy(t.phoneNum, []byte(msg.HEADER.PhoneNum))
	t.seqNum = msg.HEADER.SeqNum

	switch msg.HEADER.MID {
	case proto.TermAck:
		reqID := codec.Bytes2Word(msg.BODY[2:4])
		if reqID == proto.UpdateReq {
			//ch <- 1
			//均级命令
		}
	case proto.Register:
		devinfo := new(DevInfo)

		devinfo.PhoneNum = strings.TrimLeft(utils.HexBuffToString(t.phoneNum), "0")

		is, _ := t.Engine.Get(devinfo)
		if !is {
			return []byte{}
		}

		var reg RegisterBody
		_, err := codec.Unmarshal(msg.BODY, &reg)
		if err != nil {
			fmt.Println("err:", err)
		}

		var body []byte
		body, err = codec.Marshal(&RegisterAckBody{
			AckSeqNum: msg.HEADER.SeqNum,
			AckResult: 0,
			AuthKey:   devinfo.Authkey,
		})
		if err != nil {
			fmt.Println("err:", err)
		}

		msgAck := proto.Message{
			HEADER: proto.Header{
				MID:      proto.RegisterAck,
				Attr:     proto.MakeAttr(1, false, 0, uint16(len(body))),
				Version:  1,
				PhoneNum: string(t.phoneNum),
				SeqNum:   t.seqNum,
			},
			BODY: body,
		}
		return proto.Packer(msgAck)
	case proto.Login:
		var auth AuthBody
		_, err := codec.Unmarshal(msg.BODY, &auth)
		if err != nil {
			fmt.Println("err:", err)
		}
		t.authkey = auth.AuthKey
		t.imei = string(auth.Imei)
		t.tboxver = string(auth.Version)

		var body []byte
		body, err = codec.Marshal(&PlatAckBody{
			AckSeqNum: msg.HEADER.SeqNum,
			AckID:     msg.HEADER.MID,
			AckResult: 0,
		})
		if err != nil {
			fmt.Println("err:", err)
		}

		msgAck := proto.Message{
			HEADER: proto.Header{
				MID:      proto.PlatAck,
				Attr:     proto.MakeAttr(1, false, 0, uint16(len(body))),
				Version:  1,
				PhoneNum: string(t.phoneNum),
				SeqNum:   t.seqNum,
			},
			BODY: body,
		}
		return proto.Packer(msgAck)
	case proto.Heartbeat:
		var err error
		var body []byte
		body, err = codec.Marshal(&PlatAckBody{
			AckSeqNum: msg.HEADER.SeqNum,
			AckID:     msg.HEADER.MID,
			AckResult: 0,
		})
		if err != nil {
			fmt.Println("err:", err)
		}

		msgAck := proto.Message{
			HEADER: proto.Header{
				MID:      proto.PlatAck,
				Attr:     proto.MakeAttr(1, false, 0, uint16(len(body))),
				Version:  1,
				PhoneNum: string(t.phoneNum),
				SeqNum:   t.seqNum,
			},
			BODY: body,
		}
		return proto.Packer(msgAck)
	case proto.Gpsinfo:
		var gpsInfo GPSInfoBody
		_, err := codec.Unmarshal(msg.BODY, &gpsInfo)
		if err != nil {
			fmt.Println("err:", err)
		}

		gpsdata := new(GPSData)
		gpsdata.Imei = t.imei
		gpsdata.Stamp = time.Now()
		gpsdata.WarnFlag = gpsInfo.WarnFlag
		gpsdata.State = gpsInfo.State
		gpsdata.Latitude = gpsInfo.Lat
		gpsdata.Longitude = gpsInfo.Lng

		gpsdata.Altitude = gpsInfo.Alt
		gpsdata.Speed = gpsInfo.Speed
		gpsdata.Direction = gpsInfo.Dir

		var year, month, day, hour, minute, second uint64
		year, err = strconv.ParseUint(strconv.FormatUint(uint64(gpsInfo.Time[0]), 16), 10, 8)
		month, err = strconv.ParseUint(strconv.FormatUint(uint64(gpsInfo.Time[1]), 16), 10, 8)
		day, err = strconv.ParseUint(strconv.FormatUint(uint64(gpsInfo.Time[2]), 16), 10, 8)
		hour, err = strconv.ParseUint(strconv.FormatUint(uint64(gpsInfo.Time[3]), 16), 10, 8)
		minute, err = strconv.ParseUint(strconv.FormatUint(uint64(gpsInfo.Time[4]), 16), 10, 8)
		second, err = strconv.ParseUint(strconv.FormatUint(uint64(gpsInfo.Time[5]), 16), 10, 8)

		gpsdata.DataStamp = time.Date(int(2000+year), time.Month(month), int(day), int(hour), int(minute), int(second), 0, time.Local)

		if (gpsdata.State & 0x00000001) > 0 {
			gpsdata.AccState = 1
		} else {
			gpsdata.AccState = 0
		}

		if (gpsdata.State & 0x00000002) > 0 {
			gpsdata.GpsState = 1
		} else {
			gpsdata.GpsState = 0
		}

		_, err = t.Engine.Insert(gpsdata)
		if err != nil {
			fmt.Println("insert gps err:", err)
		}

		var body []byte
		body, err = codec.Marshal(&PlatAckBody{
			AckSeqNum: msg.HEADER.SeqNum,
			AckID:     msg.HEADER.MID,
			AckResult: 0,
		})
		if err != nil {
			fmt.Println("err:", err)
		}

		msgAck := proto.Message{
			HEADER: proto.Header{
				MID:      proto.PlatAck,
				Attr:     proto.MakeAttr(1, false, 0, uint16(len(body))),
				Version:  1,
				PhoneNum: string(t.phoneNum),
				SeqNum:   t.seqNum,
			},
			BODY: body,
		}
		return proto.Packer(msgAck)
	}

	return nil
}
