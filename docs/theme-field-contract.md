# X-Panel 主题字段契约

版本：与 `ui-design-system.md` 1.3.3 配套。  
状态：草案，**尚未冻结**为公开 schema。解析器实现必须以本表为准，不得写「由实现决定」。

用户 overlay 标量**跨深浅共用**。主题包 `palette.*` **深浅分列**。将来分模式覆盖只能新增 `overridesByTheme.<id>.byMode`，不得把已有标量改成对象。

恢复分组路径用于「按组恢复」：删除该组已知字段，以及隔离袋中前缀匹配该组的未知项。无法归组的隔离项只在全部恢复时清除。

禁止字段（识别到即剥除并报告路径，不当未知声明保留）：`derive.cache`、`derive.computed`、`derive` 下名称以 `derived` 开头的字段。不得按任意位置的名称前缀误删未来合法字段。

本表中的“回退”“派生”“夹紧”分别按 C.1、C.2 执行；不得另选公式。完整模式缺少必选字段时拒绝整个主题包，不安装半个主题。公开 schema 仍未冻结；本轮确定的 v1 数值和迁移视觉须经示例与解析器验证后冻结。

---

## 表 A — 文件契约

### A.1 外壳（两种 kind 共用）

| JSON 路径 | 类型 | 必选 | 格式/范围 | 缺失 | 非法/未知 | 模式 | 引入 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `kind` | string | 是 | `x-panel.theme` 或 `x-panel.appearance` | 拒绝 | 拒绝 | 全局 | 1.0 |
| `schemaVersion` | integer | 是 | ≥1 | 拒绝 | 主版本不支持则拒绝 | 全局 | 1.0 |
| `schemaMinor` | integer | 否 | ≥0，缺省 0 | 0 | 非整数则 0 并警告 | 全局 | 1.0 |
| `exportedAt` | string | 否 | ISO-8601 | 空 | 忽略 | 全局 | 1.0 |

外观文件校验顺序：外层 `schemaVersion` → 内层 `preference.schemaVersion`。当前支持组合：外层 1 + 内层 1。只支持主版本 1 时，`schemaVersion === 1` 合法；不得因更高 `schemaMinor` 拒绝。

### A.2 主题包 `x-panel.theme`

