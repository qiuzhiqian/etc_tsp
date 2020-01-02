<template>
  <div class="layout">
    <Layout style="height:100%;">
      <Sider ref="side1" hide-trigger collapsible :collapsed-width="78" v-model="isCollapsed">
        <Menu :active-name="activeName" theme="dark" width="auto" :class="menuitemClasses">
          <MenuItem name="1-1" to="/mainpage/devices">
            <Icon type="ios-cube" />
            <span>Devices</span>
          </MenuItem>
          <MenuItem name="1-2" to="/mainpage/gpstable">
            <Icon type="ios-pulse" />
            <span>Monitor</span>
          </MenuItem>
          <MenuItem name="1-3" to="/mainpage/mapwidget">
            <Icon type="ios-navigate" />
            <span>Map</span>
          </MenuItem>
          <MenuItem name="1-4" to="/mainpage/userManager">
            <Icon type="ios-navigate" />
            <span>userManager</span>
          </MenuItem>
        </Menu>
      </Sider>
      <Layout>
        <Header :style="{padding: 0}" class="layout-header-bar">
          <Icon
            @click.native="collapsedSider"
            :class="rotateIcon"
            :style="{margin: '10px 10px'}"
            type="md-menu"
            size="24"
          ></Icon>
        </Header>
        <Content :style="{margin: '10px', background: '#fff', height: '100%'}">
          <div class="main_container">
            <Breadcrumb>
              <BreadcrumbItem to="/">
                <Icon type="ios-home-outline"></Icon>Home
              </BreadcrumbItem>
              <BreadcrumbItem to="/components/breadcrumb">
                <Icon type="logo-buffer"></Icon>Components
              </BreadcrumbItem>
              <BreadcrumbItem>
                <Icon type="ios-cafe"></Icon>Breadcrumb
              </BreadcrumbItem>
            </Breadcrumb>
            <router-view class="main_inner"></router-view>
          </div>
        </Content>
        <Footer class="layout-footer-center">2019 &copy; xiamengliang</Footer>
      </Layout>
    </Layout>
  </div>
</template>
<script>
import Vue from "vue";
import BaiduMap from "vue-baidu-map";

export default {
  data() {
    return {
      isCollapsed: false,
      activeName: "1-1"
    };
  },
  computed: {
    rotateIcon() {
      return ["menu-icon", this.isCollapsed ? "rotate-icon" : ""];
    },
    menuitemClasses() {
      return ["menu-item", this.isCollapsed ? "collapsed-menu" : ""];
    }
  },
  methods: {
    collapsedSider() {
      this.$refs.side1.toggleCollapse();
    }
  },
  mounted: function() {
    if (this.$store.state.mapKey == "") {
      this.axios.post("/api/v1/config").then(resp => {
        window.console.log(resp.data);
        this.$store.commit("setMapkey", resp.data.mapAppKey);

        Vue.use(BaiduMap, {
          // ak 是在百度地图开发者平台申请的密钥 详见 http://lbsyun.baidu.com/apiconsole/key */
          ak: this.$store.state.mapKey
        });
      });
    }
  }
};
</script>
<style scoped>
.layout {
  border: 1px solid #d7dde4;
  background: #f5f7f9;
  border-radius: 4px;
  overflow: hidden;
  width: 100%;
  height: 100%;
  position: absolute;
}
.layout-header-bar {
  background: #fff;
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.1);
}
.layout-logo-left {
  width: 90%;
  height: 30px;
  background: #5b6270;
  border-radius: 3px;
  margin: 15px auto;
}
.menu-icon {
  transition: all 0.3s;
}
.rotate-icon {
  transform: rotate(-90deg);
}
.menu-item span {
  display: inline-block;
  overflow: hidden;
  width: 100px;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
  transition: width 0.2s ease 0.2s;
}
.menu-item i {
  transform: translateX(0px);
  transition: font-size 0.2s ease, transform 0.2s ease;
  vertical-align: middle;
  font-size: 16px;
}
.collapsed-menu span {
  width: 0px;
  transition: width 0.2s ease;
}
.collapsed-menu i {
  transform: translateX(5px);
  transition: font-size 0.2s ease 0.2s, transform 0.2s ease 0.2s;
  vertical-align: middle;
  font-size: 22px;
}
.layout-footer-center {
  height: 20px;
  padding: 0px 20px;
  text-align: center;
}

.main_container {
  height: 100%;
  display: -webkit-flex; /* Safari */
  display: flex;
  flex-direction: column;
}

.main_inner {
  flex-grow: 1;
}
</style>