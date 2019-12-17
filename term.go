package main

import (
	"bytes"
	"encoding/gob"
	"net"
	"time"

	"tsp/utils"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
)

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

	termAck     uint16 = 0x0001
	register    uint16 = 0x0100
	registerAck uint16 = 0x8100
	unregister  uint16 = 0x0003
	login       uint16 = 0x0102
	heartbeat   uint16 = 0x0002
	gpsinfo     uint16 = 0x0200
	platAck     uint16 = 0x8001
	UpdateReq   uint16 = 0x8108
	CtrlReq     uint16 = 0x8105
)

func deepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func (t *Terminal) MakeFrame(cmd uint16, ver uint8, phone []byte, seq uint16, apdu []byte) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(cmd)
	data = append(data, tempbytes...)
	datalen := uint16(len(apdu)) & 0x03FF
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

	//添加头尾
	var tmpdata []byte = []byte{0x7e}

	for _, item := range data {
		if item == 0x7d {
			log.Info("has 0x7d")
			tmpdata = append(tmpdata, 0x7d, 0x01)
		} else if item == 0x7e {
			log.Info("has 0x7e")
			tmpdata = append(tmpdata, 0x7d, 0x02)
		} else {
			tmpdata = append(tmpdata, item)
		}
	}
	tmpdata = append(tmpdata, 0x7e)

	return tmpdata
}

func (t *Terminal) MakeFrameMult(cmd uint16, ver uint8, phone []byte, seq, sum, cur uint16, apdu []byte) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(cmd)
	data = append(data, tempbytes...)
	datalen := uint16(len(apdu)) & 0x03FF
	log.Info("datalen:", datalen, " len:", uint16(len(apdu)))
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
	for _, item := range data {
		if item == 0x7d {
			log.Info("has 0x7d")
			tmpdata = append(tmpdata, 0x7d, 0x01)
		} else if item == 0x7e {
			log.Info("has 0x7e")
			tmpdata = append(tmpdata, 0x7d, 0x02)
		} else {
			tmpdata = append(tmpdata, item)
		}
	}
	tmpdata = append(tmpdata, 0x7e)

	return tmpdata
}

func (t *Terminal) makeApduRegisterAck(res uint8, authkey string) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(t.seqNum)
	data = append(data, tempbytes...)

	data = append(data, res)

	for _, item := range authkey {
		data = append(data, byte(item))
	}

	return data
}

func (t *Terminal) makeApduCommonAck(cmdid uint16, res byte) []byte {
	data := make([]byte, 0)
	tempbytes := utils.Word2Bytes(t.seqNum)
	data = append(data, tempbytes...)

	tempbytes = utils.Word2Bytes(cmdid)
	data = append(data, tempbytes...)

	data = append(data, res)

	log.Info("apdu:", data)
	return data
}

func (t *Terminal) makeApduCtrl(cmdid byte, param string) []byte {
	data := make([]byte, 0)

	data = append(data, cmdid)
	data = append(data, param...)

	log.Info("apdu:", data)
	return data
}

