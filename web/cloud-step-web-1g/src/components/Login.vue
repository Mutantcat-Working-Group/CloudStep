<script setup lang="ts">
import request from '../utils/request'
import { ref,defineEmits  } from 'vue'

const emit = defineEmits(['check'])
const username = ref("")
const password = ref("")

function login() {
    request.request<any>(
        {
            url: '../login',
            method: 'post',
            data: {
                username: username.value,
                password: password.value
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            window.sessionStorage.setItem('token', res.data.token)
            emit('check')
        } else if (res.data.code == 2) {
            alert('尝试失败次数太多，请三分钟之后再尝试登录。')
        }
        else {
            alert('用户名或密码错误。')
        }
    }).catch(() => {
        alert('登录错误。')
    });
}
</script>

<template>
    <div class="container">
        <img src="../assets/icons/CloudStep.jpg" class="icon">
        <h2 class="title">后台登陆</h2>
        <div class="userinfo">
            <div class="username">
                <div class="layui-input-wrap">
                    <div class="layui-input-prefix">
                        <i class="layui-icon layui-icon-username"></i>
                    </div>
                    <input type="text" placeholder="管理员账号" v-model="username" class="layui-input" style="caret-color: black;">
                </div>
            </div>
            <div class="password">
                <div class="layui-input-wrap">
                    <div class="layui-input-prefix">
                        <i class="layui-icon layui-icon-password"></i>
                    </div>
                    <input type="password" placeholder="管理员密码" v-model="password" class="layui-input" style="caret-color: black;">
                </div>
            </div>
        </div>
        <button type="button" class="layui-btn layui-bg-blue login" @click="login()">登 录</button>
        <a class="foot" href="https://www.mutantcat.org/"
            target="_self">https://www.mutantcat.org/</a>
    </div>
</template>

<style scoped>
.container {
    width: 100vw;
    background-color: #dfcbc0;
    height: 100vh;
    float: left;
}

.icon {
    width: 100px;
    height: 100px;
    display: block;
    margin: 0 auto;
    margin-top: 100px;
    border-radius: 15px;
}

.title {
    text-align: center;
    margin-top: 10px;
    font-size: 40px;
    color: #333;
    font-weight: 400;
}

.userinfo {
    width: 500px;
    margin: 0 auto;
}

.username {
    margin-top: 20px;
}

.password {
    margin-top: 10px;
}

.login {
    width: 200px;
    margin: 0 auto;
    margin-top: 20px;
    background-color: #1E9FFF;
    color: #fff;
    border: none;
    border-radius: 5px;
    font-size: 20px;
    font-weight: 400;
    height: 40px;
    line-height: 40px;
    text-align: center;
    cursor: pointer;
    display: block;
    transition: all 0.3s;
}

.foot {
    width: 100%;
    text-align: center;
    font-size: 20px;
    color: #727070;
    font-weight: 400;
    display: block;
    text-decoration: none;
    transition: all 0.3s;
    position: fixed;
    bottom: 20px;
}
</style>