# tsp后台服务

## 技术栈
### 后端
- golang
- gin
- postgresql
- xorm

### 前端
- vue
- iview/ant designer
  
## API
```c
$ curl -H "Content-Type:application/json" -X POST --data '{"page":1}' http://localhost:8080/api/list

{"pagecnt":1,"pagesize":10,"pageindex":1,"data":[{"ip":"127.0.0.1:52388","imei":"865501043954677","phone":"13246607267"},{"ip":"127.0.0.1:52392","imei":"865501043897165","phone":"13246607267"}]}
```