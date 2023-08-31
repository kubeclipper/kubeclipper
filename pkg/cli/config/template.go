/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package config

const KcConsoleServiceTmpl = `[Unit]
Description=kc-console
Documentation=https://caddyserver.com/docs/
After=network.target network-online.target kc-server.service
Requires=network-online.target

[Service]
Type=simple
#User=caddy
#Group=caddy
ExecStart=/usr/local/bin/caddy run --config /etc/kc-console/Caddyfile
ExecReload=/usr/local/bin/caddy reload --config /etc/kc-console/Caddyfile
TimeoutStopSec=5s
LimitNOFILE=1048576
LimitNPROC=512
PrivateTmp=true
ProtectSystem=full
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target`

const KcCaddyTmpl = `{
	# General Options
	#debug
	admin off
	grace_period 1m
}
:{{.ConsolePort}} {
	root * /etc/kc-console/dist
	encode gzip
	log {
		output stdout
	}
	uri replace /apis/cluster/ /apis/cluster-proxy/
	uri strip_prefix /apis
	route {
		@kc {
			path /api/*
			path /oauth/*
			path /version
			path /cluster-proxy/*
		}
		reverse_proxy @kc {
			to {{.ServerUpstream}}
			lb_policy round_robin
			health_uri /healthz
			#health_port
			health_interval 10s
			health_timeout 5s
			health_status 2xx
			#health_body
			#health_headers
{{- if .TLS }}
			transport http {
				tls_trusted_ca_certs {{.CACert}}
				tls_server_name {{.TLSServerName}}
				#tls_insecure_skip_verify
            }
		}
{{- end}}
		try_files {path} {path}/ /index.html
		file_server
	}
}`

const EtcdServiceTmpl = `# /usr/lib/systemd/system/kc-etcd.service
[Unit]
Description=kc-etcd
Documentation=https://github.com/etcd-io/etcd
After=network-online.target

[Service]
Type=simple
Restart=on-failure
RestartSec=5s
TimeoutStartSec=0
ExecStart=/usr/local/bin/etcd --name {{.NodeName}} \
--advertise-client-urls=https://{{.AdvertiseAddress}} \
--auto-compaction-retention=1 \
--cert-file={{.ServerCertPath}} \
--client-cert-auth=true \
--data-dir={{.DataDIR}} \
--initial-advertise-peer-urls=https://{{.PeerAddress}} \
--initial-cluster={{.InitialCluster}} \
--initial-cluster-token={{.ClusterToken}} \
--initial-cluster-state=new \
--key-file={{.ServerCertKeyPath}} \
--listen-client-urls={{.ClientURLs}} \
--listen-metrics-urls={{.MetricsURLs}} \
--listen-peer-urls={{.PeerURLs}} \
--peer-cert-file={{.PeerCertPath}} \
--peer-client-cert-auth=true \
--peer-key-file={{.PeerCertKeyPath}} \
--peer-trusted-ca-file={{.CaPath}} \
--quota-backend-bytes=8589934592 \
--snapshot-count=5000 \
--trusted-ca-file={{.CaPath}}
ExecReload=/bin/kill -HUP
KillMode=process

[Install]
WantedBy=multi-user.target`

const KcServerService = `# /usr/lib/systemd/system/kc-server.service
[Unit]
Description=kc-server
After=kc-etcd.service

[Service]
Environment="HOME=/root"
Type=simple
Restart=on-failure
RestartSec=5s
TimeoutStartSec=0
ExecStart=/usr/local/bin/kubeclipper-server serve
ExecReload=/bin/kill -HUP
KillMode=process

[Install]
WantedBy=multi-user.target`

const KcAgentService = `# /usr/lib/systemd/system/kc-agent.service
[Unit]
Description=kubeclipper-agent

[Service]
Environment="HOME=/root"
Type=simple
Restart=on-failure
RestartSec=5s
TimeoutStartSec=0
ExecStart=/usr/local/bin/kubeclipper-agent serve
ExecReload=/bin/kill -HUP
KillMode=process

[Install]
WantedBy=multi-user.target`

const KcProxyService = `# /usr/lib/systemd/system/kc-proxy.service
[Unit]
Description=kc-proxy

[Service]
Environment="HOME=/root"
Type=simple
Restart=on-failure
RestartSec=5s
TimeoutStartSec=0
ExecStart=/usr/local/bin/kubeclipper-proxy serve
ExecReload=/bin/kill -HUP
KillMode=process

[Install]
WantedBy=multi-user.target`

const KcRegistryService = `# /usr/lib/systemd/system/kc-registry.service
[Unit]
Description=kc-registry

[Service]
Environment="HOME=/root"
Type=simple
Restart=on-failure
RestartSec=5s
TimeoutStartSec=0
ExecStart=/usr/local/bin/registry serve /etc/kubeclipper-registry/kubeclipper-registry.yaml
ExecReload=/bin/kill -HUP
KillMode=process

[Install]
WantedBy=multi-user.target`

