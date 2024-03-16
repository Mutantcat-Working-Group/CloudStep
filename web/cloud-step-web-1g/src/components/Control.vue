<script setup lang="ts">
</script>

<template>
    <div class="contant">
        <div class="layui-tab layui-tab-card main-contant">
            <ul class="layui-tab-title">
                <li class="layui-this">映射管理</li>
                <li>自助模式</li>
                <li>代理模式</li>
                <li>系统管理</li>
            </ul>
            <div class="layui-tab-content">
                <div class="layui-tab-item layui-show">
                    <div class="collection">
                        <h2 class="control">添加映射集</h2>
                        <div class="layui-input-group center">
                            <div class="layui-input-split layui-input-prefix">
                                映射集名称
                            </div>
                            <input type="text" placeholder="映射集的唯一标识" class="layui-input">
                        </div>
                        <h2 class="info">映射集列表</h2>
                    </div>
                </div>
                <div class="layui-tab-item">
                    <div class="collection">
                        <h2 class="control">自助设置</h2>

                        <h2 class="info">映射管理</h2>
                    </div>
                </div>
                <div class="layui-tab-item">
                    <div class="collection">
                        <h2 class="control">代理管理</h2>

                        <h2 class="info">映射管理</h2>
                    </div>
                </div>
                <div class="layui-tab-item">
                    <div class="collection">
                        <h2 class="control">密码设置</h2>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.contant {
    width: 100vw;
    height: 100vh;
    float: left;
}

.main-contant {
    width: 96vw;
    height: 100%;
    margin: 0 auto;
}

.collection {
    width: 95%;
    height: 100%;
    margin: 0 auto;
    margin-top: 20px;
}

.collection .control {
    font-size: 20px;
    color: #333;
    font-weight: 400;
    margin-top: 10px;
    margin-bottom: 10px;
    border: #333 1px solid;
    border-radius: 15px;
    padding: 10px;
    text-align: center;
}

.collection .info {
    font-size: 20px;
    color: #333;
    font-weight: 400;
    margin-top: 10px;
    margin-bottom: 10px;
    border: #333 1px solid;
    border-radius: 15px;
    padding: 10px;
    text-align: center;
}

.center{
    margin:0 auto;
}
</style>