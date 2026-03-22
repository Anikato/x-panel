<template>
  <div class="xray-container">
    <!-- 未安装提示 -->
    <div v-if="!xrayStatus.installed" class="install-banner">
      <el-alert
        :title="t('xray.notInstalled')"
        type="warning"
        :description="t('xray.notInstalledDesc')"
        show-icon
        :closable="false"
      />
      <div class="install-actions">
        <el-button type="primary" :loading="installing" @click="handleInstall">
          {{ installing ? t('xray.installing') : t('xray.installBtn') }}
        </el-button>
      </div>
      <div v-if="installLog" class="install-log">
        <pre ref="logRef">{{ installLog }}</pre>
      </div>
    </div>

    <!-- 主内容 -->
    <template v-else>
      <!-- 顶部状态栏 -->
      <el-card class="status-bar" shadow="never">
        <div class="status-info">
          <el-tag :type="xrayStatus.running ? 'success' : 'danger'" size="large">
            <el-icon class="mr-1"><CircleCheck v-if="xrayStatus.running" /><CircleClose v-else /></el-icon>
            {{ xrayStatus.running ? t('xray.running') : t('xray.stopped') }}
          </el-tag>
          <span class="version-text">{{ xrayStatus.version }}</span>
          <span class="config-path">{{ xrayStatus.configPath }}</span>
        </div>
        <el-button size="small" @click="loadStatus">{{ t('common.refresh') }}</el-button>
      </el-card>

      <el-row :gutter="16" class="main-layout">
        <!-- 左侧：节点列表 -->
        <el-col :span="7">
          <el-card shadow="never" class="node-card">
            <template #header>
              <div class="card-header">
                <span>{{ t('xray.nodes') }}</span>
                <el-button type="primary" size="small" @click="openNodeDrawer()">
                  <el-icon><Plus /></el-icon>{{ t('xray.addNode') }}
                </el-button>
              </div>
            </template>
            <div v-if="nodes.length === 0" class="empty-text">{{ t('xray.noNodes') }}</div>
            <div
              v-for="node in nodes"
              :key="node.id"
              class="node-item"
              :class="{ active: selectedNodeId === node.id, disabled: !node.enabled }"
              @click="selectNode(node.id)"
            >
              <div class="node-header">
                <el-tag size="small" :type="protocolTagType(node.protocol)">{{ node.protocol.toUpperCase() }}</el-tag>
                <el-switch
                  v-model="node.enabled"
                  size="small"
                  @change="handleToggleNode(node)"
                  @click.stop
                />
              </div>
              <div class="node-name">{{ node.name }}</div>
              <div class="node-meta">
                <span class="meta-item">
                  <el-icon><Monitor /></el-icon>{{ node.listenAddr }}:{{ node.port }}
                </span>
                <span class="meta-item">
                  <el-icon><Connection /></el-icon>{{ node.network.toUpperCase() }}
                </span>
                <el-tag size="small" plain :type="securityTagType(node.security)">{{ node.security }}</el-tag>
              </div>
              <div class="node-meta">
                <span class="meta-item">
                  <el-icon><User /></el-icon>{{ node.userCount }} {{ t('xray.users') }}
                </span>
                <span v-if="node.flow" class="meta-item flow-badge">
                  {{ node.flow }}
                </span>
              </div>
              <div class="node-actions" @click.stop>
                <el-button size="small" text @click="openNodeDrawer(node)">{{ t('common.edit') }}</el-button>
                <el-button size="small" text type="danger" @click="handleDeleteNode(node.id)">{{ t('common.delete') }}</el-button>
              </div>
            </div>
          </el-card>
        </el-col>

        <!-- 右侧：用户列表 -->
        <el-col :span="17">
          <el-card shadow="never">
            <template #header>
              <div class="card-header">
                <span>
                  {{ t('xray.userList') }}
                  <el-tag v-if="selectedNodeId" size="small" class="ml-2">
                    {{ nodes.find(n => n.id === selectedNodeId)?.name }}
                  </el-tag>
                </span>
                <el-button type="primary" size="small" :disabled="!selectedNodeId" @click="openUserDialog()">
                  <el-icon><Plus /></el-icon>{{ t('xray.addUser') }}
                </el-button>
              </div>
            </template>

            <el-table :data="users" v-loading="userLoading" size="default">
              <el-table-column prop="name" :label="t('xray.userName')" min-width="120" />
              <el-table-column prop="uuid" :label="t('xray.uuid')" min-width="280">
                <template #default="{ row }">
                  <div class="uuid-cell">
                    <span class="uuid-text">{{ row.uuid }}</span>
                    <el-button size="small" text @click="copyText(row.uuid)"><el-icon><CopyDocument /></el-icon></el-button>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="flow" :label="t('xray.flow')" width="160">
                <template #default="{ row }">
                  <el-tag v-if="row.flow" size="small" type="warning">{{ row.flow }}</el-tag>
                  <span v-else class="text-muted">{{ t('xray.inheritNodeFlow') }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="t('xray.traffic')" width="180">
                <template #default="{ row }">
                  <div class="traffic-cell" @click="openTrafficChart(row)" style="cursor: pointer">
                    <div class="traffic-up">↑ {{ formatBytes(row.uploadTotal) }}</div>
                    <div class="traffic-down">↓ {{ formatBytes(row.downloadTotal) }}</div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="t('xray.expireAt')" width="130">
                <template #default="{ row }">
                  <span v-if="row.expireAt" :class="isExpired(row.expireAt) ? 'text-danger' : ''">
                    {{ formatDate(row.expireAt) }}
                  </span>
                  <span v-else class="text-muted">{{ t('xray.neverExpire') }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="t('common.status')" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
                    {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="t('common.operations')" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" text @click="openUserDialog(row)">{{ t('common.edit') }}</el-button>
                  <el-button size="small" text @click="getShareLink(row.id)">{{ t('xray.shareLink') }}</el-button>
                  <el-button size="small" text type="danger" @click="handleDeleteUser(row.id)">{{ t('common.delete') }}</el-button>
                </template>
              </el-table-column>
            </el-table>

            <el-pagination
              v-if="userTotal > 0"
              class="pagination"
              v-model:current-page="userPage"
              v-model:page-size="userPageSize"
              :total="userTotal"
              layout="total, prev, pager, next"
              @change="loadUsers"
            />
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- ============================================================ -->
    <!-- 节点编辑 Drawer                                               -->
    <!-- ============================================================ -->
    <el-drawer
      v-model="nodeDrawerVisible"
      :title="nodeForm.id ? t('xray.editNode') : t('xray.addNode')"
      size="640px"
      destroy-on-close
    >
      <el-form
        ref="nodeFormRef"
        :model="nodeForm"
        :rules="nodeRules"
        label-width="140px"
        label-position="right"
        size="default"
      >
        <el-tabs v-model="nodeFormTab" class="node-tabs">
          <!-- ===== Tab 1: 基础设置 ===== -->
          <el-tab-pane :label="t('xray.tabBasic')" name="basic">
            <el-form-item :label="t('xray.nodeName')" prop="name">
              <el-input v-model="nodeForm.name" :placeholder="t('xray.nodeNamePlaceholder')" />
            </el-form-item>
            <el-form-item :label="t('xray.protocol')" prop="protocol">
              <el-radio-group v-model="nodeForm.protocol">
                <el-radio-button value="vless">VLESS</el-radio-button>
                <el-radio-button value="vmess">VMess</el-radio-button>
                <el-radio-button value="trojan">Trojan</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item :label="t('xray.listenAddr')" prop="listenAddr">
              <el-select v-model="nodeForm.listenAddr" style="width: 100%">
                <el-option value="0.0.0.0" label="0.0.0.0（所有网卡）" />
                <el-option value="127.0.0.1" label="127.0.0.1（仅本机，适合 nginx 反代）" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('xray.port')" prop="port">
              <el-input-number
                v-model="nodeForm.port"
                :min="1"
                :max="65535"
                style="width: 100%"
                :placeholder="t('xray.portPlaceholder')"
              />
            </el-form-item>
            <el-form-item :label="t('xray.remark')">
              <el-input v-model="nodeForm.remark" type="textarea" :rows="2" />
            </el-form-item>
            <el-form-item v-if="nodeForm.id" :label="t('common.enabled')">
              <el-switch v-model="nodeForm.enabled" />
            </el-form-item>
          </el-tab-pane>

          <!-- ===== Tab 2: 传输协议 ===== -->
          <el-tab-pane :label="t('xray.tabTransport')" name="transport">
            <el-form-item :label="t('xray.network')" prop="network">
              <el-radio-group v-model="nodeForm.network">
                <el-radio-button value="raw">TCP (RAW)</el-radio-button>
                <el-radio-button value="ws">WebSocket</el-radio-button>
                <el-radio-button value="grpc">gRPC</el-radio-button>
                <el-radio-button value="xhttp">XHTTP</el-radio-button>
                <el-radio-button value="httpupgrade">HTTPUpgrade</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <!-- RAW/TCP 配置 -->
            <template v-if="nodeForm.network === 'raw'">
              <el-divider content-position="left">TCP (RAW) {{ t('xray.settings') }}</el-divider>
              <el-form-item :label="t('xray.headerType')">
                <el-radio-group v-model="nodeForm.rawSettings.headerType">
                  <el-radio-button value="none">None</el-radio-button>
                  <el-radio-button value="http">HTTP 伪装</el-radio-button>
                </el-radio-group>
              </el-form-item>
              <el-form-item :label="t('xray.acceptProxyProtocol')">
                <el-switch v-model="nodeForm.rawSettings.acceptProxyProtocol" />
                <span class="form-hint">{{ t('xray.acceptProxyProtocolHint') }}</span>
              </el-form-item>
            </template>

            <!-- WebSocket 配置 -->
            <template v-if="nodeForm.network === 'ws'">
              <el-divider content-position="left">WebSocket {{ t('xray.settings') }}</el-divider>
              <el-form-item :label="t('xray.path')">
                <el-input v-model="nodeForm.wsSettings.path" placeholder="/ws" />
              </el-form-item>
              <el-form-item :label="t('xray.host')">
                <el-input v-model="nodeForm.wsSettings.host" :placeholder="t('xray.hostPlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('xray.acceptProxyProtocol')">
                <el-switch v-model="nodeForm.wsSettings.acceptProxyProtocol" />
                <span class="form-hint">{{ t('xray.acceptProxyProtocolHint') }}</span>
              </el-form-item>
            </template>

            <!-- gRPC 配置 -->
            <template v-if="nodeForm.network === 'grpc'">
              <el-divider content-position="left">gRPC {{ t('xray.settings') }}</el-divider>
              <el-form-item :label="t('xray.grpcServiceName')">
                <el-input v-model="nodeForm.grpcSettings.serviceName" placeholder="grpc" />
              </el-form-item>
              <el-form-item :label="t('xray.grpcMultiMode')">
                <el-switch v-model="nodeForm.grpcSettings.multiMode" />
                <span class="form-hint">{{ t('xray.grpcMultiModeHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.grpcIdleTimeout')">
                <el-input-number v-model="nodeForm.grpcSettings.idleTimeout" :min="0" style="width:100%" />
                <span class="form-hint">{{ t('xray.seconds') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.grpcHealthTimeout')">
                <el-input-number v-model="nodeForm.grpcSettings.healthCheckTimeout" :min="0" style="width:100%" />
                <span class="form-hint">{{ t('xray.seconds') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.grpcPermitWithoutStream')">
                <el-switch v-model="nodeForm.grpcSettings.permitWithoutStream" />
              </el-form-item>
            </template>

            <!-- XHTTP 配置 -->
            <template v-if="nodeForm.network === 'xhttp'">
              <el-divider content-position="left">XHTTP (SplitHTTP) {{ t('xray.settings') }}</el-divider>
              <el-form-item :label="t('xray.path')">
                <el-input v-model="nodeForm.xhttpSettings.path" placeholder="/xhttp" />
              </el-form-item>
              <el-form-item :label="t('xray.host')">
                <el-input v-model="nodeForm.xhttpSettings.host" :placeholder="t('xray.hostPlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('xray.xhttpMode')">
                <el-select v-model="nodeForm.xhttpSettings.mode" style="width:100%">
                  <el-option value="auto" label="auto（自动）" />
                  <el-option value="packet-up" label="packet-up（上行分包）" />
                  <el-option value="stream-up" label="stream-up（上行流式）" />
                  <el-option value="stream-one" label="stream-one（单连接）" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('xray.xhttpPadding')">
                <el-input v-model="nodeForm.xhttpSettings.xPaddingBytes" placeholder="100-1000" />
                <span class="form-hint">{{ t('xray.xhttpPaddingHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.xhttpStreamUpSecs')">
                <el-input v-model="nodeForm.xhttpSettings.scStreamUpServerSecs" placeholder="20-80" />
                <span class="form-hint">{{ t('xray.xhttpStreamUpSecsHint') }}</span>
              </el-form-item>
            </template>

            <!-- HTTPUpgrade 配置 -->
            <template v-if="nodeForm.network === 'httpupgrade'">
              <el-divider content-position="left">HTTPUpgrade {{ t('xray.settings') }}</el-divider>
              <el-form-item :label="t('xray.path')">
                <el-input v-model="nodeForm.httpUpgradeSettings.path" placeholder="/" />
              </el-form-item>
              <el-form-item :label="t('xray.host')">
                <el-input v-model="nodeForm.httpUpgradeSettings.host" :placeholder="t('xray.hostPlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('xray.acceptProxyProtocol')">
                <el-switch v-model="nodeForm.httpUpgradeSettings.acceptProxyProtocol" />
              </el-form-item>
            </template>
          </el-tab-pane>

          <!-- ===== Tab 3: 安全加密 ===== -->
          <el-tab-pane :label="t('xray.tabSecurity')" name="security">
            <el-form-item :label="t('xray.security')" prop="security">
              <el-radio-group v-model="nodeForm.security">
                <el-radio-button value="none">{{ t('xray.secNone') }}</el-radio-button>
                <el-radio-button value="tls">TLS</el-radio-button>
                <el-radio-button value="reality">Reality</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <!-- TLS 配置 -->
            <template v-if="nodeForm.security === 'tls'">
              <el-divider content-position="left">TLS {{ t('xray.settings') }}</el-divider>
              <el-form-item :label="t('xray.tlsServerName')">
                <el-input v-model="nodeForm.tlsSettings.serverName" placeholder="example.com" />
                <span class="form-hint">{{ t('xray.tlsServerNameHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.tlsCertFile')">
                <el-input v-model="nodeForm.tlsSettings.certFile" placeholder="/etc/ssl/cert.pem" />
              </el-form-item>
              <el-form-item :label="t('xray.tlsKeyFile')">
                <el-input v-model="nodeForm.tlsSettings.keyFile" placeholder="/etc/ssl/key.pem" />
              </el-form-item>
              <el-form-item :label="t('xray.tlsALPN')">
                <el-checkbox-group v-model="nodeForm.tlsSettings.alpn">
                  <el-checkbox value="h2">h2</el-checkbox>
                  <el-checkbox value="http/1.1">http/1.1</el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              <el-form-item :label="t('xray.tlsFingerprint')">
                <el-select v-model="nodeForm.tlsSettings.fingerprint" clearable :placeholder="t('xray.tlsFingerprintPlaceholder')" style="width:100%">
                  <el-option value="" label="默认（Go TLS）" />
                  <el-option value="chrome" label="Chrome" />
                  <el-option value="firefox" label="Firefox" />
                  <el-option value="safari" label="Safari" />
                  <el-option value="ios" label="iOS" />
                  <el-option value="android" label="Android" />
                  <el-option value="edge" label="Edge" />
                  <el-option value="360" label="360" />
                  <el-option value="qq" label="QQ" />
                  <el-option value="random" label="random（随机浏览器）" />
                  <el-option value="randomized" label="randomized（完全随机）" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('xray.tlsMinVersion')">
                <el-select v-model="nodeForm.tlsSettings.minVersion" style="width:100%">
                  <el-option value="1.0" label="TLS 1.0" />
                  <el-option value="1.1" label="TLS 1.1" />
                  <el-option value="1.2" label="TLS 1.2（推荐）" />
                  <el-option value="1.3" label="TLS 1.3" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('xray.tlsRejectUnknownSni')">
                <el-switch v-model="nodeForm.tlsSettings.rejectUnknownSni" />
                <span class="form-hint">{{ t('xray.tlsRejectUnknownSniHint') }}</span>
              </el-form-item>
            </template>

            <!-- Reality 配置 -->
            <template v-if="nodeForm.security === 'reality'">
              <el-divider content-position="left">Reality {{ t('xray.settings') }}</el-divider>
              <el-alert
                type="info"
                :closable="false"
                style="margin-bottom: 16px"
                :description="t('xray.realityTip')"
              />
              <el-form-item :label="t('xray.realityPrivKey')">
                <el-input v-model="nodeForm.realitySettings.privateKey" placeholder="私钥" />
              </el-form-item>
              <el-form-item :label="t('xray.realityPubKey')">
                <div class="key-row">
                  <el-input v-model="nodeForm.realitySettings.publicKey" placeholder="公钥（提供给客户端）" readonly />
                  <el-button @click="generateRealityKeyPair" :loading="generatingKeys">{{ t('xray.generateKeys') }}</el-button>
                </div>
              </el-form-item>
              <el-form-item :label="t('xray.realityDest')">
                <el-input v-model="nodeForm.realitySettings.dest" placeholder="www.apple.com:443" />
                <span class="form-hint">{{ t('xray.realityDestHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.realityServerNames')">
                <div class="tag-input-area">
                  <el-tag
                    v-for="(sn, i) in nodeForm.realitySettings.serverNames"
                    :key="i"
                    closable
                    @close="nodeForm.realitySettings.serverNames.splice(i, 1)"
                  >{{ sn }}</el-tag>
                  <el-input
                    v-if="realitySnInputVisible"
                    ref="realitySnInputRef"
                    v-model="realitySnInput"
                    size="small"
                    style="width: 160px"
                    @keyup.enter="addRealitySn"
                    @blur="addRealitySn"
                  />
                  <el-button v-else size="small" @click="showRealitySnInput">+ {{ t('xray.addServerName') }}</el-button>
                </div>
                <span class="form-hint">{{ t('xray.realityServerNamesHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.realityShortIds')">
                <div class="tag-input-area">
                  <el-tag
                    v-for="(sid, i) in nodeForm.realitySettings.shortIds"
                    :key="i"
                    closable
                    @close="nodeForm.realitySettings.shortIds.splice(i, 1)"
                  >{{ sid }}</el-tag>
                  <el-input
                    v-if="realitySidInputVisible"
                    ref="realitySidInputRef"
                    v-model="realitySidInput"
                    size="small"
                    style="width: 160px"
                    @keyup.enter="addRealitySid"
                    @blur="addRealitySid"
                  />
                  <el-button v-else size="small" @click="showRealitySidInput">+ {{ t('xray.addShortId') }}</el-button>
                </div>
                <span class="form-hint">{{ t('xray.realityShortIdsHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.realityFingerprint')">
                <el-select v-model="nodeForm.realitySettings.fingerprint" style="width:100%">
                  <el-option value="chrome" label="Chrome（推荐）" />
                  <el-option value="firefox" label="Firefox" />
                  <el-option value="safari" label="Safari" />
                  <el-option value="ios" label="iOS" />
                  <el-option value="android" label="Android" />
                  <el-option value="edge" label="Edge" />
                  <el-option value="360" label="360" />
                  <el-option value="qq" label="QQ" />
                  <el-option value="random" label="random" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('xray.realityXver')">
                <el-select v-model="nodeForm.realitySettings.xver" style="width:100%">
                  <el-option :value="0" label="0（不启用）" />
                  <el-option :value="1" label="1（Proxy Protocol v1）" />
                  <el-option :value="2" label="2（Proxy Protocol v2）" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('xray.realitySpiderX')">
                <el-input v-model="nodeForm.realitySettings.spiderX" placeholder="/" />
              </el-form-item>
            </template>
          </el-tab-pane>

          <!-- ===== Tab 4: 高级设置 ===== -->
          <el-tab-pane :label="t('xray.tabAdvanced')" name="advanced">
            <!-- VLESS Flow（仅 VLESS + TCP + TLS/Reality 时推荐） -->
            <el-form-item v-if="nodeForm.protocol === 'vless'" :label="t('xray.flow')">
              <el-radio-group v-model="nodeForm.flow">
                <el-radio value="">{{ t('xray.flowNone') }}（普通 TLS/无加密）</el-radio>
                <el-radio value="xtls-rprx-vision">xtls-rprx-vision（XTLS Vision，需 TCP+TLS/Reality）</el-radio>
              </el-radio-group>
              <div class="form-hint" v-if="nodeForm.flow === 'xtls-rprx-vision' && (nodeForm.network !== 'raw' || nodeForm.security === 'none')">
                <el-icon color="#E6A23C"><Warning /></el-icon>
                {{ t('xray.flowWarning') }}
              </div>
            </el-form-item>

            <!-- 流量探测 (Sniffing) -->
            <el-divider content-position="left">{{ t('xray.sniffing') }}</el-divider>
            <el-form-item :label="t('xray.sniffEnabled')">
              <el-switch v-model="nodeForm.sniffEnabled" />
            </el-form-item>
            <el-form-item v-if="nodeForm.sniffEnabled" :label="t('xray.sniffDestOverride')">
              <el-checkbox-group v-model="nodeForm.sniffDestOverride">
                <el-checkbox value="http">HTTP</el-checkbox>
                <el-checkbox value="tls">TLS</el-checkbox>
                <el-checkbox value="quic">QUIC</el-checkbox>
                <el-checkbox value="fakedns">FakeDNS</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="nodeDrawerVisible = false">{{ t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="nodeSubmitting" @click="submitNodeForm">{{ t('common.confirm') }}</el-button>
        </div>
      </template>
    </el-drawer>

    <!-- ============================================================ -->
    <!-- 用户编辑 Dialog                                               -->
    <!-- ============================================================ -->
    <el-dialog
      v-model="userDialogVisible"
      :title="userForm.id ? t('xray.editUser') : t('xray.addUser')"
      width="560px"
      destroy-on-close
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userRules"
        label-width="120px"
      >
        <el-form-item :label="t('xray.userName')" prop="name">
          <el-input v-model="userForm.name" :placeholder="t('xray.userNamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('xray.uuid')">
          <div class="key-row">
            <el-input v-model="userForm.uuid" placeholder="留空自动生成" />
            <el-button @click="generateUUID">{{ t('xray.generateUUID') }}</el-button>
          </div>
          <span class="form-hint">{{ t('xray.uuidHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('xray.flow')">
          <el-select v-model="userForm.flow" style="width:100%">
            <el-option value="" :label="t('xray.inheritNodeFlow')" />
            <el-option value="xtls-rprx-vision" label="xtls-rprx-vision（XTLS Vision）" />
          </el-select>
          <span class="form-hint">{{ t('xray.userFlowHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('xray.expireAt')">
          <el-date-picker
            v-model="userForm.expireAt"
            type="datetime"
            style="width:100%"
            :placeholder="t('xray.neverExpire')"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            clearable
          />
        </el-form-item>
        <el-form-item :label="t('xray.level')">
          <el-input-number v-model="userForm.level" :min="0" :max="100" style="width:100%" />
          <span class="form-hint">{{ t('xray.levelHint') }}</span>
        </el-form-item>
        <el-form-item v-if="userForm.id" :label="t('common.enabled')">
          <el-switch v-model="userForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('xray.remark')">
          <el-input v-model="userForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="userSubmitting" @click="submitUserForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 分享链接 Dialog -->
    <el-dialog v-model="shareLinkVisible" :title="t('xray.shareLink')" width="580px">
      <el-input v-model="shareLink" readonly type="textarea" :rows="4" />
      <div class="dialog-link-actions">
        <el-button type="primary" @click="copyText(shareLink)">{{ t('common.copy') }}</el-button>
      </div>
    </el-dialog>

    <!-- 流量历史图表 Dialog -->
    <el-dialog v-model="trafficDialogVisible" :title="t('xray.trafficHistory')" width="680px" destroy-on-close>
      <div ref="chartRef" style="height: 320px" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, CircleCheck, CircleClose, Monitor, Connection, User, CopyDocument, Warning
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { v4 as uuidv4 } from 'uuid'
import type { FormInstance } from 'element-plus'
import {
  getXrayStatus, startXrayInstall, getXrayInstallLog,
  listXrayNodes, createXrayNode, updateXrayNode, deleteXrayNode, toggleXrayNode,
  searchXrayUsers, createXrayUser, updateXrayUser, deleteXrayUser,
  generateRealityKeys, getXrayShareLink, getXrayTrafficHistory
} from '@/api/modules/xray'
import type { XrayNode, XrayUser, XrayStatus } from '@/api/modules/xray'

const { t } = useI18n()

// ============================================================
// 状态 & 安装
// ============================================================

const xrayStatus = ref<XrayStatus>({ installed: false, running: false, version: '', configPath: '', binPath: '' })
const installing = ref(false)
const installLog = ref('')
const logRef = ref<HTMLElement>()
let installPollTimer: ReturnType<typeof setInterval> | null = null

const loadStatus = async () => {
  const res = await getXrayStatus()
  xrayStatus.value = res.data
  if (xrayStatus.value.installed) {
    loadNodes()
  }
}

const handleInstall = async () => {
  installing.value = true
  try {
    await startXrayInstall()
    installPollTimer = setInterval(async () => {
      const res = await getXrayInstallLog()
      installLog.value = res.data.log
      nextTick(() => {
        if (logRef.value) logRef.value.scrollTop = logRef.value.scrollHeight
      })
      if (!res.data.running) {
        clearInterval(installPollTimer!)
        installing.value = false
        await loadStatus()
      }
    }, 1500)
  } catch {
    installing.value = false
  }
}

// ============================================================
// 节点
// ============================================================

const nodes = ref<XrayNode[]>([])
const selectedNodeId = ref<number | null>(null)

const loadNodes = async () => {
  const res = await listXrayNodes()
  nodes.value = res.data ?? []
  if (nodes.value.length > 0 && !selectedNodeId.value) {
    selectNode(nodes.value[0].id)
  }
}

const selectNode = (id: number) => {
  selectedNodeId.value = id
  userPage.value = 1
  loadUsers()
}

const handleToggleNode = async (node: XrayNode) => {
  try {
    await toggleXrayNode(node.id)
    ElMessage.success(t('common.operationSuccess'))
  } catch {
    node.enabled = !node.enabled
  }
}

const handleDeleteNode = async (id: number) => {
  await ElMessageBox.confirm(t('xray.deleteNodeConfirm'), t('common.warning'), { type: 'warning' })
  await deleteXrayNode(id)
  ElMessage.success(t('common.deleteSuccess'))
  selectedNodeId.value = null
  await loadNodes()
}

// ---- 节点表单 Drawer ----

const nodeDrawerVisible = ref(false)
const nodeFormTab = ref('basic')
const nodeSubmitting = ref(false)
const nodeFormRef = ref<FormInstance>()
const generatingKeys = ref(false)

// Reality tag inputs
const realitySnInputVisible = ref(false)
const realitySnInput = ref('')
const realitySnInputRef = ref()
const realitySidInputVisible = ref(false)
const realitySidInput = ref('')
const realitySidInputRef = ref()

const emptyNodeForm = () => ({
  id: undefined as number | undefined,
  name: '',
  protocol: 'vless',
  listenAddr: '0.0.0.0',
  port: null as number | null,
  network: 'raw',
  security: 'none',
  flow: '',
  sniffEnabled: true,
  sniffDestOverride: ['http', 'tls'],
  rawSettings: { headerType: 'none', acceptProxyProtocol: false },
  wsSettings: { path: '/ws', host: '', acceptProxyProtocol: false },
  grpcSettings: { serviceName: 'grpc', multiMode: false, idleTimeout: 60, healthCheckTimeout: 20, permitWithoutStream: false, initialWindowsSize: 0 },
  xhttpSettings: { host: '', path: '/xhttp', mode: 'auto', noSSEHeader: false, xPaddingBytes: '100-1000', scStreamUpServerSecs: '20-80', scMaxBufferedPosts: 0 },
  httpUpgradeSettings: { path: '/', host: '', acceptProxyProtocol: false },
  tlsSettings: { serverName: '', certFile: '', keyFile: '', alpn: ['h2', 'http/1.1'], fingerprint: 'chrome', minVersion: '1.2', rejectUnknownSni: false },
    realitySettings: { privateKey: '', publicKey: '', shortIds: [] as string[], serverNames: [] as string[], dest: '', fingerprint: 'chrome', spiderX: '/', xver: 0, show: false },
  remark: '',
  enabled: true,
})

const nodeForm = ref(emptyNodeForm())

const nodeRules = {
  name: [{ required: true, message: t('xray.nodeNameRequired'), trigger: 'blur' }],
  protocol: [{ required: true }],
  port: [{ required: true, type: 'number', min: 1, max: 65535, message: t('xray.portRequired'), trigger: 'change' }],
}

const openNodeDrawer = (node?: XrayNode) => {
  nodeForm.value = emptyNodeForm()
  nodeFormTab.value = 'basic'
  if (node) {
    nodeForm.value.id = node.id
    nodeForm.value.name = node.name
    nodeForm.value.protocol = node.protocol
    nodeForm.value.listenAddr = node.listenAddr || '0.0.0.0'
    nodeForm.value.port = node.port
    nodeForm.value.network = node.network
    nodeForm.value.security = node.security
    nodeForm.value.flow = node.flow || ''
    nodeForm.value.sniffEnabled = node.sniffEnabled
    nodeForm.value.sniffDestOverride = node.sniffDestOverride || ['http', 'tls']
    nodeForm.value.remark = node.remark || ''
    nodeForm.value.enabled = node.enabled
    if (node.rawSettings) nodeForm.value.rawSettings = { ...nodeForm.value.rawSettings, ...node.rawSettings }
    if (node.wsSettings) nodeForm.value.wsSettings = { ...nodeForm.value.wsSettings, ...node.wsSettings }
    if (node.grpcSettings) nodeForm.value.grpcSettings = { ...nodeForm.value.grpcSettings, ...node.grpcSettings }
    if (node.xhttpSettings) nodeForm.value.xhttpSettings = { ...nodeForm.value.xhttpSettings, ...node.xhttpSettings }
    if (node.httpUpgradeSettings) nodeForm.value.httpUpgradeSettings = { ...nodeForm.value.httpUpgradeSettings, ...node.httpUpgradeSettings }
    if (node.tlsSettings) nodeForm.value.tlsSettings = { ...nodeForm.value.tlsSettings, ...node.tlsSettings }
    if (node.realitySettings) nodeForm.value.realitySettings = { ...nodeForm.value.realitySettings, ...node.realitySettings }
  }
  nodeDrawerVisible.value = true
}

const generateRealityKeyPair = async () => {
  generatingKeys.value = true
  try {
    const res = await generateRealityKeys()
    nodeForm.value.realitySettings.privateKey = res.data.privateKey
    nodeForm.value.realitySettings.publicKey = res.data.publicKey
  } finally {
    generatingKeys.value = false
  }
}

const showRealitySnInput = async () => {
  realitySnInputVisible.value = true
  await nextTick()
  realitySnInputRef.value?.focus()
}
const addRealitySn = () => {
  const v = realitySnInput.value.trim()
  if (v && !nodeForm.value.realitySettings.serverNames.includes(v)) {
    nodeForm.value.realitySettings.serverNames.push(v)
  }
  realitySnInput.value = ''
  realitySnInputVisible.value = false
}

const showRealitySidInput = async () => {
  realitySidInputVisible.value = true
  await nextTick()
  realitySidInputRef.value?.focus()
}
const addRealitySid = () => {
  const v = realitySidInput.value.trim()
  if (v && !nodeForm.value.realitySettings.shortIds.includes(v)) {
    nodeForm.value.realitySettings.shortIds.push(v)
  }
  realitySidInput.value = ''
  realitySidInputVisible.value = false
}

const submitNodeForm = async () => {
  if (!nodeFormRef.value) return
  await nodeFormRef.value.validate()
  nodeSubmitting.value = true
  try {
    const payload = buildNodePayload()
    if (nodeForm.value.id) {
      await updateXrayNode({ ...payload, id: nodeForm.value.id })
    } else {
      await createXrayNode(payload)
    }
    ElMessage.success(t('common.saveSuccess'))
    nodeDrawerVisible.value = false
    await loadNodes()
  } finally {
    nodeSubmitting.value = false
  }
}

const buildNodePayload = () => {
  const f = nodeForm.value
  const base = {
    name: f.name,
    protocol: f.protocol,
    listenAddr: f.listenAddr,
    port: f.port,
    network: f.network,
    security: f.security,
    flow: f.flow,
    sniffEnabled: f.sniffEnabled,
    sniffDestOverride: f.sniffDestOverride,
    remark: f.remark,
    enabled: f.enabled,
  }
  // 传输子配置
  const netKey: Record<string, object> = {
    raw: { rawSettings: f.rawSettings },
    ws: { wsSettings: f.wsSettings },
    grpc: { grpcSettings: f.grpcSettings },
    xhttp: { xhttpSettings: f.xhttpSettings },
    httpupgrade: { httpUpgradeSettings: f.httpUpgradeSettings },
  }
  // 安全子配置
  const secKey: Record<string, object> = {
    tls: { tlsSettings: f.tlsSettings },
    reality: { realitySettings: f.realitySettings },
  }
  return {
    ...base,
    ...(netKey[f.network] || {}),
    ...(secKey[f.security] || {}),
  }
}

// ============================================================
// 用户
// ============================================================

const users = ref<XrayUser[]>([])
const userLoading = ref(false)
const userTotal = ref(0)
const userPage = ref(1)
const userPageSize = ref(20)

const loadUsers = async () => {
  if (!selectedNodeId.value) {
    users.value = []
    return
  }
  userLoading.value = true
  try {
    const res = await searchXrayUsers({ nodeId: selectedNodeId.value, page: userPage.value, pageSize: userPageSize.value })
    users.value = res.data.items ?? []
    userTotal.value = res.data.total
  } finally {
    userLoading.value = false
  }
}

const userDialogVisible = ref(false)
const userSubmitting = ref(false)
const userFormRef = ref<FormInstance>()

const emptyUserForm = () => ({
  id: undefined as number | undefined,
  nodeId: selectedNodeId.value ?? 0,
  name: '',
  uuid: '',
  flow: '',
  level: 0,
  expireAt: null as string | null,
  enabled: true,
  remark: '',
})
const userForm = ref(emptyUserForm())

const userRules = {
  name: [{ required: true, message: t('xray.userNameRequired'), trigger: 'blur' }],
}

const openUserDialog = (user?: XrayUser) => {
  userForm.value = emptyUserForm()
  if (user) {
    userForm.value.id = user.id
    userForm.value.nodeId = user.nodeId
    userForm.value.name = user.name
    userForm.value.uuid = user.uuid
    userForm.value.flow = user.flow || ''
    userForm.value.level = user.level
    userForm.value.expireAt = user.expireAt
    userForm.value.enabled = user.enabled
    userForm.value.remark = user.remark
  }
  userDialogVisible.value = true
}

const generateUUID = () => {
  userForm.value.uuid = uuidv4()
}

const submitUserForm = async () => {
  if (!userFormRef.value) return
  await userFormRef.value.validate()
  userSubmitting.value = true
  try {
    const payload = { ...userForm.value, nodeId: selectedNodeId.value }
    if (userForm.value.id) {
      await updateXrayUser(payload)
    } else {
      await createXrayUser(payload)
    }
    ElMessage.success(t('common.saveSuccess'))
    userDialogVisible.value = false
    await loadUsers()
  } finally {
    userSubmitting.value = false
  }
}

const handleDeleteUser = async (id: number) => {
  await ElMessageBox.confirm(t('xray.deleteUserConfirm'), t('common.warning'), { type: 'warning' })
  await deleteXrayUser(id)
  ElMessage.success(t('common.deleteSuccess'))
  await loadUsers()
}

// ============================================================
// 分享链接
// ============================================================

const shareLinkVisible = ref(false)
const shareLink = ref('')

const getShareLink = async (userId: number) => {
  const res = await getXrayShareLink(userId)
  shareLink.value = res.data.link
  shareLinkVisible.value = true
}

// ============================================================
// 流量历史图表
// ============================================================

const trafficDialogVisible = ref(false)
const chartRef = ref<HTMLElement>()
let chartInstance: echarts.ECharts | null = null

const openTrafficChart = async (user: XrayUser) => {
  trafficDialogVisible.value = true
  const res = await getXrayTrafficHistory(user.id)
  const data = res.data ?? []
  await nextTick()
  if (!chartRef.value) return
  chartInstance?.dispose()
  chartInstance = echarts.init(chartRef.value)
  chartInstance.setOption({
    title: { text: `${user.name} - ${t('xray.trafficHistory')}`, left: 'center', textStyle: { fontSize: 13 } },
    tooltip: { trigger: 'axis', formatter: (params: any[]) => {
      return params.map((p: any) => `${p.seriesName}: ${formatBytes(p.value)}`).join('<br/>')
    }},
    legend: { data: [t('xray.upload'), t('xray.download')], bottom: 0 },
    xAxis: { type: 'category', data: data.map(d => d.date), axisLabel: { rotate: 30, fontSize: 11 } },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v) } },
    series: [
      { name: t('xray.upload'), type: 'line', data: data.map(d => d.upload), smooth: true, itemStyle: { color: '#409EFF' } },
      { name: t('xray.download'), type: 'line', data: data.map(d => d.download), smooth: true, itemStyle: { color: '#67C23A' } },
    ],
    grid: { top: 48, left: 72, right: 16, bottom: 48 },
  })
}

watch(trafficDialogVisible, (v) => {
  if (!v) {
    chartInstance?.dispose()
    chartInstance = null
  }
})

// ============================================================
// 工具函数
// ============================================================

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString()
}

const isExpired = (dateStr: string) => {
  return new Date(dateStr) < new Date()
}

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('common.copySuccess'))
  } catch {
    ElMessage.error(t('common.copyFailed'))
  }
}

const protocolTagType = (protocol: string) => {
  const map: Record<string, string> = { vless: 'primary', vmess: 'success', trojan: 'warning', shadowsocks: 'info' }
  return (map[protocol] ?? 'info') as any
}

const securityTagType = (security: string) => {
  const map: Record<string, string> = { none: 'info', tls: 'success', reality: 'warning' }
  return (map[security] ?? 'info') as any
}

// ============================================================
// 生命周期
// ============================================================

onMounted(() => {
  loadStatus()
})
</script>

<style scoped>
.xray-container {
  padding: 16px;
}

.install-banner {
  max-width: 800px;
  margin: 0 auto;
}
.install-actions {
  margin-top: 16px;
}
.install-log {
  margin-top: 16px;
  background: #1e1e1e;
  border-radius: 6px;
  padding: 12px;
  max-height: 400px;
  overflow-y: auto;
}
.install-log pre {
  color: #d4d4d4;
  font-family: monospace;
  font-size: 13px;
  white-space: pre-wrap;
  margin: 0;
}

.status-bar {
  margin-bottom: 16px;
}
.status-bar :deep(.el-card__body) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}
.status-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.version-text {
  font-size: 13px;
  color: #606266;
}
.config-path {
  font-size: 12px;
  color: #909399;
  font-family: monospace;
}

.main-layout {
  min-height: calc(100vh - 200px);
}

.node-card {
  height: 100%;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.node-item {
  padding: 10px;
  border-radius: 6px;
  cursor: pointer;
  margin-bottom: 8px;
  border: 1px solid var(--el-border-color-light);
  transition: all 0.2s;
}
.node-item:hover {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}
.node-item.active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.node-item.disabled {
  opacity: 0.6;
}
.node-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.node-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
  color: var(--el-text-color-primary);
}
.node-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
  flex-wrap: wrap;
}
.meta-item {
  display: flex;
  align-items: center;
  gap: 2px;
}
.flow-badge {
  font-family: monospace;
  color: var(--el-color-warning);
}
.node-actions {
  margin-top: 6px;
  display: flex;
  gap: 4px;
}
.empty-text {
  text-align: center;
  color: var(--el-text-color-placeholder);
  padding: 32px 0;
}

.uuid-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}
.uuid-text {
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 220px;
}

.traffic-cell {
  font-size: 12px;
}
.traffic-up { color: var(--el-color-primary); }
.traffic-down { color: var(--el-color-success); }

.text-muted { color: var(--el-text-color-placeholder); }
.text-danger { color: var(--el-color-danger); }

.pagination {
  margin-top: 12px;
  justify-content: flex-end;
  display: flex;
}

/* Drawer 表单 */
.node-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}
.form-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 8px;
}

.key-row {
  display: flex;
  gap: 8px;
  width: 100%;
}
.key-row .el-input {
  flex: 1;
}

.tag-input-area {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-link-actions {
  margin-top: 12px;
  text-align: right;
}

.mr-1 {
  margin-right: 4px;
}
.ml-2 {
  margin-left: 8px;
}
</style>
