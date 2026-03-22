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
          <el-tag :type="xrayStatus.enabledOnBoot ? 'success' : 'info'" size="small">
            {{ xrayStatus.enabledOnBoot ? t('xray.bootEnabled') : t('xray.bootDisabled') }}
          </el-tag>
          <span v-if="xrayStatus.version" class="version-text">{{ xrayStatus.version }}</span>
          <span class="config-path">{{ xrayStatus.configPath }}</span>
          <el-tooltip :content="t('xray.singleProcessNote')" placement="bottom">
            <el-tag type="info" size="small" style="cursor:help">
              <el-icon><InfoFilled /></el-icon> 单进程多节点
            </el-tag>
          </el-tooltip>
        </div>
        <div class="status-actions">
          <el-button-group>
            <el-button
              size="small" type="success" :loading="serviceControlLoading"
              :disabled="xrayStatus.running"
              @click="handleControlService('start')"
            >{{ t('xray.serviceStart') }}</el-button>
            <el-button
              size="small" type="warning" :loading="serviceControlLoading"
              @click="handleControlService('restart')"
            >{{ t('xray.serviceRestart') }}</el-button>
            <el-button
              size="small" type="danger" :loading="serviceControlLoading"
              :disabled="!xrayStatus.running"
              @click="handleControlService('stop')"
            >{{ t('xray.serviceStop') }}</el-button>
          </el-button-group>
          <el-button
            size="small"
            :type="xrayStatus.enabledOnBoot ? 'primary' : 'info'"
            :loading="serviceControlLoading"
            @click="handleControlService(xrayStatus.enabledOnBoot ? 'disable' : 'enable')"
          >{{ xrayStatus.enabledOnBoot ? t('xray.bootDisableBtn') : t('xray.bootEnableBtn') }}</el-button>
          <el-button size="small" @click="loadStatus">{{ t('commons.refresh') }}</el-button>
        </div>
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
                  <el-icon><Connection /></el-icon>{{ networkLabel(node.network) }}
                </span>
                <el-tag size="small" plain :type="securityTagType(node.security)">{{ node.security.toUpperCase() }}</el-tag>
              </div>
              <div class="node-meta">
                <span class="meta-item">
                  <el-icon><User /></el-icon>{{ node.userCount }} {{ t('xray.users') }}
                </span>
                <span v-if="node.flow" class="flow-badge">
                  Vision
                </span>
              </div>
              <div class="node-actions" @click.stop>
                <el-button size="small" text @click="openNodeDrawer(node)">{{ t('commons.edit') }}</el-button>
                <el-button size="small" text type="primary" @click="openNginxDialog(node)">Nginx</el-button>
                <el-button size="small" text type="danger" @click="handleDeleteNode(node.id)">{{ t('commons.delete') }}</el-button>
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
              <el-table-column :label="t('xray.uuid')" min-width="260">
                <template #default="{ row }">
                  <div class="uuid-cell">
                    <span class="uuid-text" :title="row.uuid">{{ row.uuid }}</span>
                    <el-button size="small" text @click="copyText(row.uuid)"><el-icon><CopyDocument /></el-icon></el-button>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="flow" :label="t('xray.flow')" width="120">
                <template #default="{ row }">
                  <el-tag v-if="row.flow" size="small" type="warning">Vision</el-tag>
                  <span v-else class="text-muted">—</span>
                </template>
              </el-table-column>
              <el-table-column :label="t('xray.traffic')" width="160">
                <template #default="{ row }">
                  <div class="traffic-cell" @click="openTrafficChart(row)" style="cursor:pointer">
                    <div class="traffic-up">↑ {{ formatBytes(row.uploadTotal) }}</div>
                    <div class="traffic-down">↓ {{ formatBytes(row.downloadTotal) }}</div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="t('xray.expireAt')" width="120">
                <template #default="{ row }">
                  <span v-if="row.expireAt" :class="isExpired(row.expireAt) ? 'text-danger' : ''">
                    {{ formatDate(row.expireAt) }}
                  </span>
                  <span v-else class="text-muted">{{ t('xray.neverExpire') }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="t('commons.status')" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
                    {{ row.enabled ? t('commons.enabled') : t('commons.disabled') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="t('commons.operations')" width="160" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" text @click="openUserDialog(row)">{{ t('commons.edit') }}</el-button>
                  <el-button size="small" text @click="getShareLink(row)">{{ t('xray.shareLink') }}</el-button>
                  <el-button size="small" text type="danger" @click="handleDeleteUser(row.id)">{{ t('commons.delete') }}</el-button>
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
      size="660px"
      destroy-on-close
    >
      <el-form
        ref="nodeFormRef"
        :model="nodeForm"
        :rules="nodeRules"
        label-width="150px"
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
                <el-radio-button value="shadowsocks">Shadowsocks</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <!-- Shadowsocks 协议专属设置 -->
            <template v-if="nodeForm.protocol === 'shadowsocks'">
              <el-form-item :label="t('xray.ssMethod')" prop="ssMethod">
                <el-select v-model="nodeForm.ssMethod" style="width:100%">
                  <el-option value="aes-256-gcm" label="AES-256-GCM（推荐）" />
                  <el-option value="aes-128-gcm" label="AES-128-GCM" />
                  <el-option value="chacha20-poly1305" label="ChaCha20-Poly1305（推荐，移动端）" />
                  <el-option value="2022-blake3-aes-256-gcm" label="2022-blake3-aes-256-gcm（SS2022）" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('xray.ssPassword')" prop="ssPassword">
                <el-input v-model="nodeForm.ssPassword" :placeholder="t('xray.ssPasswordPlaceholder')" show-password />
              </el-form-item>
            </template>
            <el-form-item :label="t('xray.listenAddr')" prop="listenAddr">
              <el-select v-model="nodeForm.listenAddr" style="width:100%">
                <el-option value="0.0.0.0" label="0.0.0.0（所有网卡，直连）" />
                <el-option value="127.0.0.1" label="127.0.0.1（仅本机，适合 nginx 反代）" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('xray.port')" prop="port">
              <el-input-number
                v-model="nodeForm.port"
                :min="1" :max="65535"
                style="width:100%"
                :placeholder="t('xray.portPlaceholder')"
              />
            </el-form-item>
            <el-form-item :label="t('xray.remark')">
              <el-input v-model="nodeForm.remark" type="textarea" :rows="2" />
            </el-form-item>
            <el-form-item v-if="nodeForm.id" :label="t('commons.enable')">
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

            <!-- RAW/TCP -->
            <template v-if="nodeForm.network === 'raw'">
              <el-divider content-position="left">TCP (RAW) 设置</el-divider>
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

            <!-- WebSocket -->
            <template v-if="nodeForm.network === 'ws'">
              <el-divider content-position="left">WebSocket 设置</el-divider>
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

            <!-- gRPC -->
            <template v-if="nodeForm.network === 'grpc'">
              <el-divider content-position="left">gRPC 设置</el-divider>
              <el-form-item :label="t('xray.grpcServiceName')">
                <el-input v-model="nodeForm.grpcSettings.serviceName" placeholder="grpc" />
              </el-form-item>
              <el-form-item :label="t('xray.grpcMultiMode')">
                <el-switch v-model="nodeForm.grpcSettings.multiMode" />
                <span class="form-hint">{{ t('xray.grpcMultiModeHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.grpcIdleTimeout')">
                <el-input-number v-model="nodeForm.grpcSettings.idleTimeout" :min="0" style="width:160px" />
                <span class="form-hint"> {{ t('xray.seconds') }}（默认 60）</span>
              </el-form-item>
              <el-form-item :label="t('xray.grpcHealthTimeout')">
                <el-input-number v-model="nodeForm.grpcSettings.healthCheckTimeout" :min="0" style="width:160px" />
                <span class="form-hint"> {{ t('xray.seconds') }}（默认 20）</span>
              </el-form-item>
              <el-form-item :label="t('xray.grpcPermitWithoutStream')">
                <el-switch v-model="nodeForm.grpcSettings.permitWithoutStream" />
              </el-form-item>
            </template>

            <!-- XHTTP -->
            <template v-if="nodeForm.network === 'xhttp'">
              <el-divider content-position="left">XHTTP (SplitHTTP) 设置</el-divider>
              <el-form-item :label="t('xray.path')">
                <el-input v-model="nodeForm.xhttpSettings.path" placeholder="/xhttp" />
              </el-form-item>
              <el-form-item :label="t('xray.host')">
                <el-input v-model="nodeForm.xhttpSettings.host" :placeholder="t('xray.hostPlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('xray.xhttpMode')">
                <el-select v-model="nodeForm.xhttpSettings.mode" style="width:100%">
                  <el-option value="auto" label="auto（自动，推荐）" />
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
              </el-form-item>
            </template>

            <!-- HTTPUpgrade -->
            <template v-if="nodeForm.network === 'httpupgrade'">
              <el-divider content-position="left">HTTPUpgrade 设置</el-divider>
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

            <!-- TLS -->
            <template v-if="nodeForm.security === 'tls'">
              <el-divider content-position="left">TLS 设置</el-divider>
              <el-form-item :label="t('xray.tlsServerName')">
                <el-input v-model="nodeForm.tlsSettings.serverName" placeholder="example.com" />
                <span class="form-hint">{{ t('xray.tlsServerNameHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.tlsCertFile')">
                <el-input v-model="nodeForm.tlsSettings.certFile" placeholder="/etc/ssl/fullchain.pem" />
              </el-form-item>
              <el-form-item :label="t('xray.tlsKeyFile')">
                <el-input v-model="nodeForm.tlsSettings.keyFile" placeholder="/etc/ssl/privkey.pem" />
              </el-form-item>
              <el-form-item :label="t('xray.tlsALPN')">
                <el-checkbox-group v-model="nodeForm.tlsSettings.alpn">
                  <el-checkbox value="h2">h2</el-checkbox>
                  <el-checkbox value="http/1.1">http/1.1</el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              <el-form-item :label="t('xray.tlsFingerprint')">
                <el-select v-model="nodeForm.tlsSettings.fingerprint" clearable style="width:100%">
                  <el-option value="" label="默认（Go 原生 TLS）" />
                  <el-option v-for="fp in fingerprintOptions" :key="fp.value" :value="fp.value" :label="fp.label" />
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

            <!-- Reality -->
            <template v-if="nodeForm.security === 'reality'">
              <el-divider content-position="left">Reality 设置</el-divider>
              <el-alert type="info" :closable="false" style="margin-bottom:16px" :description="t('xray.realityTip')" />
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
                  <el-tag v-for="(sn, i) in nodeForm.realitySettings.serverNames" :key="i" closable @close="nodeForm.realitySettings.serverNames.splice(i,1)">{{ sn }}</el-tag>
                  <el-input v-if="realitySnInputVisible" ref="realitySnInputRef" v-model="realitySnInput" size="small" style="width:180px" @keyup.enter="addRealitySn" @blur="addRealitySn" />
                  <el-button v-else size="small" @click="showRealitySnInput">+ {{ t('xray.addServerName') }}</el-button>
                </div>
                <span class="form-hint">{{ t('xray.realityServerNamesHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.realityShortIds')">
                <div class="tag-input-area">
                  <el-tag v-for="(sid, i) in nodeForm.realitySettings.shortIds" :key="i" closable @close="nodeForm.realitySettings.shortIds.splice(i,1)">{{ sid }}</el-tag>
                  <el-input v-if="realitySidInputVisible" ref="realitySidInputRef" v-model="realitySidInput" size="small" style="width:180px" @keyup.enter="addRealitySid" @blur="addRealitySid" />
                  <el-button v-else size="small" @click="showRealitySidInput">+ {{ t('xray.addShortId') }}</el-button>
                </div>
                <span class="form-hint">{{ t('xray.realityShortIdsHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('xray.realityFingerprint')">
                <el-select v-model="nodeForm.realitySettings.fingerprint" style="width:100%">
                  <el-option v-for="fp in fingerprintOptions" :key="fp.value" :value="fp.value" :label="fp.label" />
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
            <!-- VLESS/Trojan Flow -->
            <el-form-item v-if="nodeForm.protocol === 'vless'" :label="t('xray.flow')">
              <el-radio-group v-model="nodeForm.flow">
                <el-radio value="">{{ t('xray.flowNone') }}</el-radio>
                <el-radio value="xtls-rprx-vision">xtls-rprx-vision（XTLS Vision）</el-radio>
              </el-radio-group>
              <div v-if="nodeForm.flow === 'xtls-rprx-vision' && (nodeForm.network !== 'raw' || nodeForm.security === 'none')" class="flow-warn">
                <el-icon color="#E6A23C"><Warning /></el-icon>
                {{ t('xray.flowWarning') }}
              </div>
            </el-form-item>

            <!-- Fallbacks（VLESS/Trojan TCP 模式才有意义） -->
            <template v-if="['vless','trojan'].includes(nodeForm.protocol) && nodeForm.network === 'raw'">
              <el-divider content-position="left">{{ t('xray.fallbacks') }}</el-divider>
              <el-alert type="info" :closable="false" :description="t('xray.fallbacksHint')" style="margin-bottom:12px" />
              <div v-for="(fb, i) in nodeForm.fallbacks" :key="i" class="fallback-row">
                <el-input v-model="fb.dest" :placeholder="t('xray.fallbackDest')" style="width:180px" />
                <el-input v-model="fb.path" placeholder="/path（可选）" style="width:130px" />
                <el-input v-model="fb.alpn" placeholder="alpn（可选）" style="width:100px" />
                <el-button type="danger" size="small" text @click="nodeForm.fallbacks.splice(i,1)">{{ t('commons.delete') }}</el-button>
              </div>
              <el-button size="small" @click="addFallback" style="margin-top:8px">+ {{ t('xray.addFallback') }}</el-button>
            </template>

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
            <el-form-item v-if="nodeForm.sniffEnabled" :label="t('xray.sniffMetadataOnly')">
              <el-switch v-model="nodeForm.sniffMetadataOnly" />
              <span class="form-hint">{{ t('xray.sniffMetadataOnlyHint') }}</span>
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="nodeDrawerVisible = false">{{ t('commons.cancel') }}</el-button>
          <el-button type="primary" :loading="nodeSubmitting" @click="submitNodeForm">{{ t('commons.confirm') }}</el-button>
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
      <el-form ref="userFormRef" :model="userForm" :rules="userRules" label-width="120px">
        <el-form-item :label="t('xray.userName')" prop="name">
          <el-input v-model="userForm.name" :placeholder="t('xray.userNamePlaceholder')" />
        </el-form-item>
        <!-- Shadowsocks 使用密码而非 UUID -->
        <template v-if="selectedNodeProtocol === 'shadowsocks'">
          <el-form-item :label="t('xray.ssPassword')">
            <el-input v-model="userForm.uuid" :placeholder="t('xray.ssPasswordPlaceholder')" show-password />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item :label="t('xray.uuid')">
            <div class="key-row">
              <el-input v-model="userForm.uuid" placeholder="留空自动生成" />
              <el-button @click="generateUUID">{{ t('xray.generateUUID') }}</el-button>
            </div>
            <span class="form-hint">{{ t('xray.uuidHint') }}</span>
          </el-form-item>
        </template>
        <!-- flow 仅 VLESS 显示 -->
        <el-form-item v-if="selectedNodeProtocol === 'vless'" :label="t('xray.flow')">
          <el-select v-model="userForm.flow" style="width:100%">
            <el-option value="" :label="t('xray.inheritNodeFlow')" />
            <el-option value="xtls-rprx-vision" label="xtls-rprx-vision（XTLS Vision）" />
          </el-select>
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
        </el-form-item>
        <el-form-item v-if="userForm.id" :label="t('commons.enable')">
          <el-switch v-model="userForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('xray.remark')">
          <el-input v-model="userForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="userSubmitting" @click="submitUserForm">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 分享链接 -->
    <el-dialog v-model="shareLinkVisible" :title="t('xray.shareLink')" width="600px" destroy-on-close>
      <el-form label-width="100px" size="default">
        <el-form-item :label="t('xray.shareLinkHost')">
          <el-input
            v-model="shareLinkHost"
            :placeholder="t('xray.shareLinkHostPlaceholder')"
            clearable
          />
        </el-form-item>
        <el-form-item :label="t('xray.shareLinkPort')">
          <el-input-number v-model="shareLinkPort" :min="1" :max="65535" style="width:100%" />
        </el-form-item>

        <!-- 节点 security=none 时（nginx 反代场景），提供客户端加密覆盖 -->
        <template v-if="shareLinkNode?.security === 'none'">
          <el-divider content-position="left">{{ t('xray.clientEncryption') }}</el-divider>
          <el-alert
            type="info" :closable="false"
            :description="t('xray.clientEncryptionHint')"
            style="margin-bottom: 12px"
          />
          <el-form-item :label="t('xray.clientSecurity')">
            <el-radio-group v-model="shareLinkOverrideSec">
              <el-radio value="none">{{ t('xray.securityNone') }}</el-radio>
              <el-radio value="tls">TLS (via nginx)</el-radio>
            </el-radio-group>
          </el-form-item>
          <template v-if="shareLinkOverrideSec === 'tls'">
            <el-form-item :label="t('xray.tlsSni')">
              <el-input v-model="shareLinkOverrideSni" placeholder="example.com" clearable />
            </el-form-item>
            <el-form-item :label="t('xray.alpn')">
              <el-checkbox-group v-model="shareLinkOverrideAlpn">
                <el-checkbox value="h2">h2</el-checkbox>
                <el-checkbox value="http/1.1">http/1.1</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item :label="t('xray.fingerprint')">
              <el-select v-model="shareLinkOverrideFp" clearable style="width:100%" :placeholder="t('xray.fingerprintPlaceholder')">
                <el-option v-for="fp in fingerprintOptions" :key="fp.value" :value="fp.value" :label="fp.label" />
              </el-select>
            </el-form-item>
          </template>
        </template>
      </el-form>
      <div class="share-hint">{{ t('xray.shareLinkHostHint') }}</div>
      <el-divider />
      <el-input :model-value="computedShareLink" readonly type="textarea" :rows="5" class="share-link-text" resize="none" />
      <template #footer>
        <el-button type="primary" :disabled="!computedShareLink" @click="copyText(computedShareLink)">{{ t('commons.copy') }}</el-button>
        <el-button @click="shareLinkVisible = false">{{ t('commons.close') }}</el-button>
      </template>
    </el-dialog>

    <!-- 流量历史图表 -->
    <el-dialog v-model="trafficDialogVisible" :title="t('xray.trafficHistory')" width="700px" destroy-on-close>
      <div ref="chartRef" style="height: 320px" />
    </el-dialog>

    <!-- ============================================================ -->
    <!-- Nginx 反代配置生成 Dialog                                      -->
    <!-- ============================================================ -->
    <el-dialog
      v-model="nginxDialogVisible"
      :title="t('xray.nginxProxyTitle')"
      width="640px"
      destroy-on-close
    >
      <el-alert type="info" :closable="false" :description="t('xray.nginxProxyDesc')" style="margin-bottom:16px" />

      <el-form label-width="120px" size="default">
        <el-form-item :label="t('xray.nginxUpstreamAddr')">
          <el-input v-model="nginxForm.upstreamAddr" placeholder="127.0.0.1" />
          <span class="form-hint">{{ t('xray.nginxUpstreamAddrHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('xray.nginxUpstreamPort')">
          <el-input-number v-model="nginxForm.upstreamPort" :min="1" :max="65535" style="width:100%" />
        </el-form-item>
        <el-form-item v-if="nginxForm.network === 'ws' || nginxForm.network === 'httpupgrade' || nginxForm.network === 'xhttp'" :label="t('xray.path')">
          <el-input v-model="nginxForm.path" placeholder="/ws" />
        </el-form-item>
        <el-form-item v-if="nginxForm.network === 'grpc'" :label="t('xray.grpcServiceName')">
          <el-input v-model="nginxForm.grpcServiceName" placeholder="grpc" />
        </el-form-item>
        <el-form-item :label="t('xray.nginxSendProxyProtocol')">
          <el-switch v-model="nginxForm.sendProxyProtocol" />
          <span class="form-hint">{{ t('xray.nginxSendProxyProtocolHint') }}</span>
        </el-form-item>
      </el-form>

      <el-divider />
      <div class="nginx-code-header">
        <span class="nginx-code-title">{{ t('xray.nginxLocationBlock') }}</span>
        <el-button size="small" type="primary" @click="copyText(generatedNginxConfig)">{{ t('commons.copy') }}</el-button>
      </div>
      <el-input
        :model-value="generatedNginxConfig"
        type="textarea"
        :rows="nginxConfigRows"
        readonly
        class="nginx-code"
        resize="none"
      />

      <template #footer>
        <el-button @click="nginxDialogVisible = false">{{ t('commons.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, CircleCheck, CircleClose, Monitor, Connection, User, CopyDocument, Warning, InfoFilled
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { v4 as uuidv4 } from 'uuid'
import type { FormInstance } from 'element-plus'
import {
  getXrayStatus, startXrayInstall, getXrayInstallLog, controlXrayService,
  listXrayNodes, createXrayNode, updateXrayNode, deleteXrayNode, toggleXrayNode,
  searchXrayUsers, createXrayUser, updateXrayUser, deleteXrayUser,
  generateRealityKeys, getXrayShareLink, getXrayTrafficHistory
} from '@/api/modules/xray'
import type { XrayNode, XrayUser, XrayStatus } from '@/api/modules/xray'

const { t } = useI18n()

// ============================================================
// 常量
// ============================================================

const fingerprintOptions = [
  { value: 'chrome', label: 'Chrome（推荐）' },
  { value: 'firefox', label: 'Firefox' },
  { value: 'safari', label: 'Safari' },
  { value: 'ios', label: 'iOS' },
  { value: 'android', label: 'Android' },
  { value: 'edge', label: 'Edge' },
  { value: '360', label: '360' },
  { value: 'qq', label: 'QQ' },
  { value: 'random', label: 'random（随机浏览器）' },
  { value: 'randomized', label: 'randomized（完全随机）' },
]

// ============================================================
// 状态 & 安装
// ============================================================

const xrayStatus = ref<XrayStatus>({ installed: false, running: false, enabledOnBoot: false, version: '', configPath: '', binPath: '' })
const installing = ref(false)
const installLog = ref('')
const logRef = ref<HTMLElement>()
let installPollTimer: ReturnType<typeof setInterval> | null = null
const serviceControlLoading = ref(false)

const loadStatus = async () => {
  const res = await getXrayStatus()
  xrayStatus.value = res.data
  if (xrayStatus.value.installed) loadNodes()
}

const handleControlService = async (action: 'start' | 'stop' | 'restart' | 'enable' | 'disable') => {
  serviceControlLoading.value = true
  try {
    const res = await controlXrayService(action)
    xrayStatus.value = res.data
    ElMessage.success(t('commons.operationSuccess'))
  } finally {
    serviceControlLoading.value = false
  }
}

const handleInstall = async () => {
  installing.value = true
  try {
    await startXrayInstall()
    installPollTimer = setInterval(async () => {
      const res = await getXrayInstallLog()
      installLog.value = res.data.log
      nextTick(() => { if (logRef.value) logRef.value.scrollTop = logRef.value.scrollHeight })
      if (!res.data.running) {
        clearInterval(installPollTimer!)
        installing.value = false
        await loadStatus()
      }
    }, 1500)
  } catch { installing.value = false }
}

// ============================================================
// 节点
// ============================================================

const nodes = ref<XrayNode[]>([])
const selectedNodeId = ref<number | null>(null)

const selectedNodeProtocol = computed(() => {
  const n = nodes.value.find(n => n.id === selectedNodeId.value)
  return n?.protocol ?? 'vless'
})

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
    ElMessage.success(t('commons.operationSuccess'))
  } catch { node.enabled = !node.enabled }
}

const handleDeleteNode = async (id: number) => {
  await ElMessageBox.confirm(t('xray.deleteNodeConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteXrayNode(id)
  ElMessage.success(t('commons.deleteSuccess'))
  selectedNodeId.value = null
  await loadNodes()
}

// ---- 节点表单 Drawer ----

const nodeDrawerVisible = ref(false)
const nodeFormTab = ref('basic')
const nodeSubmitting = ref(false)
const nodeFormRef = ref<FormInstance>()
const generatingKeys = ref(false)

const realitySnInputVisible = ref(false)
const realitySnInput = ref('')
const realitySnInputRef = ref()
const realitySidInputVisible = ref(false)
const realitySidInput = ref('')
const realitySidInputRef = ref()

interface FallbackItem { dest: string; path: string; alpn: string }

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
  sniffMetadataOnly: false,
  fallbacks: [] as FallbackItem[],
  // Shadowsocks 专属
  ssMethod: 'aes-256-gcm',
  ssPassword: '',
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
  port: [{ required: true, type: 'number' as const, min: 1, max: 65535, message: t('xray.portRequired'), trigger: 'change' }],
}

const openNodeDrawer = (node?: XrayNode) => {
  nodeForm.value = emptyNodeForm()
  nodeFormTab.value = 'basic'
  if (node) {
    Object.assign(nodeForm.value, {
      id: node.id,
      name: node.name,
      protocol: node.protocol,
      listenAddr: node.listenAddr || '0.0.0.0',
      port: node.port,
      network: node.network,
      security: node.security,
      flow: node.flow || '',
      sniffEnabled: node.sniffEnabled,
      sniffDestOverride: node.sniffDestOverride || ['http', 'tls'],
      remark: node.remark || '',
      enabled: node.enabled,
    })
    if (node.rawSettings) Object.assign(nodeForm.value.rawSettings, node.rawSettings)
    if (node.wsSettings) Object.assign(nodeForm.value.wsSettings, node.wsSettings)
    if (node.grpcSettings) Object.assign(nodeForm.value.grpcSettings, node.grpcSettings)
    if (node.xhttpSettings) Object.assign(nodeForm.value.xhttpSettings, node.xhttpSettings)
    if (node.httpUpgradeSettings) Object.assign(nodeForm.value.httpUpgradeSettings, node.httpUpgradeSettings)
    if (node.tlsSettings) Object.assign(nodeForm.value.tlsSettings, node.tlsSettings)
    if (node.realitySettings) Object.assign(nodeForm.value.realitySettings, node.realitySettings)
  }
  nodeDrawerVisible.value = true
}

const generateRealityKeyPair = async () => {
  generatingKeys.value = true
  try {
    const res = await generateRealityKeys()
    nodeForm.value.realitySettings.privateKey = res.data.privateKey
    nodeForm.value.realitySettings.publicKey = res.data.publicKey
  } finally { generatingKeys.value = false }
}

const showRealitySnInput = async () => { realitySnInputVisible.value = true; await nextTick(); realitySnInputRef.value?.focus() }
const addRealitySn = () => {
  const v = realitySnInput.value.trim()
  if (v && !nodeForm.value.realitySettings.serverNames.includes(v)) nodeForm.value.realitySettings.serverNames.push(v)
  realitySnInput.value = ''; realitySnInputVisible.value = false
}
const showRealitySidInput = async () => { realitySidInputVisible.value = true; await nextTick(); realitySidInputRef.value?.focus() }
const addRealitySid = () => {
  const v = realitySidInput.value.trim()
  if (v && !nodeForm.value.realitySettings.shortIds.includes(v)) nodeForm.value.realitySettings.shortIds.push(v)
  realitySidInput.value = ''; realitySidInputVisible.value = false
}

const addFallback = () => nodeForm.value.fallbacks.push({ dest: '', path: '', alpn: '' })

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
    ElMessage.success(t('commons.saveSuccess'))
    nodeDrawerVisible.value = false
    await loadNodes()
  } finally { nodeSubmitting.value = false }
}

const buildNodePayload = () => {
  const f = nodeForm.value
  const base: Record<string, unknown> = {
    name: f.name,
    protocol: f.protocol,
    listenAddr: f.listenAddr,
    port: f.port,
    network: f.network,
    security: f.security,
    flow: f.flow,
    sniffEnabled: f.sniffEnabled,
    sniffDestOverride: f.sniffDestOverride,
    sniffMetadataOnly: f.sniffMetadataOnly,
    remark: f.remark,
    enabled: f.enabled,
    // Shadowsocks
    ssMethod: f.ssMethod,
    ssPassword: f.ssPassword,
    // Fallbacks
    fallbacks: f.fallbacks.filter(fb => fb.dest),
  }
  const netMap: Record<string, object> = {
    raw: { rawSettings: f.rawSettings },
    ws: { wsSettings: f.wsSettings },
    grpc: { grpcSettings: f.grpcSettings },
    xhttp: { xhttpSettings: f.xhttpSettings },
    httpupgrade: { httpUpgradeSettings: f.httpUpgradeSettings },
  }
  const secMap: Record<string, object> = {
    tls: { tlsSettings: f.tlsSettings },
    reality: { realitySettings: f.realitySettings },
  }
  return { ...base, ...(netMap[f.network] || {}), ...(secMap[f.security] || {}) }
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
  if (!selectedNodeId.value) { users.value = []; return }
  userLoading.value = true
  try {
    const res = await searchXrayUsers({ nodeId: selectedNodeId.value, page: userPage.value, pageSize: userPageSize.value })
    users.value = res.data.items ?? []
    userTotal.value = res.data.total
  } finally { userLoading.value = false }
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
    Object.assign(userForm.value, {
      id: user.id, nodeId: user.nodeId, name: user.name, uuid: user.uuid,
      flow: user.flow || '', level: user.level, expireAt: user.expireAt,
      enabled: user.enabled, remark: user.remark,
    })
  }
  userDialogVisible.value = true
}

const generateUUID = () => { userForm.value.uuid = uuidv4() }

const submitUserForm = async () => {
  if (!userFormRef.value) return
  await userFormRef.value.validate()
  userSubmitting.value = true
  try {
    const payload = { ...userForm.value, nodeId: selectedNodeId.value }
    if (userForm.value.id) await updateXrayUser(payload)
    else await createXrayUser(payload)
    ElMessage.success(t('commons.saveSuccess'))
    userDialogVisible.value = false
    await loadUsers()
  } finally { userSubmitting.value = false }
}

const handleDeleteUser = async (id: number) => {
  await ElMessageBox.confirm(t('xray.deleteUserConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteXrayUser(id)
  ElMessage.success(t('commons.deleteSuccess'))
  await loadUsers()
}

// ============================================================
// 分享链接（客户端生成，可编辑地址/端口）
// ============================================================

const shareLinkVisible = ref(false)
const shareLinkHost = ref('')
const shareLinkPort = ref(443)
const shareLinkUser = ref<XrayUser | null>(null)
const shareLinkNode = ref<XrayNode | null>(null)
// 当节点 security=none 时，允许覆盖客户端加密（用于 nginx 反代场景）
const shareLinkOverrideSec = ref<'none' | 'tls'>('none')
const shareLinkOverrideSni = ref('')
const shareLinkOverrideAlpn = ref<string[]>([])
const shareLinkOverrideFp = ref('')

// 实时计算分享链接
const computedShareLink = computed(() => {
  const user = shareLinkUser.value
  const node = shareLinkNode.value
  const host = shareLinkHost.value.trim()
  const port = shareLinkPort.value
  if (!user || !node || !host) return ''
  return buildShareLinkClient(node, user, host, port, {
    security: node.security === 'none' ? shareLinkOverrideSec.value : undefined,
    sni: shareLinkOverrideSni.value,
    alpn: shareLinkOverrideAlpn.value,
    fingerprint: shareLinkOverrideFp.value,
  })
})

const getShareLink = (user: XrayUser) => {
  const node = nodes.value.find(n => n.id === user.nodeId)
  if (!node) { ElMessage.error('找不到对应节点'); return }
  shareLinkUser.value = user
  shareLinkNode.value = node
  // 自动填充连接地址
  let defaultHost = ''
  if (node.security === 'tls' && node.tlsSettings?.serverName) {
    defaultHost = node.tlsSettings.serverName
  } else if (node.security === 'reality' && node.realitySettings?.serverNames?.length) {
    defaultHost = node.realitySettings.serverNames[0]
  }
  shareLinkHost.value = defaultHost
  shareLinkPort.value = node.port
  // 当节点 security=none 时，默认建议使用 TLS（nginx 反代场景）
  shareLinkOverrideSec.value = 'tls'
  shareLinkOverrideSni.value = ''
  shareLinkOverrideAlpn.value = ['h2', 'http/1.1']
  shareLinkOverrideFp.value = 'chrome'
  shareLinkVisible.value = true
}

// 客户端分享链接生成器
interface ShareLinkOverride {
  security?: 'none' | 'tls'
  sni?: string
  alpn?: string[]
  fingerprint?: string
}

function buildShareLinkClient(node: XrayNode, user: XrayUser, host: string, port: number, override?: ShareLinkOverride): string {
  const userFlow = user.flow || node.flow || ''
  const name = encodeURIComponent(user.name)

  // 计算实际生效的安全设置
  const effectiveSecurity = override?.security ?? node.security
  // 构建 TLS 参数（可能来自节点设置或覆盖）
  const tlsSni = override?.sni || node.tlsSettings?.serverName || ''
  const tlsAlpn = (override?.alpn && override.alpn.length > 0) ? override.alpn : (node.tlsSettings?.alpn ?? [])
  const tlsFp = override?.fingerprint || node.tlsSettings?.fingerprint || ''

  switch (node.protocol) {
    case 'vless': {
      const p = new URLSearchParams()
      p.set('type', node.network === 'raw' ? 'tcp' : node.network)
      p.set('security', effectiveSecurity)
      if (userFlow) p.set('flow', userFlow)
      // 传输参数
      switch (node.network) {
        case 'ws':
          if (node.wsSettings?.path) p.set('path', node.wsSettings.path)
          if (node.wsSettings?.host) p.set('host', node.wsSettings.host)
          break
        case 'grpc':
          if (node.grpcSettings?.serviceName) p.set('serviceName', node.grpcSettings.serviceName)
          p.set('mode', node.grpcSettings?.multiMode ? 'multi' : 'gun')
          break
        case 'xhttp':
          if (node.xhttpSettings?.path) p.set('path', node.xhttpSettings.path)
          if (node.xhttpSettings?.host) p.set('host', node.xhttpSettings.host)
          if (node.xhttpSettings?.mode && node.xhttpSettings.mode !== 'auto') p.set('mode', node.xhttpSettings.mode)
          break
        case 'httpupgrade':
          if (node.httpUpgradeSettings?.path) p.set('path', node.httpUpgradeSettings.path)
          if (node.httpUpgradeSettings?.host) p.set('host', node.httpUpgradeSettings.host)
          break
      }
      // 安全参数
      if (effectiveSecurity === 'reality' && node.realitySettings) {
        p.set('pbk', node.realitySettings.publicKey)
        p.set('fp', node.realitySettings.fingerprint || 'chrome')
        if (node.realitySettings.shortIds?.length) p.set('sid', node.realitySettings.shortIds[0])
        if (node.realitySettings.serverNames?.length) p.set('sni', node.realitySettings.serverNames[0])
        p.set('spx', node.realitySettings.spiderX || '/')
      } else if (effectiveSecurity === 'tls') {
        if (tlsSni) p.set('sni', tlsSni)
        if (tlsFp) p.set('fp', tlsFp)
        if (tlsAlpn.length) p.set('alpn', tlsAlpn.join(','))
      }
      return `vless://${user.uuid}@${host}:${port}?${p.toString()}#${name}`
    }

    case 'vmess': {
      const v: Record<string, string | number> = {
        v: '2', ps: user.name, add: host, port: String(port),
        id: user.uuid, aid: '0', scy: 'auto',
        net: node.network === 'raw' ? 'tcp' : node.network,
        tls: effectiveSecurity === 'tls' ? 'tls' : '',
      }
      if (node.network === 'ws') {
        v.path = node.wsSettings?.path || '/'
        v.host = node.wsSettings?.host || ''
      } else if (node.network === 'grpc') {
        v.path = node.grpcSettings?.serviceName || 'grpc'
        v.type = node.grpcSettings?.multiMode ? 'multi' : 'gun'
      }
      if (effectiveSecurity === 'tls' && tlsSni) v.sni = tlsSni
      return 'vmess://' + btoa(unescape(encodeURIComponent(JSON.stringify(v))))
    }

    case 'trojan': {
      const p = new URLSearchParams()
      p.set('type', node.network === 'raw' ? 'tcp' : node.network)
      p.set('security', effectiveSecurity === 'none' ? 'none' : 'tls')
      if (effectiveSecurity === 'tls') {
        if (tlsSni) p.set('sni', tlsSni)
        if (tlsFp) p.set('fp', tlsFp)
        if (tlsAlpn.length) p.set('alpn', tlsAlpn.join(','))
      }
      if (node.network === 'ws' && node.wsSettings?.path) p.set('path', node.wsSettings.path)
      if (node.network === 'grpc' && node.grpcSettings?.serviceName) {
        p.set('serviceName', node.grpcSettings.serviceName)
        p.set('mode', node.grpcSettings.multiMode ? 'multi' : 'gun')
      }
      return `trojan://${user.uuid}@${host}:${port}?${p.toString()}#${name}`
    }

    default:
      return `# ${node.protocol} 暂不支持 URI 格式`
  }
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
    title: { text: `${user.name} - 30 天流量`, left: 'center', textStyle: { fontSize: 13 } },
    tooltip: {
      trigger: 'axis',
      formatter: (params: Array<{seriesName: string; value: number}>) =>
        params.map(p => `${p.seriesName}: ${formatBytes(p.value)}`).join('<br/>')
    },
    legend: { data: [t('xray.upload'), t('xray.download')], bottom: 0 },
    xAxis: { type: 'category', data: data.map(d => d.date), axisLabel: { rotate: 30, fontSize: 11 } },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v) } },
    series: [
      { name: t('xray.upload'), type: 'line', data: data.map(d => d.upload), smooth: true, itemStyle: { color: '#409EFF' } },
      { name: t('xray.download'), type: 'line', data: data.map(d => d.download), smooth: true, itemStyle: { color: '#67C23A' } },
    ],
    grid: { top: 48, left: 80, right: 16, bottom: 48 },
  })
}

watch(trafficDialogVisible, v => { if (!v) { chartInstance?.dispose(); chartInstance = null } })

// ============================================================
// Nginx 反代配置生成
// ============================================================

const nginxDialogVisible = ref(false)
const nginxForm = ref({
  network: 'ws',
  upstreamAddr: '127.0.0.1',
  upstreamPort: 0,
  path: '/ws',
  grpcServiceName: 'grpc',
  sendProxyProtocol: false,
})

const generatedNginxConfig = computed(() => generateNginxConfig())
const nginxConfigRows = computed(() => (generatedNginxConfig.value.split('\n').length + 1))

const openNginxDialog = (node: XrayNode) => {
  nginxForm.value.network = node.network
  nginxForm.value.upstreamAddr = node.listenAddr === '0.0.0.0' ? '127.0.0.1' : node.listenAddr
  nginxForm.value.upstreamPort = node.port
  // 预填 path
  if (node.wsSettings?.path) nginxForm.value.path = node.wsSettings.path
  else if (node.xhttpSettings?.path) nginxForm.value.path = node.xhttpSettings.path
  else if (node.httpUpgradeSettings?.path) nginxForm.value.path = node.httpUpgradeSettings.path
  else nginxForm.value.path = '/ws'
  if (node.grpcSettings?.serviceName) nginxForm.value.grpcServiceName = node.grpcSettings.serviceName
  nginxDialogVisible.value = true
}

function generateNginxConfig(): string {
  const f = nginxForm.value
  const upstream = `${f.upstreamAddr}:${f.upstreamPort}`

  // Proxy Protocol: 向 Xray 传递真实 IP
  const ppLine = f.sendProxyProtocol ? '\n    proxy_protocol     on;' : ''
  const realIpLine = f.sendProxyProtocol
    ? '\n    proxy_set_header   X-Real-IP          $proxy_protocol_addr;\n    proxy_set_header   X-Forwarded-For    $proxy_protocol_addr;'
    : '\n    proxy_set_header   X-Real-IP          $remote_addr;\n    proxy_set_header   X-Forwarded-For    $proxy_add_x_forwarded_for;'

  switch (f.network) {
    case 'ws':
    case 'httpupgrade':
      return `location ${f.path} {
    proxy_pass          http://${upstream};
    proxy_http_version  1.1;
    proxy_set_header    Upgrade            $http_upgrade;
    proxy_set_header    Connection         "upgrade";
    proxy_set_header    Host               $host;${realIpLine}
    proxy_set_header    X-Forwarded-Proto  $scheme;${ppLine}
    proxy_connect_timeout  60s;
    proxy_read_timeout     86400s;
    proxy_send_timeout     86400s;
    # 防止 nginx 缓冲 WebSocket 数据
    proxy_buffering        off;
}`

    case 'grpc': {
      const svc = f.grpcServiceName || 'grpc'
      return `# [!] 需确保 server 块已包含: listen 443 ssl; + http2 on;
location /${svc} {
    grpc_pass           grpc://${upstream};
    grpc_set_header     Host               $host;
    grpc_set_header     X-Real-IP          $remote_addr;
    grpc_connect_timeout  60s;
    grpc_read_timeout     86400s;
    grpc_send_timeout     86400s;
    # gRPC 流式传输不限制 body 大小
    client_max_body_size   0;
    client_body_timeout    86400s;
}`
    }

    case 'xhttp':
      return `location ${f.path} {
    proxy_pass          http://${upstream};
    proxy_http_version  1.1;
    # XHTTP/SplitHTTP 必须禁用 Connection 复用
    proxy_set_header    Connection         "";
    proxy_set_header    Host               $host;${realIpLine}${ppLine}
    # 关闭所有缓冲，保证流式传输
    proxy_buffering         off;
    proxy_cache             off;
    proxy_request_buffering off;
    proxy_connect_timeout   60s;
    proxy_read_timeout      86400s;
    proxy_send_timeout      86400s;
    # TCP 优化
    tcp_nodelay  on;
}`

    case 'raw':
      return `# TCP(RAW) 模式直接监听端口，无需 HTTP 反代
# 如需在同一端口与 HTTPS 共存，可使用 nginx stream 模块：
#
# stream {
#     upstream xray_backend {
#         server ${upstream};
#     }
#     server {
#         listen     443;
#         proxy_pass xray_backend;
#     }
# }
#
# 注意：stream 模块需在 nginx 主配置（nginx.conf）中配置，
# 不能放在 http {} 块内。`

    default:
      return `# 暂不支持 ${f.network} 的 nginx 反代模板`
  }
}

// ============================================================
// 工具函数
// ============================================================

const formatBytes = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}
const formatDate = (s: string) => s ? new Date(s).toLocaleDateString() : ''
const isExpired = (s: string) => new Date(s) < new Date()

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('commons.copySuccess'))
  } catch { ElMessage.error(t('commons.copyFailed')) }
}

const networkLabel = (n: string) => {
  const m: Record<string, string> = { raw: 'TCP', ws: 'WS', grpc: 'gRPC', xhttp: 'XHTTP', httpupgrade: 'HTTPUpgrade' }
  return m[n] ?? n.toUpperCase()
}
const protocolTagType = (p: string) => {
  const m: Record<string, string> = { vless: 'primary', vmess: 'success', trojan: 'warning', shadowsocks: 'info' }
  return (m[p] ?? 'info') as 'primary' | 'success' | 'warning' | 'info'
}
const securityTagType = (s: string) => {
  const m: Record<string, string> = { none: 'info', tls: 'success', reality: 'warning' }
  return (m[s] ?? 'info') as 'primary' | 'success' | 'warning' | 'info'
}

onMounted(() => { loadStatus() })
</script>

<style scoped>
.xray-container { padding: 16px; }

.install-banner { max-width: 800px; margin: 0 auto; }
.install-actions { margin-top: 16px; }
.install-log { margin-top: 16px; background: #1e1e1e; border-radius: 6px; padding: 12px; max-height: 400px; overflow-y: auto; }
.install-log pre { color: #d4d4d4; font-family: monospace; font-size: 13px; white-space: pre-wrap; margin: 0; }

.status-bar :deep(.el-card__body) { display: flex; align-items: center; justify-content: space-between; padding: 10px 16px; flex-wrap: wrap; gap: 8px; }
.status-bar { margin-bottom: 16px; }
.status-info { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.status-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.version-text { font-size: 13px; color: #606266; }
.config-path { font-size: 12px; color: #909399; font-family: monospace; }

.main-layout { min-height: calc(100vh - 200px); }
.node-card { height: 100%; }
.card-header { display: flex; justify-content: space-between; align-items: center; }

.node-item { padding: 10px; border-radius: 6px; cursor: pointer; margin-bottom: 8px; border: 1px solid var(--el-border-color-light); transition: all 0.2s; }
.node-item:hover { border-color: var(--el-color-primary-light-5); background: var(--el-color-primary-light-9); }
.node-item.active { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.node-item.disabled { opacity: 0.6; }
.node-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.node-name { font-weight: 600; font-size: 14px; margin-bottom: 4px; }
.node-meta { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 2px; flex-wrap: wrap; }
.meta-item { display: flex; align-items: center; gap: 2px; }
.flow-badge { font-size: 11px; color: var(--el-color-warning); background: var(--el-color-warning-light-9); padding: 1px 6px; border-radius: 3px; }
.node-actions { margin-top: 6px; display: flex; gap: 2px; }
.empty-text { text-align: center; color: var(--el-text-color-placeholder); padding: 32px 0; }

.uuid-cell { display: flex; align-items: center; gap: 4px; }
.uuid-text { font-family: monospace; font-size: 12px; color: var(--el-text-color-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 210px; }
.traffic-cell { font-size: 12px; line-height: 1.6; }
.traffic-up { color: var(--el-color-primary); }
.traffic-down { color: var(--el-color-success); }
.text-muted { color: var(--el-text-color-placeholder); }
.text-danger { color: var(--el-color-danger); }
.pagination { margin-top: 12px; justify-content: flex-end; display: flex; }

.node-tabs :deep(.el-tabs__header) { margin-bottom: 16px; }
.form-hint { font-size: 12px; color: var(--el-text-color-secondary); margin-left: 8px; display: inline; }
.key-row { display: flex; gap: 8px; width: 100%; }
.key-row .el-input { flex: 1; }
.tag-input-area { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.drawer-footer { display: flex; justify-content: flex-end; gap: 12px; }
.flow-warn { font-size: 12px; color: var(--el-color-warning); margin-top: 4px; display: flex; align-items: center; gap: 4px; }

.fallback-row { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }

.nginx-code-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.nginx-code-title { font-weight: 600; font-size: 14px; }
.nginx-code :deep(textarea) { font-family: 'Courier New', monospace !important; font-size: 13px !important; background: #1a1a2e !important; color: #e2e8f0 !important; }

.share-hint { font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.share-link-text :deep(textarea) { font-family: monospace !important; font-size: 12px !important; word-break: break-all; }

.mr-1 { margin-right: 4px; }
.ml-2 { margin-left: 8px; }
</style>
