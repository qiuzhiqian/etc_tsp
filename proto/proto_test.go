package proto

import (
	"testing"
)

func TestFilter(t *testing.T) {
	data := []byte{ProtoHeader, 0x00, 0x01, 0x00, 0x03, 0x01, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x01, 0x12, 0x34, 0x56}
	cs := checkSum(data[1:])
	data = append(data, cs, ProtoHeader)
	t.Log("data:", data)

	testbuff := make([]byte, 10)
	copy(testbuff, data)
	msg, lens, err := Filter(testbuff)
	if err != nil {
		t.Log("err:", err)
	}

	t.Log("msg:", msg)
	t.Log("lens:", lens)

	testbuff = append(testbuff[lens:], data[10:]...)
	testbuff = append(testbuff, 0x01, 0x05)
	testbuff = append(testbuff, data[:3]...)
	msg, lens, err = Filter(testbuff)
	if err != nil {
		t.Log("err:", err)
	}

	t.Log("msg:", msg)
	t.Log("lens:", lens)

	testbuff = append(testbuff[lens:], data[3:]...)
	msg, lens, err = Filter(testbuff)
	if err != nil {
		t.Log("err:", err)
	}

	t.Log("msg:", msg)
	t.Log("lens:", lens)
}

func TestEscape(t *testing.T) {
	data := []byte{0x7e, 0x02, 0x00, 0x40, 0x7d, 0x01, 0x04, 0x7d, 0x02, 0x7e}
	t.Log("data:", data)
	data1 := Escape(data, []byte{0x7d, 0x02}, []byte{0x7d})
	t.Log("data1:", data1)
}
