# X-Panel 工作日志

## 2026-08-17

### 完成内容

- [x] 通知降噪：去掉 logrus Error hook 和「任意写接口失败」通知
- [x] 事件名统一小写；计划任务失败能命中偏好；安静事件的 `show_badge=false` 能写入 SQLite
- [x] 默认规则允许全关，不再被零值重置
- [x] 登录失败、证书自动续签失败改为显式事件；成功类任务默认不打红点
- [x] 捆绑 Agent 安装跳过 tar 包根目录 `./`，避免 `invalid archive entry name`

### 关键决策

- 先修站内 inbox，不加邮件/Webhook
- 旧的 `operation.failed` / `system.log.error` 记录仍可显示，但不再产生、也不再出现在默认偏好里

### 遗留问题

- 仍是 30 秒轮询，没有站外通道和保留策略
- 登录失败每次尝试都会写一条通知，暴力破解时可能刷屏

### 下一步计划

- 发布 v0.7.85；节点升级后即可安装捆绑 Agent
- 需要时再加 WebSocket 推送和独立 Alert 通道

## 2026-08-13

### 完成内容

- [x] 对齐三角色设计：X-Panel 安装、配置保存和升级写入 `node_role: xpanel`
- [x] 事务升级只合并该字段，保留 UUID、密钥和未知字段；缺配置不造文件；损坏 YAML 不阻断升级
- [x] 安装脚本首次配置写入 `node_role`，已有 `config.yml` 升级时同样只合并该字段
- [x] 更新现行文档：`docs/dashboard-agent-xpanel.md`、`docs/nezha-agent.md`、`XPANEL_COLLABORATION.md`

### 关键决策

- 升级不整文件替换 `config.yml`，也不从发布包套用配置
- OpenWrt 打包与 `node_role: openwrt` 仍不在本产品路径处理
- 本次不发 Agent 标签、不发 X-Panel 版本

### 遗留问题

- 认识 `node_role` 的 Agent 仍在本地工作区，生产节点要等先发 Agent、再发 X-Panel 补丁后才会带上角色

### 下一步计划

- 按 `RELEASE.md` 先发带该字段的 Agent，再发绑定该 Agent 的 X-Panel 补丁