| JSON 路径 | 类型 | 必选 | 格式/范围 | 缺失 | 非法/未知 | 模式 | 引入 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `id` | string | 是 | `[a-z0-9][a-z0-9-]{1,63}` | 拒绝 | 拒绝 | 全局 | 1.0 |
| `name` | string | 是 | 1–64 字 | 拒绝 | 截断并警告 | 全局 | 1.0 |
| `version` | string | 是 | 主题内容 semver | 拒绝 | 警告，仍可读 | 全局 | 1.0 |
| `palette.dark` / `palette.light` | object | 是 | 见色键 | 拒绝该模式 | 见各键 | 分模式 | 1.0 |
| `palette.<mode>.bg` | string | 是 | #RRGGBB | 拒绝整包 | 回退 `#111318`/`#E4E6EC` 并警告 | 分模式 | 1.0 |
| `palette.<mode>.surface` | string | 是 | 同上 | 拒绝该模式 | 回退并警告 | 分模式 | 1.0 |
| `palette.<mode>.elevated` | string | 否 | 同上 | C.2 elevated | 回退 C.2 elevated | 分模式 | 1.0 |
| `palette.<mode>.overlay` | string | 否 | 同上 | C.2 overlay | 回退 C.2 overlay | 分模式 | 1.0 |
| `palette.<mode>.inset` | string | 否 | 同上 | C.2 inset | 回退 C.2 inset | 分模式 | 1.0 |
| `palette.<mode>.sidebar` | string | 否 | 同上 | C.2 sidebar | 回退 C.2 sidebar | 分模式 | 1.0 |
| `palette.<mode>.header` | string | 否 | 同上 | C.2 header | 回退 C.2 header | 分模式 | 1.0 |
| `palette.<mode>.input` | string | 否 | 同上 | 派生 inset | 回退 inset | 分模式 | 1.0 |
| `palette.<mode>.tableHeader` | string | 否 | 同上 | 派生 inset | 回退 inset | 分模式 | 1.0 |
| `palette.<mode>.auth` | string | 否 | 同上 | 派生 bg | 回退 bg | 分模式 | 1.0 |
| `palette.<mode>.text` | string | 是 | hex | 拒绝该模式 | 回退 | 分模式 | 1.0 |
| `palette.<mode>.textSecondary` | string | 否 | hex | 派生 text/muted 之间 | 回退 | 分模式 | 1.0 |
| `palette.<mode>.textMuted` | string | 是 | hex | 拒绝该模式 | 回退 | 分模式 | 1.0 |
| `palette.<mode>.border` | string | 是 | #RRGGBB 或 C.1 rgba | 拒绝整包 | C.1 安全 border | 分模式 | 1.0 |
| `palette.<mode>.borderLight` | string | 否 | 同上 | 派生 border | 回退 | 分模式 | 1.0 |
| `palette.<mode>.borderHover` | string | 否 | 同上 | 派生 border | 回退 | 分模式 | 1.0 |
| `palette.<mode>.accent` | string | 是 | `#RRGGBB` | 拒绝该模式 | 回退官方预设 | 分模式 | 1.0 |
| `palette.<mode>.accentSecondary` | string | 否 | `#RRGGBB` | 策略 v1 从 accent 派生 | 回退派生 | 分模式 | 1.0 |
| `palette.<mode>.success` `.warning` `.danger` `.info` | string | 是 | `#RRGGBB` | 拒绝该模式 | 回退安全色 | 分模式 | 1.0 |
| `derive.strategy` | string | 否 | 已知 id，现仅 `v1` | `v1` | 运行回退 v1 并警告，**导出保留原 id** | 全局 | 1.0 |
| `derive.cache` 等禁止字段 | — | 禁止 | — | — | 剥除并报告路径 | 全局 | 1.0 |
| `defaults.mode` | enum | 否 | `dark` `light` `auto` | `dark` | `dark` 并警告 | 全局 | 1.0 |
| `defaults.density` | enum | 否 | `compact` `default` `comfortable` | `default` | `default` | 全局 | 1.0 |
| `defaults.uiFont` | enum | 否 | `system` `inter` `noto` `lxgw` | `system` | `system` | 全局 | 1.0 |
| `defaults.sidebarWidth` | enum | 否 | `narrow` `default` `wide` | `default` | `default` | 全局 | 1.0 |
| `defaults.headerHeight` | enum | 否 | `compact` `default` `comfortable` | `default` | `default` | 全局 | 1.0 |
| `defaults.radius` | enum | 否 | `sharp` `default` `rounded` | `default` | `default` | 全局 | 1.0 |
| `defaults.transparency` | boolean | 否 | | `false` | 非法则 `false` | 全局 | 1.0 |
| `defaults.iconSet` | enum | 否 | `outline` `solid` | `outline` | `outline` | 全局 | 1.0 |
| `defaults.termFont` | string | 否 | 登记族名 | `jetbrains` | 回退 | 全局 | 1.0 |
| `defaults.termFontSize` | number | 否 | 整数 10–24 | `14` | 夹紧到范围 | 全局 | 1.0 |
| `defaults.termBgOpacity` | number | 否 | 0–1 | `1` | 夹紧 | 全局 | 1.0 |
| `defaults.chromeTexture` | enum | 否 | `none` `ribbon` `galaxy` `starfield` | `none` | `none` | 全局 | 1.0 |
| `defaults.variants.sidebar` | enum | 否 | `marker` `block` | `marker` | `marker` | 全局 | 1.0 |
| `defaults.variants.subnav` | enum | 否 | `line` `block` `pill` | `block` | `block` | 全局 | 1.0 |
| `defaults.variants.card` | enum | 否 | `flat` `outline` `raised` | `outline` | `outline` | 全局 | 1.0 |
| `defaults.variants.iconContainer` | enum | 否 | `none` `tile` | `none` | `none` | 全局 | 1.0 |
| `tokens.materials.surfaceOpacity` | number | 否 | 0–1 | `0.88` | 夹紧 | 全局 | 1.0 |
| `tokens.materials.blurPx` | number | 否 | 0–20 | `0` | 夹紧；减少动效时强制 0 | 全局 | 1.0 |
| `tokens.materials.sidebarOpacity` | number | 否 | 0–1 | `0.78` | 夹紧 | 全局 | 1.0 |
| `tokens.materials.headerOpacity` | number | 否 | 0–1 | `0.82` | 夹紧 | 全局 | 1.0 |
| `tokens.materials.overlayOpacity` | number | 否 | 0–1 | `0.9` | 夹紧 | 全局 | 1.0 |
| `tokens.motion.hoverMs` | number | 否 | 0–240 | `100` | 夹紧 | 全局 | 1.0 |
| `tokens.motion.pageMs` | number | 否 | 0–240 | `80` | 夹紧 | 全局 | 1.0 |
| `tokens.motion.menuMs` | number | 否 | 0–240 | `140` | 夹紧 | 全局 | 1.0 |
| `tokens.motion.drawerMs` | number | 否 | 0–240 | `180` | 夹紧 | 全局 | 1.0 |
| `tokens.motion.ease` | string | 否 | CSS easing | 规范默认贝塞尔 | 回退默认 | 全局 | 1.0 |
| `surfaces.card.topEdge` | boolean | 否 | | `true`，所有 card 消费者共用 | 非法则按缺失 | 全局 | 1.0 |
| `surfaces.inset.hoverStrength` | number | 否 | 0–1 | v1 默认 `1` | 夹紧；`0` = 无悬停叠层 | 全局 | 1.0 |
| `terminal.dark` / `terminal.light` | string | 否 | 已登记终端预设 id | 由 v1 从 palette 生成 | 回退 `default`/`paper` | 分模式 | 1.0 |
| `editor.dark` / `editor.light` | string | 否 | `vs-dark` `vs` 或登记 id | `vs-dark`/`vs` | 回退 | 分模式 | 1.0 |
| `charts.categorical` | string[] | 否 | 3–12 个 `#RRGGBB` | v1 从 accent 生成并校验可区分 | 回退安全序列 | 全局 | 1.0 |
| `charts.sequential` | string[] | 否 | 3–8 个 `#RRGGBB` | v1 从 accent 生成 | 回退 | 全局 | 1.0 |
| `fonts.uiUrl` / `fonts.monoUrl` | string | 否 | HTTPS 直链 woff2/ttf/otf 二进制，禁止 CSS | 无 | 不合规剥除并报告；加载失败回退 | 全局 | 1.0 |
| `assets.chromeImageUrl` | string | 否 | `https:` 图 | 无 | 非 https 剥除 | 全局 | 1.0 |

