<script setup lang="ts">
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
                    <input type="text" placeholder="管理员账号" class="layui-input">
                </div>
            </div>
            <div class="password">
                <div class="layui-input-wrap">
                    <div class="layui-input-prefix">
                        <i class="layui-icon layui-icon-password"></i>
                    </div>
                    <input type="text" placeholder="管理员密码" class="layui-input">
                </div>
            </div>
        </div>
        <button type="button" class="layui-btn layui-bg-blue login">登录后台</button>
        <a class="foot" href="https://github.com/MutantCat-Working-Group/CloudStep" target="_self">https://github.com/MutantCat-Working-Group/CloudStep</a>
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

.userinfo{
    width: 500px;
    margin: 0 auto;
}

.username {
    margin-top: 20px;
}

.password{
    margin-top: 10px;
}

.login{
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

.foot{
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