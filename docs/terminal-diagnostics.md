# 终端诊断

默认关闭详细诊断。排查终端输出积压或断连时，可以为 `xpanel.service` 设置环境变量 `XPANEL_TERMINAL_DEBUG=1` 并重启面板进程。重启会结束面板管理的终端，操作前请处理运行中的后台终端。停止排查时删除该环境变量并再次重启面板。

例如使用 systemd drop-in：

```ini
[Service]
Environment=XPANEL_TERMINAL_DEBUG=1
```

执行 `systemctl daemon-reload` 和 `systemctl restart xpanel` 后生效，可用 `journalctl -u xpanel` 查看日志。详细日志记录会话 ID、累计输出量、队列深度、背压等待时间和较慢的 WebSocket 写入，不记录终端输入或输出内容。断连和会话删除原因仍会在普通日志中记录。
