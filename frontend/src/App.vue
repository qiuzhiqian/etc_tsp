<template>
  <MainPage v-if="pageSelect == 'main'"></MainPage>
  <Login v-else></Login>
</template>
<script>
import MainPage from "./components/MainPage";
import Login from "./components/Login";
export default {
  data() {
    return {
      pageSelect: "main"
    };
  },
  components: {
    MainPage,
    Login
  }
};
</script>
<style scoped>
</style>