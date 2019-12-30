package codec

import (
	"testing"
)

func TestUnmarshal(t *testing.T) {
	type Data struct {
		Size    int8
		Size2   uint16
		Size3   uint32
		Name    string `len:"5"`
		Message string
		Sec     []byte `len:"3"`
	}

	type Body struct {
		Age1   int8
		Age2   int16
		Length int32
		Data1  Data
	}

	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0x31, 0x32, 0x33, 0x34, 0x35, 0x31, 0x30, 0x03, 0x02, 0x01}
	pack := Body{}
	i, err := Unmarshal(data, &pack)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	t.Log("len:", i)
	t.Log("pack:", pack)
}

func TestMarshal(t *testing.T) {
	type Data struct {
		Size    int8
		Size2   uint16
		Size3   uint32
		Name    string `len:"5"`
		Message string
		Sec     []byte `len:"3"`
	}

	type Body struct {
		Age1   int8
		Age2   int16
		Length int32
		Data1  Data
	}

	pack := Body{
		Age1:   13,
		Age2:   1201,
		Length: 81321,
		Data1: Data{
			Size:    110,
			Size2:   39210,
			Size3:   85632,
			Name:    "ASDFG",
			Message: "ZXCVBN",
			Sec:     []byte{0x01, 0x02, 0x03},
		},
	}
	data, err := Marshal(&pack)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	t.Log("data:", data)
	t.Log("pack:", pack)
}
