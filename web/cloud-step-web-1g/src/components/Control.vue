<script setup lang="ts">
import { ref } from 'vue'
import request from '../utils/request'

const addCollections = ref([{
    address: "",
    status: "未知"
}])

const addCollectionsName = ref("")

function addCollection() {
    addCollections.value.push({
        address: "",
        status: "未知"
    })
}

function removeCollection() {
    addCollections.value.pop()
}

function clearCollection() {
    addCollections.value = [{
        address: "",
        status: "未知"
    }]
    addCollectionsName.value = ""
}

function pingAddress(newindex:any){
    request.request<any>(
        {
            url: '../ping',
            method: 'post',
            data: {
                url: addCollections.value[newindex].address
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            addCollections.value[newindex].status = res.data.ms
        } else {
            addCollections.value[newindex].status = "Ping失败"
        }
    }).catch(() => {
        addCollections.value[newindex].status = "Ping失败"
    });

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
            <div class="layui-tab-content" style="height:95%;overflow:auto;">
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
                                        <div class="text-center bigger-text">延迟</div>
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
                                                placeholder="abc.def:1234/xxx" class="layui-input"
                                                style="caret-color: black;" v-model="item.address"></div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item">{{ item.status }}</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item">
                                            <button type="button"
                                                class="layui-btn layui-btn-primary layui-btn-sm" @click="pingAddress(index)">服务端Ping</button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div style="height:38px;">
                                <button type="button" class="layui-btn" @click="removeCollection()"
                                    style="float:left;">减少</button>
                                <button type="button" class="layui-btn" @click="addCollection()"
                                    style="float:right;">增加</button>
                            </div>
                        </div>
                        <div class="layui-input-group center">
                            <div class="layui-input-split layui-input-prefix">
                                映射集名称
                            </div>
                            <input type="text" placeholder="映射集的唯一标识" class="layui-input" style="caret-color: black;" v-model="addCollectionsName"/>
                            <div class="layui-input-suffix">
                                <button class="layui-btn layui-btn-primary">添加映射集</button>
                                <button class="layui-btn layui-btn-primary" @click="clearCollection()">重置映射集</button>
                            </div>
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
                        <h2 class="control">通用设置</h2>
                        <h2 class="control">密码设置</h2>
                        <h2 class="info">关于</h2>
                        <div class="text-center">
                            <p>安全的、高性能的、可独立部署的、代理（反向代理）的、自助代理的、负载均衡的、可持久化的服务地址管理工具。</p>
                            <p>https://www.mutantcat.org/software/cloudstep</p>
                            <p>Copyright © 2024 Mutantcat ALL Rights Reserved.</p>
                            <p>版本：1.0.20240317</p>
                        </div>
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
    width: 100vw;
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
    color: #225864;
    font-weight: 400;
    margin-top: 10px;
    margin-bottom: 10px;
    border: #d5e9f7 1px solid;
    border-radius: 5px;
    padding: 10px;
    text-align: center;
    background-color: rgb(54, 201, 255);
}

.collection .info {
    font-size: 20px;
    color: #225864;
    font-weight: 400;
    margin-top: 10px;
    margin-bottom: 10px;
    border: #d5e9f7 1px solid;
    border-radius: 5px;
    padding: 10px;
    text-align: center;
    background-color: rgb(54, 201, 255);
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