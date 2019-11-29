package term

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"audotsp/utils"

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
}

const (
	protoHeader byte = 0x7e

	register    uint16 = 0x0100
	registerAck uint16 = 0x8100
	unregister  uint16 = 0x0003
	login       uint16 = 0x0102
	heartbeat   uint16 = 0x0002
	gpsinfo     uint16 = 0x0200
	platAck     uint16 = 0x8001
	UpdateReq   uint16 = 0x8108
)

func init() {
	fmt.Println("hello module init function")
}

func deepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func (t Terminal) MakeFrame(cmd uint16, ver uint8, phone []byte, seq uint16, apdu []byte) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(cmd)
	data = append(data, tempbytes...)
	datalen := uint16(len(apdu)) & 0x03FF
	//fmt.Println("datalen:", datalen, " len:", uint16(len(apdu)))
	datalen = datalen | 0x4000

	tempbytes = utils.Word2Bytes(datalen)
	data = append(data, tempbytes...)

	data = append(data, ver)

	data = append(data, phone...)

	tempbytes = utils.Word2Bytes(seq)
	data = append(data, tempbytes...)

	data = append(data, apdu...)

	csdata := byte(t.checkSum(data[:]))
	data = append(data, csdata)

	//转义

	//添加头尾
	var tmpdata []byte = []byte{0x7e}
	data = append(tmpdata, data...)
	data = append(data, 0x7e)

	return data
}

func (t Terminal) MakeFrameMult(cmd uint16, ver uint8, phone []byte, seq, sum, cur uint16, apdu []byte) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(cmd)
	data = append(data, tempbytes...)
	datalen := uint16(len(apdu)) & 0x03FF
	fmt.Println("datalen:", datalen, " len:", uint16(len(apdu)))
	datalen = datalen | 0x4000

	datalen = datalen | 0x2000
	tempbytes = utils.Word2Bytes(datalen)
	data = append(data, tempbytes...)

	data = append(data, ver)

	data = append(data, phone...)

	tempbytes = utils.Word2Bytes(seq)
	data = append(data, tempbytes...)

	tempbytes = utils.Word2Bytes(sum)
	data = append(data, tempbytes...)
	tempbytes = utils.Word2Bytes(cur)
	data = append(data, tempbytes...)

	data = append(data, apdu...)

	csdata := byte(t.checkSum(data[:]))
	data = append(data, csdata)

	//转义

	//添加头尾
	var tmpdata []byte = []byte{0x7e}
	data = append(tmpdata, data...)
	data = append(data, 0x7e)

	return data
}

func (t Terminal) makeApduRegisterAck(res uint8, authkey string) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(t.seqNum)
	data = append(data, tempbytes...)

	data = append(data, res)

	for _, item := range authkey {
		data = append(data, byte(item))
	}

	return data
}

func (t Terminal) makeApduCommonAck(cmdid uint16, res byte) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(t.seqNum)
	data = append(data, tempbytes...)

	tempbytes = utils.Word2Bytes(cmdid)
	data = append(data, tempbytes...)

	data = append(data, res)

	fmt.Println("apdu:", data)
	return data
}

func (t Terminal) DataFilter(data []byte) int {
	//--------------------------------------------------
	//int iRet = 0;
	// static int curLen=0;
	fmt.Printf("len = %d,data[0]=0x%X.\n", len(data), data[0])
	if data[0] == protoHeader {
		fmt.Println("find start.")
		var endindex int = -1
		for i := 1; i < len(data); i++ {
			if data[i] == protoHeader {
				fmt.Println("find end.")
				endindex = i
				break
			}
		}

		if endindex > 0 {
			data = data[:endindex+1]
		}

		return len(data)
	} else {
		return -2
	}
}

func (t Terminal) FrameHandle(data []byte) []byte {
	cmdid := utils.Bytes2Word(data[1:3])
	t.phoneNum = make([]byte, 10)
	deepCopy(t.phoneNum, data[6:6+10])
	t.seqNum = utils.Bytes2Word(data[16:18])
	fmt.Println("cmdid:", cmdid)
	len := len(data)
	return t.apduHandle(cmdid, data[18:len-2])
}

func (t Terminal) apduHandle(cmdType uint16, apdu []byte) []byte {
	switch cmdType {
	case register:
		fmt.Println("rcv register.")

		devinfo := new(DevInfo)

		sta, err := t.Engine.IsTableExist(devinfo)
		if err != nil {
			fmt.Println("IsTableExist ", err)
		}

		if sta == false {
			err = t.Engine.Sync2(devinfo)
			if err != nil {
				fmt.Println("sync dev ", err)
			}
		}

		devinfo.PhoneNum = utils.HexBuffToString(t.phoneNum)
		fmt.Println("phnoe:", devinfo.PhoneNum)

		tempinfo := &DevInfo{PhoneNum: devinfo.PhoneNum}
		is, _ := t.Engine.Get(tempinfo)
		if !is {
			fmt.Println("no this phone")
			return []byte{}
		}

		//_, err = t.Engine.Insert(devinfo)
		//if err != nil {
		//	fmt.Println("insert dev ", err)
		//}

		apduack := t.makeApduRegisterAck(0, tempinfo.Authkey)
		sendBuf := t.MakeFrame(registerAck, 1, t.phoneNum, t.seqNum, apduack)
		return sendBuf
	case login:
		fmt.Println("rcv login.")
		authkeylen := apdu[0]
		t.authkey = string(apdu[1 : 1+authkeylen])
		t.imei = string(apdu[1+authkeylen : 1+authkeylen+15])

		verArray := apdu[1+authkeylen+15 : 1+authkeylen+15+20]

		var emptyLen int = 0
		for i := 0; i < len(verArray); i++ {
			if verArray[len(verArray)-1-i] != 0x00 {
				break
			}
			emptyLen++
		}
		fmt.Println("emptylen:", emptyLen)
		t.tboxver = string(verArray[:len(verArray)-emptyLen])
		apduack := t.makeApduCommonAck(cmdType, 0)
		sendBuf := t.MakeFrame(platAck, 1, t.phoneNum, t.seqNum, apduack)

		return sendBuf
		//return []byte{}
	case heartbeat:
		fmt.Println("rcv heartbeat.")
		apduack := t.makeApduCommonAck(cmdType, 0)
		sendBuf := t.MakeFrame(platAck, 1, t.phoneNum, t.seqNum, apduack)

		return sendBuf
	case gpsinfo:
		fmt.Println("rcv gpsinfo.")
		apduack := t.makeApduCommonAck(cmdType, 0)
		sendBuf := t.MakeFrame(platAck, 1, t.phoneNum, t.seqNum, apduack)

		return sendBuf
	}

	return nil
}

func (t Terminal) paramHandle(id byte, data []byte) int {
	return 0
}

func (t Terminal) checkSum(data []byte) byte {
	var sum byte = 0
	for _, itemdata := range data {
		sum ^= itemdata
	}
	return sum
}

func (t Terminal) GetImei() string {
	return t.imei
}

func (t Terminal) GetIccid() string {
	return t.iccid
}

func (t Terminal) GetPhone() []byte {
	//fmt.Println("ret phone:", t.phoneNum)
	return []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x72, 0x55, 0x11, 0x11, 0x11}
}
