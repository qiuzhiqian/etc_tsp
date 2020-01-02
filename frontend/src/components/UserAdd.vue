<template>
  <div class="login_form">
    <Form class="inner_form" ref="formInline" :model="formInline" :rules="ruleInline">
      <FormItem prop="user">
        <Input
          prefix="ios-person-outline"
          type="text"
          v-model="formInline.user"
          placeholder="Username"
        />
      </FormItem>
      <FormItem prop="password">
        <Input
          prefix="ios-lock-outline"
          type="password"
          v-model="formInline.password"
          placeholder="Password"
        />
      </FormItem>
      <FormItem prop="password">
        <Input
          prefix="ios-lock-outline"
          type="password"
          v-model="formInline.repassword"
          placeholder="Password"
        />
      </FormItem>
      <FormItem label="Admin">
        <i-switch size="large" v-model="formInline.isAdmin">
          <Icon type="md-checkmark" slot="open"></Icon>
          <Icon type="md-close" slot="close"></Icon>
        </i-switch>
      </FormItem>
      <FormItem>
        <Button type="primary" @click="handleSubmit('formInline')">Submit</Button>
      </FormItem>
    </Form>
  </div>
</template>
<script>
export default {
  data() {
    return {
      formInline: {
        user: "",
        password: "",
        repassword: "",
        isAdmin: false
      },
      ruleInline: {
        user: [
          {
            required: true,
            message: "Please fill in the user name",
            trigger: "blur"
          }
        ],
        password: [
          {
            required: true,
            message: "Please fill in the password.",
            trigger: "blur"
          },
          {
            type: "string",
            min: 6,
            message: "The password length cannot be less than 6 bits",
            trigger: "blur"
          }
        ],
        repassword: [
          {
            required: true,
            message: "Please fill in the password.",
            trigger: "blur"
          },
          {
            type: "string",
            min: 6,
            message: "The password length cannot be less than 6 bits",
            trigger: "blur"
          }
        ]
      }
    };
  },
  methods: {
    handleSubmit(name) {
      this.$refs[name].validate(valid => {
        if (valid) {
          if (this.formInline.password != this.formInline.repassword) {
            this.$Message.error("password is not match!");
            return;
          }
          //window.console.log(name);
          this.axios
            .post("/api/v1/useradd", {
              user: this.formInline.user,
              password: this.formInline.password,
              admin: this.formInline.isAdmin
            })
            .then(function() {
              this.$Message.success("set success!");
            })
            .catch(error => {
              this.$Message.error("set error!");
              window.console.log(error);
            });
        } else {
          this.$Message.error("Fail!");
        }
      });
    }
  }
};
</script>

<style scoped>
.login_form {
  display: flex;
  justify-content: center; /* 水平居中 */
  align-items: center; /* 垂直居中 */
  width: 100%;
  height: 100%;
  position: absolute;
}
.inner_form {
  width: 300px;
  height: 250px;
}
</style>