密度/圆角/侧栏宽度的**数值表**（32/36/40、圆角 px 等）是共享查找表，不进主题包。主题只选 preset key。

### A.3 外观 overlay `x-panel.appearance`

| JSON 路径 | 类型 | 必选 | 格式/范围 | 缺失 | 非法/未知 | 模式 | 引入 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `preference.schemaVersion` | integer | 是 | 与外层组合见 A.1 | 拒绝 | 不支持则拒绝整包，保留原外观 | 全局 | 1.0 |
| `preference.themeId` | string | 是 | slug | 拒绝 | 运行回退 atelier，**持久化保留原值** | 全局 | 1.0 |
| `preference.mode` | enum | 否 | `dark` `light` `auto` | `dark` | `dark` | 全局 | 1.0 |
| `preference.reduceMotion` | boolean | 否 | | `false` | `false` | 全局 | 1.0 |
| `preference.keepPersonalPrefsAcrossThemes` | boolean | 否 | | `false` | `false` | 全局 | 1.0 |
| `preference.overridesByTheme.<id>.accentKey` | string | 否 | 预设 key 或 `custom` | 主题包该模式 accent | 回退主题包 | 跨模式 | 1.0 |
| `preference.overridesByTheme.<id>.accentCustom` | string | 否 | `#RRGGBB` | 无 | 剥除 | 跨模式 | 1.0 |
| `preference.overridesByTheme.<id>.accentSecondary` | string | 否 | `#RRGGBB` | 按 C.2 副色优先级 | 剥除 | 跨模式 | 1.0 |
| `preference.overridesByTheme.<id>.surfaces.card.topEdge` | boolean | 否 | `false` 关闭 | 主题 surfaces.card.topEdge，再缺则 true | 按缺失 | 跨模式 | 1.0 |
| `preference.overridesByTheme.<id>.surfaces.inset.hoverStrength` | number | 否 | 0–1；`0` 有效 | 主题同字段，再缺则 1 | 夹紧 | 跨模式 | 1.0 |
| `preference.overridesByTheme.<id>.chromeImageMode` | enum | 否 | `none` `url` `upload` | `none` | `none` | 跨模式 | 1.0 |
| `preference.overridesByTheme.<id>.chromeImageUrl` | string | 否 | https 或剥离 | 空 | upload/data/file 导出剥除 | 跨模式 | 1.0 |

未列在表中的 `overridesByTheme.<id>.*`：隔离保存，警告路径，不解释、不加载资源。

下表也是 A.3 的组成部分。路径统一为 `preference.overridesByTheme.<id>.<字段>`；均可选、跨模式、引入版本 1.0。不存在“等”所代表的隐式字段。非法类型按缺失并警告；未知枚举运行按缺失、隔离保留原值。

