import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';

class Request {
    private instance: AxiosInstance | undefined;

    constructor(config: AxiosRequestConfig) {
        this.instance = axios.create(config);

        // 请求拦截器
        this.instance.interceptors.request.use(
            (config) => {
                // 在这里可以添加一些请求前的处理逻辑
                return config;
            },
            (error) => {
                return Promise.reject(error);
            }
        );

        // 响应拦截器
        this.instance.interceptors.response.use(
            (response) => {
                // 在这里可以添加一些响应后的处理逻辑
                return response;
            },
            (error) => {
                return Promise.reject(error);
            }
        );
    }

    request<T>(config: AxiosRequestConfig): Promise<T> {
        return new Promise<T>((resolve, reject) => {
            this.instance?.request<any, T>(config)
                .then((res) => {
                    resolve(res);
                })
                .catch((err) => {
                    reject(err);
                });
        });
    }
}

const request = new Request({
    timeout: 2000, // 设置请求超时时间
    headers: { 'Content-Type': 'application/json;charset=utf-8' } // 设置自定义请求头
});

export default request;