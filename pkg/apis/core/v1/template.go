package v1

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

const KcAgentConfigTmpl = `agentID: {{.AgentID}}
ipDetect: {{.IPDetect}}
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