| 字段 | 类型/合法值 | 缺失时的确切来源 |
| --- | --- | --- |
| density | compact/default/comfortable | defaults.density |
| uiFont | system/inter/noto/lxgw，或 `upload:<资源id>` | defaults.uiFont |
| sidebarWidth | narrow/default/wide | defaults.sidebarWidth |
| headerHeight | compact/default/comfortable | defaults.headerHeight |
| radius | sharp/default/rounded | defaults.radius |
| card | flat/outline/raised | defaults.variants.card |
| sidebarVariant | marker/block | defaults.variants.sidebar |
| subnav | line/block/pill | defaults.variants.subnav |
| iconSet | outline/solid | defaults.iconSet |
| iconContainer | none/tile | defaults.variants.iconContainer |
| transparency | boolean | defaults.transparency |
| surfacePreset | graphite/abyss/void/tinted/cosmos/warm | 不覆盖主题色板；合法值见 C.3 |
| termTheme | default/dracula/onedark/solarized/monokai/paper/lumen-dark/lumen-light | 当前模式 terminal；再缺走 C.2 |
| termFont | jetbrains/firacode/cascadia/consolas/system，或 `upload:<资源id>` | defaults.termFont |
| termFontSize | 整数 10–24 | defaults.termFontSize |
| termBgOpacity | number 0–1 | defaults.termBgOpacity |
| termWallpaper | none | none；保留旧字段，不赋予新含义 |
| chromeTexture | none/ribbon/galaxy/starfield | defaults.chromeTexture |
| termFollowChrome | boolean | 有有效面板纹理或图片时 true，否则 false |
| termImageMode | none/url/upload | none |
| termImageUrl | URL 字符串 | 空；模式规则同 chromeImageUrl |

`chromeImageMode` 缺失时：主题 assets.chromeImageUrl 存在则 url，否则 none；URL 缺失继承主题 assets.chromeImageUrl。显式 none 禁止显示主题图片。termFollowChrome=true 时忽略独立终端图片的运行时效果，但保留其数据。

旧纹理 diagonal→ribbon、dots→starfield、grain/grid→galaxy，属于明示迁移并报告路径。旧 HTTP 图片 URL 导入时隔离保存并警告、不请求；再次分享时剥除并报告有损路径。不得静默替换为 HTTPS。upload 模式的图片导出为 none 并清 URL；上传字体引用导出为 system，不修改本地原配置。

---

## 表 B — 解析与消费

| 属性 | 主题路径 | overlay 路径 | 派生依赖 | 恢复分组 | 导出 | 运行时消费者 | 兼容验证 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 主色 | `palette.<mode>.accent` | `…accentKey` / `accentCustom` | 变化后重算 hover、muted、onAccent、未覆盖的副色、未覆盖的图表首色、顶边色。**不**重写显式副色 | `harmony.accent` | 保留原值 | `--xp-accent*`、按钮、进度、card 顶边色 | 缺失、custom hex、浅色对比度 |
| 副色 | `palette.<mode>.accentSecondary` | `…accentSecondary` | 用户副色 > 用户主色触发的派生 > palette 副色 > v1 派生，见 C.2 | `advanced.secondary` | 保留显式 hex | `--xp-accent-secondary`、双指标 | 缺失、显式、改主色后仍在 |
| hover/muted/onAccent | 无文件字段 | 无 | 始终由当前有效主色 + strategy 计算 | （无独立恢复） | 禁止写入文件 | 按钮 hover、muted 底 | 禁止缓存往返 |
| 表面色 | `palette.<mode>.surface` 等 | `…surfacePreset` 可整体替换家族 | 表面家族覆盖后重算 page/card/inset/float 底 | `harmony.surface` | 保留 preset key | `--xp-bg-*`、四种配方 | 深浅都生效 |
| card 顶边 | `surfaces.card.topEdge` | `…surfaces.card.topEdge` | 无 | `advanced.recipe` | 保留布尔 | 公共 card 配方，非业务页 | 缺失=收口基线；`false`=关 |
| inset 悬停 | `surfaces.inset.hoverStrength` | 同左 overlay | 只影响 hover 叠层不透明度 | `advanced.recipe` | 保留数字 | inset hover | `0`、缺失、`1` |
| 密度/字号 | `defaults.density` | `…density` | 查共享表 | `harmony.density` | 保留 enum | `--xp-control-height` 等 | 三档无裁切 |
| 圆角 | `defaults.radius` | `…radius` | 查共享表 | `harmony.radius` | 保留 enum | `--xp-radius-*` | 胶囊例外仍全圆 |
| 侧栏/顶栏尺寸 | `defaults.sidebarWidth` `headerHeight` | 同名 overlay | 查共享表 | `advanced.layout` | 保留 enum | 布局壳 | 折叠宽 64 不变 |
| 字体 | `defaults.uiFont` `termFont` `termFontSize` | 同名 | 无 | `harmony.fonts` | 不含上传二进制 | `html` font、xterm | 失败回退 system |
| 材质 opacity/blur | `tokens.materials.*` | `…transparency` 及以后分项 | 透明关时用 solid 替代，不垫底 | `harmony.materials` | 保留 | 侧栏顶栏 card float 背景层 | 0、1、false |
| 动效 | `tokens.motion.*` | `reduceMotion` | 减少动效时时长→0 | 无障碍优先于组 | 保留 | 过渡 | reduceMotion |
| 变体 | `defaults.variants.*` | `card` `sidebarVariant` `subnav` `iconContainer` | 无 | `advanced.variants` | 保留 | `data-*-variant` | 非法回退 |
| 终端/编辑器 | `terminal.*` `editor.*` | A.3 的终端字段；首期无 editor overlay | 编辑器底跟随终端透明度 | `advanced.terminal` | 保留 id | xterm / monaco 主题 API | 不重建会话 |
| 图表 | `charts.*` | 无独立字段 | 用户主色重算首色/连续色；否则继承主题数组，再缺走 C.2；副色不阻断 | — | 保留作者数组，不写派生结果 | ECharts | 可区分性 |
| 未知声明 | — | 隔离袋，路径=原 JSON 路径 | 不计算 | 前缀匹配组则随组删；否则仅全部恢复 | 再导出原路径；禁止字段除外 | 无 | 往返、恢复组、全部恢复 |

