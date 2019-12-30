package proto

import (
	"testing"
)

func TestFilter(t *testing.T) {
	data := []byte{protoHeader, 0x00, 0x01, 0x00, 0x03, 0x01, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x01, 0x12, 0x34, 0x56}
	cs := checkSum(data[1:])
	data = append(data, cs, protoHeader)
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
