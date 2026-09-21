# features/

业务模块落位目录。按前端工程约定：

- `pages/` 只放路由页面，页面专属的业务组件放在本目录对应的 `features/<module>/` 下。
- 被多个业务复用的组件才放 `components/`。
- 服务端数据由页面或请求层持有，跨页面共享的客户端状态才进 `store/`。

骨架阶段暂无业务模块，后续功能变更（如 feed、player、comment）在此建立各自目录。