### 三个细项（路径在本表已定）

| 属性 | 类型 | 语义 |
| --- | --- | --- |
| `surfaces.inset.hoverStrength` | 可选 number 0–1 | `0` 无悬停叠层；缺失走 v1（视为 1） |
| `surfaces.card.topEdge` | 可选 boolean | `false` 关闭；主题缺失=true；overlay 缺失先继承主题，不改其它边框 |
| `overridesByTheme.<id>.accentSecondary` 或主题 `palette.<mode>.accentSecondary` | 可选 hex | 缺失则派生；显式值不随主色清除；跨深浅共用 overlay 标量 |

---

## 示例包对照

文件：`frontend/src/theme/examples/atelier.theme.json`、`lumen.theme.json`。

| 类别 | 显式写入示例包 | v1 补齐 | 第 14 节有意迁移（不是包字段） |
| --- | --- | --- | --- |
| 色板 | 深浅 `bg/surface/sidebar/header/input/text*/border*/accent*/status` 全写 | elevated/overlay/inset 若未写才派生；本示例全写 | — |
| 差异必须显式 | atelier：`transparency false`、`chromeTexture ribbon`、`uiFont lxgw`、`termFont consolas`、`termBgOpacity 0.5`、`variants.marker/block/outline/none`、材料 opacity、motion 默认 | hover/muted/onAccent | 首页局部顶边收到公共 `surfaces.card.topEdge`（示例写 `true`） |
| lumen 必须显式 | `transparency true`、更高 blur/opacity、`headerHeight comfortable`、`radius rounded`、`uiFont noto`、`iconSet solid`、`variants.block/pill/raised/tile`、独立 motion | 同上 | 不得靠 id=`lumen` 隐式套这些值 |
| 细项 | 示例省略 hoverStrength，主题包不包含用户 overlay | hoverStrength=1；无用户主色覆盖时副色继承包内 palette，不能据此证明副色派生 | topEdge 对全部 card 收口为开 |

验证：两个包不得出现 `if (id === 'lumen')` 才能看出差异；三个细项不得改变字段类型，也不得要求改已收口业务页。

示例不是当前 TS 定义的无损导出，属有意配色迁移（验收以解析结果为准，不保证旧截图）：