const KcServerConfigTmpl = `generic:
  bindAddress: {{.ServerAddress}}
{{- if .TLS }}
  insecurePort: 0
  securePort: {{.ServerPort}}
  tlsCertFile: "{{.TLSCertFile}}"
  tlsPrivateKey: "{{.TLSPrivateKey}}"
  caCertFile: "{{.CACertFile}}"
{{- else }}
  insecurePort: {{.ServerPort}}
  securePort: 0
  tlsCertFile: ""
  tlsPrivateKey: ""
  caCertFile: ""
{{- end}}
authentication:
  authenticateRateLimiterMaxTries: {{.AuthenticateRateLimiterMaxTries}}
  authenticateRateLimiterDuration: {{.AuthenticateRateLimiterDuration}}
  loginHistoryMaximumEntries: {{.LoginHistoryMaximumEntries}}
  loginHistoryRetentionPeriod: {{.LoginHistoryRetentionPeriod}}
  maximumClockSkew: 10s
  multipleLogin: true
  jwtSecret: {{.JwtSecret}}
  initialPassword: {{.InitialPassword}}
audit:
  retentionPeriod: {{.RetentionPeriod}}
  maximumEntries: {{.MaximumEntries}}
  auditLevel: {{.AuditLevel}}
staticServer:
  bindAddress: {{.ServerAddress}}
  insecurePort: {{.StaticServerPort}}
  securePort: 0
  tlsCertFile: ""
  tlsPrivateKey: ""
  path: {{.StaticServerPath}}
log:
  logFile: ""
  logFileMaxSizeMB: 100
  toStderr: true
  level: {{.LogLevel}}
  encodeType: json
  maxBackups: 5
  maxAge: 30
  compress: false
  useLocalTime: true
etcd:
  serverList:
  {{range .EtcdEndpoints -}}
  - {{.}}
  {{end -}}
  keyFile: {{.EtcdKeyPath}}
  certFile: {{.EtcdCertPath}}
  trustedCAFile: {{.EtcdCaPath}}
  prefix: "/registry/kc-server"
  paging: true
  compactionInterval: 5m
  countMetricPollPeriod: 1m
  defaultStorageMediaType: "application/json"
  deleteCollectionWorkers: 1
  enableGarbageCollection: true
  enableWatchCache: true
  defaultWatchCacheSize: 100
  #watchCacheSizes
mq:
  external: {{.MQExternal}}
  client:
    serverAddress:
    {{range .MQServerEndpoints -}}
    - {{.}}
    {{end -}}
    subjectSuffix: kubeclipper
    queueGroupName: status-report-queue
    nodeReportSubject: status-report-subj
    timeOutSeconds: 10
    reconnectInterval: 2s
    maxReconnect: 600
    pingInterval: 2m
    maxPingsOut: 2
{{- if .MQTLS}}
    tls: {{.MQTLS}}
    tlsCaPath: {{.MQCaPath}}
    tlsCertPath: {{.MQClientCertPath}}
    tlsKeyPath: {{.MQClientKeyPath}}
{{- end }}
{{- if not .MQExternal }}
  server:
    host: {{.MQServerAddress}}
    port: {{.MQServerPort}}
    cluster:
      host: {{.MQServerAddress}}
      port: {{.MQClusterPort}}
      leaderHost: {{.LeaderHost}}
{{- if .MQTLS}}
    tls: {{.MQTLS}}
    tlsCaPath: {{.MQCaPath}}
    tlsCertPath: {{.MQServerCertPath}}
    tlsKeyPath: {{.MQServerKeyPath}}
{{- end }}
{{- end }}
  auth:
    username: {{.MQUser}}
    password: {{.MQAuthToken}}
`

const KcAgentConfigTmpl = `agentID: {{.AgentID}}
ipDetect: {{.IPDetect}}
nodeIPDetect: {{.NodeIPDetect}}
metadata:
  region: {{.Region}}
{{- if .FloatIP}}
  floatIP: {{.FloatIP}}
{{- end}}
registerNode: true
nodeStatusUpdateFrequency: 1m
downloader:
  address: {{.StaticServerAddress}}
  tlsCertFile: ""
  tlsPrivateKey: ""
log:
  logFile: ""
  logFileMaxSizeMB: 100
  toStderr: true
  level: {{.LogLevel}}
  encodeType: json
  maxBackups: 5
  maxAge: 30
  compress: false
  useLocalTime: true
mq:
  client:
    serverAddress:
    {{range .MQServerEndpoints -}}
    - {{.}}
    {{end -}}
    subjectSuffix: kubeclipper
    queueGroupName: status-report-queue
    nodeReportSubject: status-report-subj
    timeOutSeconds: 10
    reconnectInterval: 2s
    maxReconnect: 600
    pingInterval: 2m
    maxPingsOut: 2
{{- if .MQTLS}}
    tls: {{.MQTLS}}
    tlsCaPath: {{.MQCaPath}}
    tlsCertPath: {{.MQClientCertPath}}
    tlsKeyPath: {{.MQClientKeyPath}}
{{- end }}
  auth:
    username: admin
    password: {{.MQAuthToken}}
oplog:
  dir: {{.OpLogDir}}
  singleThreshold: {{.OpLogThreshold}}
backupStore:
  type: fs
  provider:
    rootdir: /opt/kc/backups
imageProxy:
  kcImageRepoMirror: {{.KcImageRepoMirror}}
`

const DockerDaemonTmpl = `
{
    "insecure-registries": ["{{.Node}}"],
    "data-root": "{{.DataRoot}}",
    "exec-opts": ["native.cgroupdriver=systemd"]
}
`

const KcRegistryConfigTmpl = `
# https://github.com/distribution/distribution/blob/main/docs/configuration.md
version: 0.1
log:
  level: info # error, warn, info, and debug
  formatter: text
  fields:
    service: registry
    environment: staging
storage:
  filesystem:
    rootdirectory: {{ .DataRoot }}
    maxthreads: 100
  delete:
    enabled: true
  cache:
    blobdescriptor: inmemory
http:
  addr: :{{ .RegistryPort }}
  headers:
    X-Content-Type-Options: [nosniff]
`
