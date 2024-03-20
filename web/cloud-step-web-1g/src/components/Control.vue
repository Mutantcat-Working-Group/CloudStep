<script setup lang="ts">
import { ref, onMounted,defineEmits } from 'vue'
import request from '../utils/request'

const emit = defineEmits(['check'])

onMounted(() => {
    // 获得映射集列表
    request.request<any>(
        {
            url: '/collection/getall',
            method: 'get',
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            allCollections.value = res.data.data
        } else {
            alert('获取映射集失败。')
        }
    }).catch(() => {
        alert('获取映射集失败。')
    });

})

//（一）添加映射集操作

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

function pingAddress(newindex: any) {
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

// Ping全部
function pingAllAddress() {
    for (let index = 0; index < addCollections.value.length; index++) {
        pingAddress(index)
    }
}

// 移除无效项 （空项和ping结果为'Ping失败'和'timeout'的）
function removeInvalidItem() {
    for (let index = 0; index < addCollections.value.length; index++) {
        if (addCollections.value[index].address === "") {
            addCollections.value.splice(index, 1)
            index--
        }
    }
}

// 添加映射集
function addCollectionsF() {
    if (addCollectionsName.value === "") {
        alert('映射集名称不能为空。')
        return
    }
    // 先清空无效项
    removeInvalidItem()
    if (addCollections.value.length === 0) {
        alert('有效映射集内容不能为空。')
        return
    }
    request.request<any>(
        {
            url: '../collection/add',
            method: 'post',
            data: {
                name: addCollectionsName.value,
                urls: addCollections.value
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            alert('添加映射集成功。')
            allCollections.value.push(
                {
                    Id: res.data.id,
                    Name: addCollectionsName.value
                }
            )
            clearCollection()
        } else if (res.data.code === 3) {
            alert('映射集名称已存在。')
        }
        else {
            alert('添加映射集失败。')
        }
    }).catch(() => {
        alert('添加映射集失败。')
    });
}

//（二）管理已有映射集操作

// 当前后台的所有映射集
const allCollections: any = ref([])

// 当前选择的映射集id
const selectedCollectionId: any = ref("")

// 当前选择的映射集中的成员
const allGetedUrls: any = ref([{
    address: "未选择映射集",
    status: "未知"
}])

// 新添加的url
const newUrl: any = ref("")

function onSelcetdCollectionChange() {
    newUrl.value = ""
    if (selectedCollectionId.value == '') {
        allGetedUrls.value = [{
            address: "未选择",
            status: "未知"
        }]
        selectedCollectionId.value = ""
        return
    }
    request.request<any>(
        {
            url: '/collection/geturls',
            method: 'get',
            params: {
                id: selectedCollectionId.value
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            allGetedUrls.value = res.data.data
        } else {
            alert('获取映射集内容失败。')
        }
    }).catch(() => {
        alert('获取映射集内容失败。')
    });
}

function pingUrl(index: any) {
    request.request<any>(
        {
            url: '../ping',
            method: 'post',
            data: {
                url: allGetedUrls.value[index].address
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            allGetedUrls.value[index].status = res.data.ms
        } else {
            allGetedUrls.value[index].status = "Ping失败"
        }
    }).catch(() => {
        allGetedUrls.value[index].status = "Ping失败"
    });
}

function pingUrls() {
    for (let index = 0; index < allGetedUrls.value.length; index++) {
        pingUrl(index)
    }
}

function deleteCollection() {
    if (selectedCollectionId.value == '') {
        alert('请选择映射集。')
        return
    }
    // 显示确认框
    if (!confirm('确定删除映射集吗？')) {
        return
    }
    request.request<any>(
        {
            url: '../collection/delete',
            method: 'GET',
            params: {
                id: selectedCollectionId.value
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            alert('删除映射集成功。')
            for (let index = 0; index < allCollections.value.length; index++) {
                if (allCollections.value[index].Id == selectedCollectionId.value) {
                    allCollections.value.splice(index, 1)
                    break
                }
            }
            selectedCollectionId.value = ""
            onSelcetdCollectionChange()
        } else {
            alert('删除映射集失败。')
        }
    }).catch(() => {
        alert('删除映射集失败。')
    });

}

// 修改url

function updateUrl(index: any, address: any) {
    if (
        address.value == '' ||
        index.value == '' ||
        selectedCollectionId.value == ''
    ) {
        alert('地址不能为空。')
        return
    }
    request.request<any>(
        {
            url: '../url/update',
            method: 'post',
            data: {
                id: index,
                address: address
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            alert('修改链接成功。')
            onSelcetdCollectionChange()
        } else {
            alert('修改链接失败。')
        }
    }).catch(() => {
        alert('修改链接失败。')
    });
}

// 删除url
function deleteUrlfromCollection(index: any) {
    if (!confirm('确定删除链接吗？')) {
        return
    }
    if (selectedCollectionId.value == '') {
        alert('请选择映射集。')
        return
    }
    request.request<any>(
        {
            url: '../url/delete',
            method: 'GET',
            params: {
                id: index
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            alert('删除链接成功。')
            onSelcetdCollectionChange()
        } else {
            alert('删除链接失败。')
        }
    }).catch(() => {
        alert('删除链接失败。')
    });

}

// 添加url
function addUrltoCollection() {
    if (selectedCollectionId.value == '') {
        alert('请选择映射集。')
        return
    }
    if (newUrl.value == '') {
        alert('新链接地址不能为空。')
        return
    }
    request.request<any>(
        {
            url: '../url/add',
            method: 'post',
            data: {
                "parent": selectedCollectionId.value,
                "address": newUrl.value
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            alert('添加链接成功。')
            onSelcetdCollectionChange()
        } else {
            alert('添加链接失败。')
        }
    }).catch(() => {
        alert('添加链接失败。')
    });
}

//（三）修改密码操作

// 新密码
const newPassword = ref("")

function changePassword() {
    if (newPassword.value == '') {
        alert('新密码不能为空。')
        return
    }
    if (!confirm('确定修改密码吗？')) {
        return
    }
    request.request<any>(
        {
            url: '../change',
            method: 'post',
            data: {
                password: newPassword.value
            },
            headers: {
                "Token": window.sessionStorage.getItem('token')
            }
        }
    ).then((res) => {
        if (res.data.code === 0) {
            alert('修改密码成功。')
            emit('check')
        } else {
            alert('修改密码失败。')
        }
    }).catch(() => {
        alert('修改密码失败。')
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
                        <h2 class="info">管理映射集</h2>
                        当前映射集：
                        <select v-model="selectedCollectionId" @change="onSelcetdCollectionChange">
                            <option value="">请选择</option>
                            <option v-for="(item, index) in allCollections" :key="index" :value="item.Id">{{ item.Name
                                }}
                            </option>
                        </select>
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
                            <div class="items" style="width:100%" v-for="(item, index) in allGetedUrls" :key="index">
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
                                        <div class="text-center center-item">{{ item.status == undefined ? "未知" :
                            item.status
                                            }}</div>
                                    </div>
                                    <div class="layui-col-xs3">
                                        <div class="text-center center-item">
                                            <button type="button" class="layui-btn layui-btn-primary layui-btn-sm"
                                                @click="updateUrl(item.id, item.address)">提交修改</button>
                                            <button type="button" class="layui-btn layui-btn-primary layui-btn-sm"
                                                @click="pingUrl(index)">服务端Ping</button>
                                            <button type="button" class="layui-btn layui-btn-primary layui-btn-sm"
                                                @click="deleteUrlfromCollection(item.id)">删除</button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="layui-input-group center" style="margin-top:5px;">
                            <div class="layui-input-split layui-input-prefix">
                                新链接地址
                            </div>
                            <input type="text" placeholder="abc.def:1234/xxx" class="layui-input"
                                style="caret-color: black;" v-model="newUrl" />
                            <div class="layui-input-suffix">
                                <button class="layui-btn layui-btn-primary" @click="addUrltoCollection()">添加链接</button>
                                <button class="layui-btn layui-btn-primary" @click="pingUrls()">Ping全部</button>
                                <!-- <button class="layui-btn layui-btn-primary" @click="">启用全部</button> -->
                                <button class="layui-btn layui-btn-primary"
                                    @click="onSelcetdCollectionChange()">刷新</button>
                                <button class="layui-btn layui-btn-primary" @click="deleteCollection()">删除映射集</button>
                            </div>
                        </div>
                        <h2 class="control">新建映射集</h2>
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
                                            <button type="button" class="layui-btn layui-btn-primary layui-btn-sm"
                                                @click="pingAddress(index)">服务端Ping</button>
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
                        <div class="layui-input-group center" style="margin-top:5px;">
                            <div class="layui-input-split layui-input-prefix">
                                映射集名称
                            </div>
                            <input type="text" placeholder="映射集的唯一标识" class="layui-input" style="caret-color: black;"
                                v-model="addCollectionsName" />
                            <div class="layui-input-suffix">
                                <button class="layui-btn layui-btn-primary" @click="addCollectionsF()">添加此映射集</button>
                                <button class="layui-btn layui-btn-primary" @click="pingAllAddress()">Ping全部</button>
                                <button class="layui-btn layui-btn-primary" @click="removeInvalidItem()">移除无效项</button>
                                <button class="layui-btn layui-btn-primary" @click="clearCollection()">重置</button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="layui-tab-item">
                    <div class="collection">
                        <h2 class="control">自助设置</h2>

                        <h2 class="info">自助管理</h2>

                    </div>
                </div>
                <div class="layui-tab-item">
                    <div class="collection">
                        <h2 class="control">代理设置</h2>

                        <h2 class="info">代理管理</h2>
                    </div>
                </div>
                <div class="layui-tab-item">
                    <div class="collection">
                        <h2 class="control">通用设置</h2>
                        <h2 class="control">密码设置</h2>
                        <div class="layui-input-group center">
                            <div class="layui-input-split layui-input-prefix">
                                新密码
                            </div>
                            <input style="caret-color: black;" type="text" placeholder="重置后需要重新登录" class="layui-input" v-model="newPassword">
                            <div class="layui-input-suffix">
                                <button class="layui-btn layui-btn-primary" @click="changePassword()">修改密码</button>
                            </div>
                        </div>
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
    color: #1a393f;
    font-weight: 400;
    margin-top: 10px;
    margin-bottom: 10px;
    border: #d5e9f7 1px solid;
    border-radius: 5px;
    padding: 10px;
    text-align: center;
    background-color: #CCCCCC;
}

.collection .info {
    font-size: 20px;
    color: #1a393f;
    font-weight: 400;
    margin-top: 10px;
    margin-bottom: 10px;
    border: #d5e9f7 1px solid;
    border-radius: 5px;
    padding: 10px;
    text-align: center;
    background-color: #CCCCCC;
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