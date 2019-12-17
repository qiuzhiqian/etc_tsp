package utils

func Bytes2Word(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return (uint16(data[0]) << 8) + uint16(data[1])
}

func Word2Bytes(data uint16) []byte {
	buff := make([]byte, 2)
	buff[0] = byte(data >> 8)
	buff[1] = byte(data)
	return buff
}

func Bytes2DWord(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return (uint32(data[0]) << 24) + (uint32(data[1]) << 16) + (uint32(data[2]) << 8) + uint32(data[3])
}

func Dword2Bytes(data uint32) []byte {
	buff := make([]byte, 4)
	buff[0] = byte(data >> 24)
	buff[1] = byte(data >> 16)
	buff[2] = byte(data >> 8)
	buff[3] = byte(data)
	return buff
}

func Str2bytes(s string) []byte {
	p := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		p[i] = c
	}
	return p
}

func HexToAsc(hex byte) byte {
	var asc byte = 0
	if hex >= 0 && hex <= 0x09 {
		asc = hex - 0 + '0'
	} else if hex >= 0x0a && hex <= 0x0f {
		asc = hex - 0x0a + 'A'
	}
	return asc
}

func AscToHex(asc byte) byte {
	var hex byte = '0'
	if asc >= '0' && asc <= '9' {
		hex = asc - '0' + 0
	} else if asc >= 'a' && asc <= 'f' {
		hex = asc - 'a' + 0x0a
	} else if asc >= 'A' && asc <= 'F' {
		hex = asc - 'A' + 0x0A
	}
	return hex
}

func HexBuffToString(hex []byte) string {
	var ret []byte

	for _, item := range hex {
		hasc := HexToAsc((item >> 4) & 0x0F)
		lasc := HexToAsc((item) & 0x0F)

		if hasc == 0 || lasc == 0 {
			break
		}

		ret = append(ret, hasc, lasc)
	}

	var index int = -1
	for i, val := range ret {
		if val != '0' {
			index = i
			break
		}
	}

	if index < 0 {
		return string("")
	}

	return string(ret[index:])
}

/*std::vector<uint8_t> CUtils::StringToHexBuff(const std::string& str){
    std::vector<uint8_t> ret;
    if( (str.size()%2) !=0 ){
        return ret;
    }

    int len = str.size()/2;
    for(int i=0;i<len;i++){
        char hhex = AscToHex(str.at(i*2));
        char lhex = AscToHex(str.at(i*2+1));

        if(hhex == '0' || lhex == '0'){
            printf("StringToHexBuff error.\n");
            return ret;
        }

        ret.push_back((hhex<<4)+lhex);
    }
    return ret;
}*/
