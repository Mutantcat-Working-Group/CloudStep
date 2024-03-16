<script setup lang="ts">
import { ref } from 'vue'

const addCollections = ref([{
    address: "",
    status: "未知"
}])

function addCollection() {
    addCollections.value.push({
        address: "",
        status: "未知"
    })
}

function removeCollection() {
    addCollections.value.pop()
}

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
                        <div class="member">
                            <div style="width:100%">
                                <div class="layui-row">
                                    <div class="layui-col-xs3">
                                        <div class="text-center bigger-text">序号</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center bigger-text">地址</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center bigger-text">状态</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center bigger-text">操作</div>
                                    </div>
                                </div>
                            </div>
                            <div class="items" style="width:100%" v-for="(item, index) in addCollections" :key="index">
                                <div class="layui-row">
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item">{{ index + 1 }}</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item"><input type="text" lay-affix="clear"
                                                placeholder="http://abc.def:1234/xxx" class="layui-input"></div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item">{{ item.status }}</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item">
                                            <button type="button"
                                                class="layui-btn layui-btn-primary layui-btn-sm">本地Ping</button>
                                            <button type="button"
                                                class="layui-btn layui-btn-primary layui-btn-sm">服务端Ping</button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div class="center">
                                <button type="button" class="layui-btn" @click="removeCollection()" style="float:left;">减少</button>
                                <button type="button" class="layui-btn" @click="addCollection()" style="float:right;">增加</button>
                            </div>
                        </div>
                        <div class="layui-input-group center">
                            <div class="layui-input-split layui-input-prefix">
                                映射集名称
                            </div>
                            <input type="text" placeholder="映射集的唯一标识" class="layui-input">
                        </div>
                        <button type="button" class="layui-btn layui-bg-blue" style="margin:0 auto;display:block;margin-top:10px;width:300px;">添加此映射集</button>
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
    background-color: rgb(81, 199, 242);
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
    background-color: rgb(81, 199, 242);
}

.center {
    margin: 0 auto;
}

.collection .member {
    width: 80%;
    height: 100%;
    margin: 0 auto;
    margin-top: 10px;
}

.text-center {
    text-align: center;
}

.bigger-text {
    font-size: 20px;
    color: #333;
    font-weight: 400;
}

.center-item {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 50px;
}

.collection .items {
    margin-top: 10px;
    margin-bottom: 10px;
    border: #333 1px solid;
}
</style>