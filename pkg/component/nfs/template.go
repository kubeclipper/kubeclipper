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

package nfsprovisioner

const nameSpaceTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
`

// manifest reference https://github.com/kubernetes-csi/csi-driver-nfs/blob/v2.0.0/deploy/kubernetes/
const manifestsTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nfs-client-provisioner
  namespace: {{.Namespace}}

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nfs-client-provisioner-runner
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "update", "patch"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: run-nfs-client-provisioner
subjects:
  - kind: ServiceAccount
    name: nfs-client-provisioner
    namespace: {{.Namespace}}
roleRef:
  kind: ClusterRole
  name: nfs-client-provisioner-runner
  apiGroup: rbac.authorization.k8s.io

---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: leader-locking-nfs-client-provisioner
  namespace: {{.Namespace}}
rules:
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]

---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: leader-locking-nfs-client-provisioner
  namespace: {{.Namespace}}
subjects:
  - kind: ServiceAccount
    name: nfs-client-provisioner
    namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: leader-locking-nfs-client-provisioner
  apiGroup: rbac.authorization.k8s.io

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-client-provisioner-{{.StorageClassName}}
  labels:
    app: nfs-client-provisioner-{{.StorageClassName}}
  namespace: {{.Namespace}}
spec:
  replicas: {{with .Replicas}}{{.}}{{else}}1{{end}}
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: nfs-client-provisioner-{{.StorageClassName}}
  template:
    metadata:
      labels:
        app: nfs-client-provisioner-{{.StorageClassName}}
    spec:
      serviceAccountName: nfs-client-provisioner
      tolerations:
      - key: "node-role.kubernetes.io/master"
        operator: "Exists"
        effect: "NoSchedule"
      - key: node-role.kubernetes.io/control-plane
        operator: "Exists"
        effect: NoSchedule
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: In
                values:
                - ""
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/control-plane
                operator: In
                values:
                - ""
      containers:
        - name: nfs-client-provisioner
          image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/nfs-subdir-external-provisioner:v4.0.2
          volumeMounts:
            - name: nfs-client-root
              mountPath: /persistentvolumes
          env:
            - name: PROVISIONER_NAME
              value: k8s-sigs.io/nfs-subdir-external-provisioner-{{.StorageClassName}}
            - name: NFS_SERVER
              value: {{.ServerAddr}}
            - name: NFS_PATH
              value: {{.SharedPath}}
      volumes:
        - name: nfs-client-root
          nfs:
            server: {{.ServerAddr}}
            path: {{.SharedPath}}

---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: {{.StorageClassName}}
  {{- if .IsDefault}}
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
  {{- end}}
parameters:
  archiveOnDelete: "{{.ArchiveOnDelete}}"
reclaimPolicy: "{{.ReclaimPolicy}}"
{{- if .MountOptions}}
mountOptions:
  {{- range .MountOptions }}
  - {{ . }}
  {{- end }}
{{- end}}
provisioner: k8s-sigs.io/nfs-subdir-external-provisioner-{{.StorageClassName}}
`
