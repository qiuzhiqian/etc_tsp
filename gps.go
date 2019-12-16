package main

import "math"

const EARTH_RADIUS float64 = 6370.856 //km 地球半径 平均值，千米
//地球半径
const EARTH_R float64 = 6378245.0

func HaverSin(theta float64) float64 {
	v := math.Sin(theta / 2)
	return v * v
}

func Distance(lat1, lon1, lat2, lon2 float64) float64 {
	//用haversine公式计算球面两点间的距离。
	//经纬度转换成弧度
	lat1 = ConvertDegreesToRadians(lat1)
	lon1 = ConvertDegreesToRadians(lon1)
	lat2 = ConvertDegreesToRadians(lat2)
	lon2 = ConvertDegreesToRadians(lon2)

	//差值
	var vLon float64 = math.Abs(lon1 - lon2)
	var vLat float64 = math.Abs(lat1 - lat2)

	//h is the great circle distance in radians, great circle就是一个球体上的切面，它的圆心即是球心的一个周长最大的圆。
	var h float64 = HaverSin(vLat) + math.Cos(lat1)*math.Cos(lat2)*HaverSin(vLon)

	var distance float64 = 2 * EARTH_RADIUS * math.Asin(math.Sqrt(h))

	return distance
}

/**
* 计算两点之间的角度
*
* @param pntFirst
* @param pntNext
* @return
 */
func GetAngle(lat1, lon1, lat2, lon2 float64) float64 {
	var dRotateAngle float64 = math.Atan2(math.Abs(lon1-lon2), math.Abs(lat1-lat2))
	if lon2 >= lon1 {
		if lat2 >= lat1 {
		} else {
			dRotateAngle = math.Pi - dRotateAngle
		}
	} else {
		if lat2 >= lat1 {
			dRotateAngle = 2*math.Pi - dRotateAngle
		} else {
			dRotateAngle = math.Pi + dRotateAngle
		}
	}
	dRotateAngle = dRotateAngle * 180 / math.Pi
	return dRotateAngle
}

func ConvertDegreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func ConvertRadiansToDegrees(radian float64) float64 {
	return radian * 180.0 / math.Pi
}

const ee float64 = 0.00669342162296594323

const XPI float64 = math.Pi * 3000.0 / 180.0

//世界大地坐标转为百度坐标
func wgs2bd(lat, lon float64) (bdlat, bdlng float64) {
	gcjlat, gcjlng := wgs2gcj(lat, lon)
	bdlat, bdlng = gcj2bd(gcjlat, gcjlng)
	return
}

func gcj2bd(lat, lon float64) (bdlat, bdlng float64) {
	var x float64 = lon
	var y float64 = lat
	var z float64 = math.Sqrt(x*x+y*y) + 0.00002*math.Sin(y*XPI)
	var theta float64 = math.Atan2(y, x) + 0.000003*math.Cos(x*XPI)
	bdlng = z*math.Cos(theta) + 0.0065
	bdlat = z*math.Sin(theta) + 0.006
	return
}

func bd2gcj(lat, lon float64) (gcjlat, gcjlng float64) {
	var x float64 = lon - 0.0065
	var y float64 = lat - 0.006
	var z float64 = math.Sqrt(x*x+y*y) - 0.00002*math.Sin(y*XPI)
	var theta float64 = math.Atan2(y, x) - 0.000003*math.Cos(x*XPI)
	gcjlng = z * math.Cos(theta)
	gcjlat = z * math.Sin(theta)
	return
}

func wgs2gcj(lat, lon float64) (gcjlat, gcjlng float64) {
	var dLat float64 = transformLat(lon-105.0, lat-35.0)
	var dLon float64 = transformLon(lon-105.0, lat-35.0)
	var radLat float64 = lat / 180.0 * math.Pi
	var magic float64 = math.Sin(radLat)
	magic = 1 - ee*magic*magic
	var sqrtMagic float64 = math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((EARTH_R * (1 - ee)) / (magic * sqrtMagic) * math.Pi)
	dLon = (dLon * 180.0) / (EARTH_R / sqrtMagic * math.Cos(radLat) * math.Pi)
	gcjlat = lat + dLat
	gcjlng = lon + dLon
	return
}

func transformLat(lat, lon float64) float64 {
	var ret = -100.0 + 2.0*lat + 3.0*lon + 0.2*lon*lon + 0.1*lat*lon + 0.2*math.Sqrt(math.Abs(lat))
	ret += (20.0*math.Sin(6.0*lat*math.Pi) + 20.0*math.Sin(2.0*lat*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(lon*math.Pi) + 40.0*math.Sin(lon/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(lon/12.0*math.Pi) + 320*math.Sin(lon*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformLon(lat, lon float64) float64 {
	var ret = 300.0 + lat + 2.0*lon + 0.1*lat*lat + 0.1*lat*lon + 0.1*math.Sqrt(math.Abs(lat))
	ret += (20.0*math.Sin(6.0*lat*math.Pi) + 20.0*math.Sin(2.0*lat*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(lat*math.Pi) + 40.0*math.Sin(lat/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(lat/12.0*math.Pi) + 300.0*math.Sin(lat/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}
