<template>
  <div>
    <DatePicker
      type="datetimerange"
      v-model="datepick"
      placeholder="Select date and time"
      style="width: 300px"
    ></DatePicker>
    <Select v-model="model1" style="width:100px">
      <Option v-for="item in searchList" :value="item.value" :key="item.value">{{ item.label }}</Option>
    </Select>
    <Input v-model="value" placeholder="Enter something..." style="width: 300px" />
    <Button @click="getdata(1)">Search</Button>

    <Table stripe :columns="columns1" :data="data1"></Table>
    <Page
      :current="current"
      :total="total"
      :page-size="pagesize"
      show-elevator
      @on-change="pagechange"
    />
  </div>
</template>
<script>
export default {
  data() {
    return {
      searchList: [
        {
          value: "IMEI",
          label: "IMEI"
        },
        {
          value: "PHONE",
          label: "PHONE"
        },
        {
          value: "ICCID",
          label: "ICCID"
        }
      ],
      model1: "IMEI",
      value: "",
      columns1: [
        {
          title: "IMEI",
          key: "imei"
        },
        {
          title: "Time",
          key: "time"
        },
        {
          title: "WarnFlag",
          key: "warnFlag"
        },
        {
          title: "State",
          key: "state"
        },
        {
          title: "Latitude",
          key: "latitude"
        },
        {
          title: "Longitude",
          key: "longitude"
        },
        {
          title: "Altitude",
          key: "altitude"
        },
        {
          title: "Speed",
          key: "speed"
        },
        {
          title: "Direction",
          key: "direction"
        },
        {
          title: "DataTime",
          key: "dataTime"
        }
      ],
      data1: [],
      total: 20,
      pagesize: 10,
      current: 1,
      datepick: []
    };
  },
  methods: {
    timeChange: function(t1, t2) {
      window.console.log(t1);
      window.console.log(t2);
    },
    getdata: function(index) {
      if (this.value === "") {
        this.$Message.warning("Please imei!");
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
        .post("/api/v1/data", {
          imei: this.value,
          starttime: this.datepick[0].getTime() / 1000,
          endtime: this.datepick[1].getTime() / 1000,
          page: index
        })
        .then(response => {
          window.console.log(response.data);
          window.console.log(new Date(response.data.data[0].stamp));
          this.data1.splice(0, this.data1.length);
          for (var i = 0, len = response.data.data.length; i < len; i++) {
            //console.log(arr[j]);
            var tempdate = new Date(response.data.data[i].stamp * 1000);
            var datestr =
              tempdate.toLocaleDateString().replace(/\//g, "-") +
              " " +
              tempdate.toTimeString().substr(0, 8);

            var datadate = new Date(response.data.data[i].dataStamp * 1000);
            var datastr =
              datadate.toLocaleDateString().replace(/\//g, "-") +
              " " +
              datadate.toTimeString().substr(0, 8);

            var item = {
              imei: response.data.data[i].imei,
              time: datestr,
              warnFlag: response.data.data[i].warnflag,
              state: response.data.data[i].state,
              latitude: response.data.data[i].latitude,
              longitude: response.data.data[i].longitude,
              altitude: response.data.data[i].altitude,
              speed: response.data.data[i].speed,
              direction: response.data.data[i].direction,
              dataTime: datastr,
            };
            this.data1.push(item);

            this.total = response.data.pagecnt * response.data.pagesize;
            this.pagesize = response.data.pagesize;
            this.current = response.data.pageindex;
          }
          window.console.log(this.data1);
        })
        .catch(error => {
          window.console.log(error);
        });
    },
    pagechange: function(index) {
      window.console.log(index);
      this.getdata(index);
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
</style>
