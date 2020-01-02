<template>
  <div class="map_container">
    <div>
      <DatePicker
        type="datetimerange"
        v-model="datepick"
        placeholder="Select date and time"
        style="width: 300px"
      ></DatePicker>
      <Input v-model="value" placeholder="Enter IMEI..." style="width: 300px" @on-enter="doEnter" />
    </div>
    <baidu-map
      class="map_center"
      :center="center"
      :zoom="zoom"
      scroll-wheel-zoom="true"
      @ready="readyhandler"
    >
      <bm-marker :position="markPoint"></bm-marker>
      <bm-polyline
        :path="polylinePath"
        stroke-color="blue"
        :stroke-opacity="0.5"
        :stroke-weight="4"
        :editing="false"
        @lineupdate="updatePolylinePath"
      ></bm-polyline>
    </baidu-map>
  </div>
</template>

<script>
export default {
  data() {
    return {
      center: { lng: 0, lat: 0 },
      markPoint: { lng: 0, lat: 0 },
      zoom: 3,
      value: "",
      polylinePath: [],
      datepick: []
    };
  },
  methods: {
    readyhandler() {
      this.center.lng = 113.912593;
      this.center.lat = 22.585452;
      this.markPoint.lng = 113.912593;
      this.markPoint.lat = 22.585452;
      this.zoom = 15;
    },
    getNowGps: function(imeistr) {
      if (imeistr === "") {
        return;
      }

      if (this.datepick.length != 2) {
        this.$Message.warning("Please select start and end time!");
        return;
      }

      if (this.datepick[0] == "" || this.datepick[1] == "") {
        this.$Message.warning("Please select start and end time!");
        return;
      }

      this.axios
        .post("/api/v1/nowgps", {
          imei: imeistr
        })
        .then(response => {
          if (response.data.latitude == 0 || response.data.longitude == 0) {
            return;
          }

          this.center.lng = response.data.longitude / 1000000;
          this.center.lat = response.data.latitude / 1000000;
          this.markPoint.lng = response.data.longitude / 1000000;
          this.markPoint.lat = response.data.latitude / 1000000;
          window.console.log(
            "lng:",
            this.markPoint.lng,
            "lat:",
            this.markPoint.lat
          );

          var bdloc = this.utils.wgs2bd(this.markPoint.lat, this.markPoint.lng);
          this.markPoint.lat = bdloc[0];
          this.markPoint.lng = bdloc[1];
        })
        .catch(error => {
          window.console.log(error);
        });

      this.axios
        .post("/api/v1/gpsmap", {
          imei: imeistr,
          starttime: this.datepick[0].getTime() / 1000,
          endtime: this.datepick[1].getTime() / 1000
        })
        .then(response => {
          window.console.log("len:", response.data.length);
          this.polylinePath.splice(0, this.polylinePath.length);
          for (var i = 0, len = response.data.length; i < len; i++) {
            if (
              response.data[i].latitude == 0 ||
              response.data[i].longitude == 0
            ) {
              continue;
            }

            if (
              i > 0 &&
              Math.abs(
                response.data[i].latitude - response.data[i - 1].latitude
              ) < 500 &&
              Math.abs(
                response.data[i].longitude - response.data[i - 1].longitude
              ) < 500
            ) {
              continue;
            }
            var bdloc = this.utils.wgs2bd(
              response.data[i].latitude / 1000000,
              response.data[i].longitude / 1000000
            );
            this.polylinePath.push({
              lng: bdloc[1],
              lat: bdloc[0]
            });
          }
          window.console.log("len2:", this.polylinePath.length);
        })
        .catch(error => {
          window.console.log(error);
        });
    },
    doEnter() {
      this.getNowGps(this.value);
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
.map_container {
  display: -webkit-flex; /* Safari */
  display: flex;
  flex-direction: column;
}

.map_center {
  flex-grow: 1;
}
</style>
