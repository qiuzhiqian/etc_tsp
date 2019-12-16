<template>
  <div>
    <Button type="primary" to="/mainpage/userAdd">
      <Icon type="md-add" />
    </Button>
    <Table stripe :columns="columns1" :data="data1">
      <template slot-scope="{ row }" slot="action">
        <Button
          type="primary"
          size="small"
          style="margin-right: 5px"
          @click="userDelete(row.user)"
        >Map</Button>
      </template>
    </Table>
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
      columns1: [
        {
          title: "User",
          key: "user"
        },
        {
          title: "Role",
          key: "role"
        },
        {
          title: "CreateTime",
          key: "createTime"
        },
        {
          title: "Action",
          slot: "action",
          width: 300,
          align: "center"
        }
      ],
      data1: [],
      total: 10,
      pagesize: 10,
      current: 1
    };
  },
  methods: {
    getdata: function(index) {
      this.axios
        .post("/api/userlist", {
          page: index
        })
        .then(response => {
          this.data1.splice(0, this.data1.length);
          for (var i = 0, len = response.data.data.length; i < len; i++) {
            var item = {
              ip: response.data.data[i].ip,
              imei: response.data.data[i].imei,
              phoneNum: response.data.data[i].phone
            };
            this.data1.push(item);
          }

          this.total = response.data.pagecnt * response.data.pagesize;
          this.pagesize = response.data.pagesize;
          this.current = response.data.pageindex;
        })
        .catch(error => {
          window.console.log(error);
        });
    },
    pagechange: function(index) {
      window.console.log(index);
      this.getdata(index);
    },
    userDelete: function(param) {
      window.console.log("userDelete:", param);
    }
  },
  mounted: function() {
    this.getdata(1);
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
</style>
