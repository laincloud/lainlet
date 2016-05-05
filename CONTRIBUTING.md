## 代码结构

```
root
 |_api  // 第一层, v2目录下,每个api是一个单独的.go文件
 |  |_ v2/         // v2版本的api
 |  |_ event.go    // 处理watch功能的event相关函数, 弃用旧版本的eventsource package
 |  |_ server.go   // 初始化http server framework, 并实现些通用的 middleware 和 function
 |  |_ maker.go    // 定义一个新的api需要实现的interface
 |
 |_auth // 基于watcher实现的auth功能, 默认开启,可通过-noauth参数关闭
 |
 |_watcher  //第二层
 |  |_config       // `/lain/config/`
 |  |_podgroup     // `/lain/deployd/pod_group/<appname>`
 |  |_container    // 对`/lain/deployd/pod_group/`下的数据创建<nodename>/<containerid>索引后的数据
 |  |_depends      //  `/lain/deployd/depends/pods`
 |  |_nodes        // `/lain/nodes/nodes`
 |  |
 |  |_watcher.go   // 通用watcher模块, 监听store并广播
 |  |_sender.go    // 通用广播模块, 负责接收watch请求，并广播
 |  |_cacher.go    // 通用cache模块,从etcd拿到的数据cache到这里
 |
 |
 |_store  //第三层
 |  |_store.go   // store 接口
 |  |_etcd        // etcd 存储模块
 |
 |_client  // lainlet的客户端库
 |_vendor  // 依赖库

```

## 功能开发

1. 在`api/v2/`目录下创建新的`.go`文件, 以`newapi.go`为例
2. 在`newapi.go`文件中，定义api使用的数据结构, 并实现API及Watcher接口

    ```golang
    type API interface {
            // decode the data
            Decode([]byte) error

            // encode the data
            Encode() ([]byte, error)

            // the http route uri for this api
            URI() string

            // which watcher want to use
            WatcherName() string

            // the key used to watch,
            // eg. return '/lain/config/vip' for coreinfoWatcher,
            //     return 'console' as appname for coreifnoWacher and podgroupWatcher
            Key(r *http.Request) (string, error)

            // create new api by data
            Make(data map[string]interface{}) (API, error)
    }

    // 如果想禁用watch功能, 可实现BanWatch接口, 写个空函数就行
    type BanWatcher interface {
            BanWatch()
    }

    ```

3. 在main.go的main函数下面注册新API, eg.`server.Register(new(apiv2.NewAPI))`。注册函数会自动给uri加上`/v2/`前缀


### 若有什么特殊需求，可用原生的方法，步骤如下:

1. 在`api/v2/`目录下创建新的`.go`文件, 在其中实现一个函数,形如:
    ```golang
    // 函数的参数用到什么写什么，目前支持 `http.ResponseWriter`,`*http.Request`,`*api.EventSource`,`context.Context`,`martini.Context`
    func VipWatcher(w http.ResponseWriter, r *http.Request, es *api.EventSource, ctx context.Context) {
        // ....
    }
    ```

2. 在`main.go`最下面定义路由, 加上一行代码,如:
   ```golang
	server.Get("/vipconfwatcher/", config.VipWatcher)
   ```
3. `go build`构建测试


