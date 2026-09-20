# comment/rpc

评论域的 RPC 服务层，承载本域业务规则与事务编排，供 api 层及其他域调用。

骨架阶段占位：尚无业务实体时不预生成 proto 与 pb 代码。后续由 goctl 生成：

```bash
cd server/app/comment/rpc
goctl rpc protoc comment.proto --go_out=. --go-grpc_out=. --zrpc_out=.
```
