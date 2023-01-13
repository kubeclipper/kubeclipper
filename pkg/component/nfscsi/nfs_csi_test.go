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

package nfscsi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderTo(t *testing.T) {
	expected := `---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-nfs-controller-sa
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-nfs-node-sa
  namespace: kube-system
---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nfs-external-provisioner-role
rules:
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
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]
  # for attacher https://github.com/kubernetes-csi/external-attacher/blob/c3741eaae2974a2292e562221a2d7ea684275366/deploy/kubernetes/rbac.yaml
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "patch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "patch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments/status"]
    verbs: ["patch"]
---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nfs-csi-provisioner-binding
subjects:
  - kind: ServiceAccount
    name: csi-nfs-controller-sa
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: nfs-external-provisioner-role
  apiGroup: rbac.authorization.k8s.io

---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-nfs-node
  namespace: kube-system
spec:
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
  selector:
    matchLabels:
      app: csi-nfs-node
  template:
    metadata:
      labels:
        app: csi-nfs-node
    spec:
      hostNetwork: true  # original nfs connection would be broken without hostNetwork setting
      dnsPolicy: Default  # available values: Default, ClusterFirstWithHostNet, ClusterFirst
      serviceAccountName: csi-nfs-node-sa
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
        - operator: "Exists"
      containers:
        - name: liveness-probe
          image: 192.168.1.1:5000/caas4/livenessprobe:v2.7.0
          args:
            - --csi-address=/csi/csi.sock
            - --probe-timeout=3s
            - --health-port=29653
            - --v=2
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
          resources:
            limits:
              memory: 100Mi
            requests:
              cpu: 10m
              memory: 20Mi
        - name: node-driver-registrar
          image: 192.168.1.1:5000/caas4/csi-node-driver-registrar:v2.5.1
          args:
            - --v=2
            - --csi-address=/csi/csi.sock
            - --kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)
          livenessProbe:
            exec:
              command:
                - /csi-node-driver-registrar
                - --kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)
                - --mode=kubelet-registration-probe
            initialDelaySeconds: 30
            timeoutSeconds: 15
          env:
            - name: DRIVER_REG_SOCK_PATH
              value: /plugins/csi-nfsplugin/csi.sock
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: registration-dir
              mountPath: /registration
          resources:
            limits:
              memory: 100Mi
            requests:
              cpu: 10m
              memory: 20Mi
        - name: nfs
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          image: 192.168.1.1:5000/caas4/nfsplugin:v4.1.0
          args:
            - "-v=5"
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix:///csi/csi.sock
          ports:
            - containerPort: 29653
              name: healthz
              protocol: TCP
          livenessProbe:
            failureThreshold: 5
            httpGet:
              path: /healthz
              port: healthz
            initialDelaySeconds: 30
            timeoutSeconds: 10
            periodSeconds: 30
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: pods-mount-dir
              mountPath: /pods
              mountPropagation: "Bidirectional"
          resources:
            limits:
              memory: 300Mi
            requests:
              cpu: 10m
              memory: 20Mi
      volumes:
        - name: socket-dir
          hostPath:
            path: /plugins/csi-nfsplugin
            type: DirectoryOrCreate
        - name: pods-mount-dir
          hostPath:
            path: /pods
            type: Directory
        - hostPath:
            path: /plugins_registry
            type: Directory
          name: registration-dir
---
apiVersion: storage.k8s.io/v1
kind: CSIDriver
metadata:
  name: nfs.csi.k8s.io
spec:
  attachRequired: true
  volumeLifecycleModes:
    - Persistent
    - Ephemeral
  fsGroupPolicy: File
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: csi-nfs-controller
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: csi-nfs-controller
  template:
    metadata:
      labels:
        app: csi-nfs-controller
    spec:
      hostNetwork: true  # controller also needs to mount nfs to create dir
      dnsPolicy: Default  # available values: Default, ClusterFirstWithHostNet, ClusterFirst
      serviceAccountName: csi-nfs-controller-sa
      nodeSelector:
        kubernetes.io/os: linux  # add "kubernetes.io/role: master" to run controller on master node
      priorityClassName: system-cluster-critical
      tolerations:
        - key: "node-role.kubernetes.io/master"
          operator: "Exists"
          effect: "NoSchedule"
        - key: "node-role.kubernetes.io/controlplane"
          operator: "Exists"
          effect: "NoSchedule"
        - key: "node-role.kubernetes.io/control-plane"
          operator: "Exists"
          effect: "NoSchedule"
      containers:
        - name: csi-provisioner
          image: 192.168.1.1:5000/caas4/csi-provisioner:v3.2.0
          args:
            - "-v=2"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election"
            - "--leader-election-namespace=kube-system"
            - "--extra-create-metadata=true"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          volumeMounts:
            - mountPath: /csi
              name: socket-dir
          resources:
            limits:
              memory: 400Mi
            requests:
              cpu: 10m
              memory: 20Mi
        - name: liveness-probe
          image: 192.168.1.1:5000/caas4/livenessprobe:v2.7.0
          args:
            - --csi-address=/csi/csi.sock
            - --probe-timeout=3s
            - --health-port=29652
            - --v=2
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
          resources:
            limits:
              memory: 100Mi
            requests:
              cpu: 10m
              memory: 20Mi
        #add attacher to support rwo
        - name: csi-attacher
          image: 192.168.1.1:5000/caas4/csi-attacher:v3.2.1
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election=true"
            - "--retry-interval-start=500ms"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          imagePullPolicy: "IfNotPresent"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: nfs
          image: 192.168.1.1:5000/caas4/nfsplugin:v4.1.0
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          imagePullPolicy: IfNotPresent
          args:
            - "-v=5"
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix:///csi/csi.sock
          ports:
            - containerPort: 29652
              name: healthz
              protocol: TCP
          livenessProbe:
            failureThreshold: 5
            httpGet:
              path: /healthz
              port: healthz
            initialDelaySeconds: 30
            timeoutSeconds: 10
            periodSeconds: 30
          volumeMounts:
            - name: pods-mount-dir
              mountPath: /pods
              mountPropagation: "Bidirectional"
            - mountPath: /csi
              name: socket-dir
          resources:
            limits:
              memory: 200Mi
            requests:
              cpu: 10m
              memory: 20Mi
      volumes:
        - name: pods-mount-dir
          hostPath:
            path: /pods
            type: Directory
        - name: socket-dir
          emptyDir: {}
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-sc
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: nfs.csi.k8s.io
parameters:
  server: 192.168.1.2
  share: /nfs/data
reclaimPolicy: Delete
volumeBindingMode: Immediate
mountOptions:
  - nfsvers=3
  - time=60`

	np := &NFS{
		Namespace:        defaultNamespace,
		ServerAddr:       "192.168.1.2",
		SharedPath:       "/nfs/data",
		StorageClassName: "nfs-sc",
		Replicas:         1,
		ReclaimPolicy:    defaultReclaimPolicy,
		IsDefault:        true,
		MountOptions:     []string{"nfsvers=3", "time=60"},
		ImageRepoMirror:  "192.168.1.1:5000",
	}
	sb := &strings.Builder{}
	if err := np.renderTo(sb); err != nil {
		assert.FailNow(t, "template renderring failed, err: %v", err)
	}
	if !assert.Equal(t, expected, sb.String()) {
		t.Errorf("expected is not the same as actual")
	}
}