func (t *Terminal) DataFilter(data []byte) int {
	//--------------------------------------------------
	if data[0] == protoHeader {
		log.Info("find start.")
		var endindex int = -1
		for i := 1; i < len(data); i++ {
			if data[i] == protoHeader {
				log.Info("find end.")
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

func (t *Terminal) FrameHandle(data []byte) []byte {
	cmdid := utils.Bytes2Word(data[1:3])
	if t.phoneNum == nil {
		t.phoneNum = make([]byte, 10)
	}

	//deepCopy(t.phoneNum, data[6:6+10])
	for index, item := range data[6 : 6+10] {
		t.phoneNum[index] = item
	}
	t.seqNum = utils.Bytes2Word(data[16:18])
	log.Info("cmdid:", cmdid)
	len := len(data)
	return t.apduHandle(cmdid, data[18:len-2])
}

func (t *Terminal) apduHandle(cmdType uint16, apdu []byte) []byte {
	switch cmdType {
	case termAck:
		log.Info("rcv termAck.")
		reqId := utils.Bytes2Word(apdu[2:4])
		log.Info("reqId:", reqId)
		if reqId == UpdateReq {
			ch <- 1
		}
	case register:
		log.Info("rcv register.")

		devinfo := new(DevInfo)

		devinfo.PhoneNum = utils.HexBuffToString(t.phoneNum)
		log.Info("phnoe:", devinfo.PhoneNum)

		//tempinfo := &DevInfo{PhoneNum: devinfo.PhoneNum}
		is, _ := t.Engine.Get(devinfo)
		if !is {
			log.Info("no this phone")
			return []byte{}
		}

		apduack := t.makeApduRegisterAck(0, devinfo.Authkey)
		sendBuf := t.MakeFrame(registerAck, 1, t.phoneNum, t.seqNum, apduack)
		return sendBuf
	case login:
		log.Info("rcv login.")
		authkeylen := apdu[0]
		tempSlice := make([]byte, authkeylen)
		copy(tempSlice, apdu[1:1+authkeylen])
		t.authkey = string(tempSlice)

		tempSlice = make([]byte, 15)
		copy(tempSlice, apdu[1+authkeylen:1+authkeylen+15])
		t.imei = string(tempSlice)
		log.Info("imei:", t.imei)

		verArray := apdu[1+authkeylen+15 : 1+authkeylen+15+20]

		var emptyLen int = 0
		for i := 0; i < len(verArray); i++ {
			if verArray[len(verArray)-1-i] != 0x00 {
				break
			}
			emptyLen++
		}
		log.Info("emptylen:", emptyLen)
		t.tboxver = string(verArray[:len(verArray)-emptyLen])
		apduack := t.makeApduCommonAck(cmdType, 0)
		sendBuf := t.MakeFrame(platAck, 1, t.phoneNum, t.seqNum, apduack)

		return sendBuf
		//return []byte{}
	case heartbeat:
		log.Info("rcv heartbeat.")
		apduack := t.makeApduCommonAck(cmdType, 0)
		sendBuf := t.MakeFrame(platAck, 1, t.phoneNum, t.seqNum, apduack)

		return sendBuf
	case gpsinfo:
		log.Info("rcv gpsinfo.")

		var index int = 0
		gpsdata := new(GPSData)
		gpsdata.Imei = t.imei
		gpsdata.Stamp = time.Now()
		gpsdata.WarnFlag = utils.Bytes2DWord(apdu[index : index+4])
		index += 4
		gpsdata.State = utils.Bytes2DWord(apdu[index : index+4])
		index += 4
		gpsdata.Latitude = utils.Bytes2DWord(apdu[index : index+4])
		index += 4
		gpsdata.Longitude = utils.Bytes2DWord(apdu[index : index+4])
		index += 4

		gpsdata.Altitude = utils.Bytes2Word(apdu[index : index+2])
		index += 2
		gpsdata.Speed = utils.Bytes2Word(apdu[index : index+2])
		index += 2
		gpsdata.Direction = utils.Bytes2Word(apdu[index : index+2])
		index += 2

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

		_, err := engine.Insert(gpsdata)
		if err != nil {
			log.Info("insert gps err:", err)
		}

		apduack := t.makeApduCommonAck(cmdType, 0)
		sendBuf := t.MakeFrame(platAck, 1, t.phoneNum, t.seqNum, apduack)

		return sendBuf
	}

	return nil
}

func (t *Terminal) paramHandle(id byte, data []byte) int {
	return 0
}

func (t *Terminal) checkSum(data []byte) byte {
	var sum byte = 0
	for _, itemdata := range data {
		sum ^= itemdata
	}
	return sum
}

func (t *Terminal) GetImei() string {
	log.Info("get imei:", t.imei)
	return t.imei
}

func (t *Terminal) GetIccid() string {
	return t.iccid
}

func (t *Terminal) GetPhone() []byte {
	//log.Info("ret phone:", t.phoneNum)
	//return []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x72, 0x55, 0x11, 0x11, 0x11}
	return t.phoneNum
}
