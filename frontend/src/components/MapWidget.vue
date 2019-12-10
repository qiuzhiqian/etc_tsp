<template>
  <div class="map_container">
    <Input v-model="value" placeholder="Enter IMEI..." style="width: 300px" @on-enter="doEnter" />
    <baidu-map
      class="map_center"
      :center="center"
      :zoom="zoom"
      scroll-wheel-zoom="true"
      @ready="readyhandler"
    >
      <bm-marker :position="markPoint"></bm-marker>
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
      value: ""
    };
  },
  methods: {
    readyhandler({ BMap, map }) {
      window.console.log(BMap, map);
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

      this.axios
        .post("/api/nowgps", {
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
  height: 100%;
  display: -webkit-flex; /* Safari */
  display: flex;
  flex-direction: column;
}

.map_center {
  flex-grow: 1;
}
</style>
