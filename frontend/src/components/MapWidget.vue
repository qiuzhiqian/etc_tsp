<template>
  <div class="map_container">
    <baidu-map class="map" :center="{lng: 116.404, lat: 39.915}" :zoom="15"></baidu-map>
  </div>
</template>

<script>
export default {
  data() {
    return {
      center: { lng: 0, lat: 0 },
      zoom: 3
    };
  },
  methods: {
    handler({ BMap, map }) {
      window.console.log(BMap, map);
      this.center.lng = 116.404;
      this.center.lat = 39.915;
      this.zoom = 15;
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
.map_container {
  height: 100%;
}

.map {
  height: 100%;
}
</style>