| 项 | 旧运行时 | 示例包 | 原因 |
| --- | --- | --- | --- |
| atelier 浅色主色 | steel `#7AA2FF` | `#5B8CFF` | `#7AA2FF` 在 `#F4F5F8` 上约不够 4.5；C.1 浅色安全 accent 同步为此值 |
| atelier 深色主色 | vermilion `#CB2028` | 同 | 保持 |
| atelier 图表分类 | 钢蓝首色 | dark-first：`#CB2028` + 原浅色点缀 | 深色用作者数组；浅色五色对比 1.53–2.50，**预期整套 light 安全序列** |
| atelier 连续色 | 蓝斜坡 | `Math.round` 的 M(surface,#CB2028,t) 五档 `#191C23,#461D24,#721E26,#9F1F27,#CB2028` | 与公式逐字节一致 |
| lumen 深色主色 | orange `#FB923C` | `#E08A4A` | 与原 categorical[0] 对齐，略闷，对比更稳 |
| lumen 浅色主色 | orange `#FB923C` | `#C2410C` | 奶油底上要够 4.5 |
| hover/muted | `generatePaletteFromHex` | 同 C.2（0.2 向黑 / alpha 0.15） | 公式对齐现网自定义主色 |

工坊示例内容版本 `1.2.0`。v1 省略 elevated 时用「向 text/白混合」而不是等于 surface，避免短包把卡片压平。官方包仍显式写 elevated/sidebar/inset，不以 v1 替代手调层次。

## C — 确定性规则与复核用例

### C.1 输入、回退及资源边界

- 输入必须是 JSON 对象，数组不能充当对象。文件 UTF-8 不超过 256 KiB，嵌套不超过 16 层；超限拒绝整包。未知声明也计入限额。对象键 `__proto__`、`prototype`、`constructor` 剥除并报告，不参与合并。
- 必选字段缺失/类型错误拒绝整包。必选颜色字符串格式非法时可用下方安全值回退并警告。可选字段类型错误（包括 null、字符串冒充数字）按缺失；不强制类型转换。有限数字越界夹紧；整数项先四舍五入再夹紧，变更时警告。schemaMinor 非法或负数回退 0。空 name 拒绝，超长 name 按 Unicode 码点截为 64。非法 version 字符串原样保留并警告，不用于 schema 判断。
- 所有未识别枚举运行时按缺失处理，原值隔离保存；用户编辑同一路径时新值替代隔离值。无效 slug 拒绝，合法但未安装的主题 ID 保留并仅运行回退。slug 必须完整匹配 `^[a-z0-9][a-z0-9-]{1,63}$`。
- 普通色只接受 #RRGGBB；仅 border/borderLight/borderHover 额外允许 `rgba(r,g,b,a)`（r/g/b 为 0–255 整数，a 为 0–1 数字）。背景全部要求不透明 #RRGGBB，以区域 opacity 唯一控制透明；A.2 中背景的 hex/rgba 范围统一收紧为此规则。不接受任意 CSS、var()、url() 或表达式。
- 安全回退按 dark/light：bg=#111318/#E4E6EC，surface=#191C23/#F4F5F8，text=#EEF0F4/#141820，textMuted=#7B8494/#5B6472，border=#303641/#C3C9D4，accent=#CB2028/#5B8CFF；success=#22C55E/#15803D，warning=#F59E0B/#B45309，danger=#EF4444/#B91C1C，info=#3B82F6/#1D4ED8。可选非法颜色按缺失派生。
- easing 缺省为 `cubic-bezier(0.2, 0, 0, 1)`；仅接受 linear/ease/ease-in/ease-out/ease-in-out 或四个有限数值的 cubic-bezier，x1/x2 必须在 0–1，其他字符串按缺失。
- 字体 URL 是字体二进制，绝不把包内 URL 作为 stylesheet 加载。URL 最长 2048 字符，必须 HTTPS 且无用户名/密码；图片同规则。上传资源 ID 只能引用本浏览器字体存储，缺失回退 system。字体单文件上限 8 MiB，总存储 32 MiB，加载超时 10 秒；格式头与 FontFace 加载均须有效。远程字体遵循同一大小限制。fonts.uiUrl/monoUrl 加载成功后用于相应字体槽，失败回退所选内置字体栈；用户显式 uiFont/termFont 优先于主题 URL。内部族名由资源标识生成，不能使用作者字符串拼 CSS。
- A.2 的登记终端预设来自 `frontend/src/utils/terminal-theme.ts`，字体 key 来自同模块及 shared-tokens 的对应栈；实现时把现有注册数据作为 v1 固定数据迁入解析器依赖，不得随主题 ID 分支。修改注册色值或共享尺寸表须记录视觉版本，不能暗改冻结后的 v1。defaults.termFont 非法回退 jetbrains。

### C.2 v1 派生、状态与覆盖

以下数值为待复核的 v1 基线，不是解析器自行挑选的示意值。M(a,b,t) 表示在编码 sRGB 的 RGB 三通道逐通道计算 `Math.round(a×(1−t)+b×t)`（ECMAScript：正数一半向 +∞，`69.5→70`、`158.5→159`），输出 #RRGGBB；不做 HSL 或线性光混色。

| 结果 | v1 规则 |
| --- | --- |
| elevated / overlay | dark: elevated=M(surface,text,0.06)；light: elevated=M(surface,#FFFFFF,0.40)。overlay=elevated。这是浮层/抬起面的缺省层次，**不是** card 面：card 消费 surface，省略 elevated 不会把卡片面「塌成没有表面」。省略时变弱的是 float/elevated 相对 card 的抬起 |
| inset | dark: M(surface,#000000,0.12)；light: M(surface,#000000,0.04) |
| sidebar / header / auth | dark: M(bg,#000000,0.12)；light: M(bg,#000000,0.04)。header 默认同 sidebar |
| input / tableHeader | =inset |
| textSecondary | M(text,textMuted,0.5) |
| borderLight / borderHover | borderLight=M(border,surface,0.5)；borderHover=M(border,text,0.2)。显式 border 带 alpha 时先合成到 surface 再派生 |
| hover / muted | hover=M(accent,#000000,0.2)（与现 `generatePaletteFromHex` 一致）；muted=accent alpha 0.15 |
| 副色 | M(有效 accent,#8B5CF6,0.6)（与现自定义主色公式一致，有意偏色相以便双指标可分）。朱红等品牌副色必须在 palette 里显式写出，不能指望 v1 得到 #E85A62 |
| onAccent | #000000 与 #FFFFFF 中对 accent 对比度较高者，平局选黑；hover/active 按各自底色算各自前景 |
| fill-1 / fill-2 | fill-1=inset；fill-2=M(inset,text,0.08) |
| inset hover | 在 default 或 selected 底上叠 text，alpha=0.08×hoverStrength；0 无叠层 |
| selected / disabled | selected=M(fill-1,accent,0.15)；disabled 抑制 hover；disabled+selected 保留 1px borderHover 描边，背景回到 fill-1；不降低正文或焦点环 opacity |
| card 顶边 | 开启时 1px accent、alpha 0.16，独立于其他边框；所有 card 消费者相同 |
| card/float/page 的状态 | 非交互外壳四态背景相同；内部交互项用 inset 状态 |
| wash | 首期四种配方 wash=0；纹理走 chromeTexture。未来 wash 字段缺省 0 |

覆盖顺序逐字段执行：用户高级副色 > 有效用户主色覆盖触发的副色派生 > palette 副色 > v1 派生。未设置用户主色时不得无故改写作者副色。显式副色只影响自身。accentKey=custom 但 accentCustom 缺失/非法时不视为有效主色覆盖；只有 accentCustom、没有 custom key 时也不激活覆盖。

配方参数均为用户值 > 主题值 > v1。card 背景取 surface，float 取 overlay，page 取 bg；drawer 外壳用 card。background 与 wash 的区域 alpha 只合成一次，边框、状态层、文字、焦点环不跟随区域 alpha。sidebar/header blur 只在透明启用、环境支持且未减少动效时有效。

对比度函数使用 sRGB 相对亮度：通道 s=c/255，s≤0.04045 时 s/12.92，否则 ((s+0.055)/1.055)^2.4；L=0.2126R+0.7152G+0.0722B；比值=(Lmax+0.05)/(Lmin+0.05)。浅色 accent 作为文字先对 surface 做 4.5 校正；text/textSecondary/textMuted 在两种模式下均对 bg/surface/inset/overlay 校验 4.5。校正按 M(原色,黑或白,i/100)，i=0…100，取第一个满足所有目标背景的值；两方向均可时取较小 i，平局黑。无公共解则保留原色并发对比度警告，正式主题验收不通过。onAccent、hover、muted、派生副色等在 accent 校正之后计算。不得因校正修改导出原始声明；透明图片的实际画面另做视觉验收，不以固底校验宣称达标。

图表：显式数组为作者缺省。用户主色覆盖时分类首色替换为有效 accent（其余继承作者数组），连续色按缺数组公式重算；无覆盖时保留显式数组。缺分类数组取 [accent,success,warning,danger,有效副色,info]；缺连续数组取 t=[0,0.25,0.5,0.75,1] 的 M(surface,accent,t)。分类校验为任意两色 RGB 欧氏距离≥48，且各色对**当前模式 surface** 对比度≥3。不满足则**整套**回退该模式安全序列并警告：dark=[#7AA2FF,#34D399,#FBBF24,#FB7185,#A78BFA,#22D3EE]，light=[#1D4ED8,#15803D,#92400E,#B91C1C,#7E22CE,#0E7490]。不做逐色替换（避免朱红首色留下后与安全 danger 距离 <48）。安全序列也做同一检查，失败标记验收失败，不无限重试。距离 48 只是自动筛查，**不等同色觉无障碍**。连续色要求相对亮度严格单调，否则回退 dark=[#303641,#EEF0F4]、light=[#F4F5F8,#141820] 两端渐变。副色不控制图表派生。

图表数组不分模式。工坊示例分类是 **dark-first**：首色 `#CB2028` 在浅色 surface 上对比约 5.1，但其余五色约 1.53–2.50，**浅色模式预期整套走 light 安全序列**。这不是实现错误。若以后要浅色品牌色，再加可选分模式图表字段，不改现有数组含义。

终端：显式预设整套优先。未指定时，背景=bg、前景=text、光标=text、选区为 accent alpha 0.25。ANSI 普通八色初始为 [bg, danger, success, warning, accent, 有效副色, info, text]。black 初始对比度为 1:1，**校正后不会保持等于 bg**。亮八色：dark 为 M(普通,#FFFFFF,0.2)；light 为 M(普通,#000000,0.2)（较好的浅色起点，不是因为「向白混必然无解」）。每个 ANSI 色对终端固底校正至 3:1；对单个不透明底，黑白至少一端可达 3:1。仅当某色在两端校正后仍无解才整套回退 default/paper 并警告。编辑器语法缺省 vs-dark/vs；工作区底与透明度跟随终端。

### C.3 表面家族、恢复分组与验收

表面家族 dark 使用 `frontend/src/theme/surfaces.ts` 当前六套颜色作为 v1 固定输入。light 以 C.1 的 bg/surface 为基底，graphite/abyss/void/tinted/cosmos/warm 的染色色分别为 #64748B/#1D4ED8/#000000/#0F766E/#7E22CE/#92400E；bg=M(浅色 bg,染色,0.06)，surface=M(浅色 surface,染色,0.03)，其余背景按 C.2 从新 bg/surface 派生。家族仅替换背景，不覆盖作者文字/状态/品牌声明；之后重做对比度校验。禁止按 themeId 限制使用。

前缀匹配按 JSON 路径段，不按字符串 startsWith。以下字段均相对 `preference.overridesByTheme.<id>`；没有指定未知前缀时未知项仅随全部恢复清除。恢复当前主题全部时清除该 id 的全部已知覆盖及隔离覆盖；全局全部恢复另清全部偏好及全部隔离声明，不删除安装的主题包或资源文件。

| 组 | 删除的已知字段 | 隔离项路径前缀 |
| --- | --- | --- |
| harmony.accent | accentKey、accentCustom | 无 |
| advanced.secondary | accentSecondary | 无 |
| harmony.surface | surfacePreset | 无 |
| advanced.recipe | surfaces.card.topEdge、surfaces.inset.hoverStrength | surfaces |
| harmony.density | density | 无 |
| harmony.radius | radius | 无 |
| advanced.layout | sidebarWidth、headerHeight | 无 |
| harmony.fonts | uiFont、termFont、termFontSize | 无 |
| harmony.materials | transparency、chromeTexture、chromeImageMode、chromeImageUrl | 无 |
| advanced.variants | card、sidebarVariant、subnav、iconContainer、iconSet | 无 |
| advanced.terminal | termTheme、termBgOpacity、termWallpaper、termFollowChrome、termImageMode、termImageUrl | 无 |

首期无 editor overlay，不提供空的恢复组；以后新增必须补 A.3 和映射。旧恢复组 API 若沿用 common/accent/variants/material/terminal，按旧字段集合处理，不能将旧组名直接当新路径前缀。未知枚举隔离在已知字段原路径上，删除该已知字段时连同隔离值删除。

解析器验收输入与预期（本轮只是契约，不表示测试已执行）：

| 输入 | 预期 |
| --- | --- |
| 任一示例 + 空 overlay | 继承作者副色；hoverStrength=1；topEdge=true |
| 主题 topEdge=false、hoverStrength=0.4 + 空 overlay | false、0.4，不被全局默认覆盖 |
| 同上 + overlay topEdge=true、hoverStrength=0 | true、0 |
| 删除主题副色且无用户主色 | 按 M(有效 accent,#8B5CF6,0.6) 派生 |
| overlay 主色=custom/#123456、副色=#ABCDEF | 副色仍 #ABCDEF；hover/muted/图表独立重算 |
| overlay 仅副色=#ABCDEF | 作者主色不变；图表不因副色而停止处理 |
| 未知 surfaces.inset.future=0.2，再恢复 advanced.recipe | 该隔离项删除，再导出不复活 |
| 未知 futureOption=true，再恢复 advanced.recipe | 保留；当前主题全部恢复才删除 |
| 内外版本 1/1，schemaMinor=99 | 读取已知字段并隔离未知声明 |
| 内外版本 1/2 | 拒绝，保留原外观 |
| derive.strategy=future、derive.cache={} | 运行 v1；导出 future；cache 剥除并报告 |
| 两示例分别改为任意合法新 id | 除身份/资源索引外，相同输入得到相同外观结果 |
