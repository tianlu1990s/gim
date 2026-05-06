# GIM K8S 部署指南

> 按照本文档一步步操作，从零搭建一主两从 K8S 集群，并将 gim 全部服务部署运行起来。

---

## 目录

1. [集群规划](#1-集群规划)
2. [服务器准备](#2-服务器准备)
3. [K8S 集群搭建](#3-k8s-集群搭建)
4. [集群验证与常用操作](#4-集群验证与常用操作)
5. [存储方案：NFS](#5-存储方案nfs)
6. [Helm 安装](#6-helm-安装)
7. [基础设施部署](#7-基础设施部署)
8. [gim 服务部署](#8-gim-服务部署)
9. [Ingress 与外部访问](#9-ingress-与外部访问)
10. [监控部署](#10-监控部署)
11. [日常运维操作](#11-日常运维操作)
12. [故障排查](#12-故障排查)

---

## 1. 集群规划

### 1.1 拓扑

```
┌─────────────────────────────────────────────────────────┐
│                    K8S 集群                               │
│                                                           │
│  ┌─────────────────┐  Master 节点                        │
│  │  kube-apiserver  │  IP: 192.168.1.10                   │
│  │  kube-scheduler  │  hostname: k8s-master               │
│  │  kube-controller │  OS: Debian 12                      │
│  │  etcd            │  CPU: 4C  RAM: 8G  Disk: 100G      │
│  │  NFS Server      │                                     │
│  └─────────────────┘                                     │
│                                                           │
│  ┌─────────────────┐  Worker 节点 1                      │
│  │  kubelet         │  IP: 192.168.1.11                   │
│  │  kube-proxy      │  hostname: k8s-worker1              │
│  │  Container       │  OS: Debian 12                      │
│  │  Runtime         │  CPU: 4C  RAM: 8G  Disk: 100G      │
│  └─────────────────┘                                     │
│                                                           │
│  ┌─────────────────┐  Worker 节点 2                      │
│  │  kubelet         │  IP: 192.168.1.12                   │
│  │  kube-proxy      │  hostname: k8s-worker2              │
│  │  Container       │  OS: Debian 12                      │
│  │  Runtime         │  OS: Debian 12                      │
│  └─────────────────┘  CPU: 4C  RAM: 8G  Disk: 100G      │
│                                                           │
│  ┌───────────────────────────────────────────────────┐   │
│  │  Pod 网络: Calico (10.244.0.0/16)                  │   │
│  │  Service 网络: 10.96.0.0/12                        │   │
│  └───────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 1.2 最低配置

| 节点 | CPU | 内存 | 磁盘 | 用途 |
|------|-----|------|------|------|
| k8s-master | 4C | 8G | 100G | 控制面 + NFS + 基础设施 |
| k8s-worker1 | 4C | 8G | 100G | gim 服务 Pod |
| k8s-worker2 | 4C | 8G | 100G | gim 服务 Pod |

> 如果是本地测试，可以用 3 台虚拟机（VirtualBox/VMware）或云服务器。Master 最低可降到 2C4G，但生产不建议。

### 1.3 网络规划

| 网段 | 用途 |
|------|------|
| 192.168.1.0/24 | 物理网络（节点间通信） |
| 10.244.0.0/16 | Pod 网络（Calico） |
| 10.96.0.0/12 | Service 网络（ClusterIP） |

---

## 2. 服务器准备

> 以下步骤在**所有 3 台机器上**执行，除非特别说明。

### 2.1 设置 hostname 和 hosts

```bash
# 在 192.168.1.10 上执行
sudo hostnamectl set-hostname k8s-master

# 在 192.168.1.11 上执行
sudo hostnamectl set-hostname k8s-worker1

# 在 192.168.1.12 上执行
sudo hostnamectl set-hostname k8s-worker2
```

在**所有节点**的 `/etc/hosts` 中添加：

```bash
sudo tee -a /etc/hosts <<EOF
192.168.1.10 k8s-master
192.168.1.11 k8s-worker1
192.168.1.12 k8s-worker2
EOF
```

验证：

```bash
ping -c 2 k8s-master
ping -c 2 k8s-worker1
ping -c 2 k8s-worker2
```

### 2.2 关闭 swap

K8S 要求关闭 swap，否则 kubelet 启动失败。

```bash
# 临时关闭
sudo swapoff -a

# 永久关闭（注释掉 fstab 中的 swap 行）
sudo sed -i '/swap/s/^/#/' /etc/fstab

# 验证（应该没有输出）
free -h | grep Swap
```

### 2.3 加载内核模块

```bash
sudo tee /etc/modules-load.d/k8s.conf <<EOF
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter
```

### 2.4 内核网络参数

```bash
sudo tee /etc/sysctl.d/k8s.conf <<EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sudo sysctl --system

# 验证
sysctl net.bridge.bridge-nf-call-iptables
# 应输出 net.bridge.bridge-nf-call-iptables = 1
```

### 2.5 安装 containerd

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg

# 添加 Docker 官方 GPG key 和仓库
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo $VERSION_CODENAME) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y containerd.io
```

配置 containerd 使用 SystemdCgroup（K8S 要求）：

```bash
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml > /dev/null

# 修改 SystemdCgroup 为 true
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

# 重启 containerd
sudo systemctl restart containerd
sudo systemctl enable containerd
```

验证：

```bash
sudo systemctl status containerd
# 应显示 active (running)
```

### 2.6 安装 kubeadm、kubelet、kubectl

```bash
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl gpg

# 添加 Kubernetes apt 仓库
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list

sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

> `apt-mark hold` 防止这些包被自动升级，K8S 版本升级需要手动控制。

验证：

```bash
kubeadm version
# 输出类似：kubeadm version: &version.Info{Major:"1", Minor:"29", ...}
```

---

## 3. K8S 集群搭建

### 3.1 Master 节点初始化

**仅在 k8s-master 上执行：**

```bash
sudo kubeadm init \
  --apiserver-advertise-address=192.168.1.10 \
  --pod-network-cidr=10.244.0.0/16 \
  --service-cidr=10.96.0.0/12 \
  --kubernetes-version=v1.29.3 \
  --ignore-preflight-errors=NumCPU
```

> `--ignore-preflight-errors=NumCPU`：如果 Master 只有 2 核，跳过 CPU 核数检查。生产环境不建议。

成功后会输出类似：

```
Your Kubernetes control-plane has initialized successfully!

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 192.168.1.10:6443 --token xxxxxx.xxxxxxxxxxxxxxxx \
        --discovery-token-ca-cert-hash sha256:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**保存这个 join 命令！** 后面 Worker 节点要用。

### 3.2 配置 kubectl

**仅在 k8s-master 上执行：**

```bash
# 为当前用户配置 kubeconfig
mkdir -p $HOME/.kube
sudo cp /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# 验证
kubectl get nodes
# 输出：
# NAME          STATUS     ROLES           AGE   VERSION
# k8s-master    NotReady   control-plane   60s   v1.29.3
# NotReady 是正常的，因为还没装网络插件
```

💡 **kubectl 命令只能从有 kubeconfig 的机器上执行。** 默认是 Master 节点。如果你想从本地电脑远程操作，把 Master 上的 `~/.kube/config` 复制到本地即可。

### 3.3 安装 Pod 网络插件（Calico）

**仅在 k8s-master 上执行：**

```bash
# 下载 Calico manifests
curl -O https://raw.githubusercontent.com/projectcalico/calico/v3.27.0/manifests/calico.yaml

# 应用
kubectl apply -f calico.yaml

# 等待 Calico Pod 启动（约 1-2 分钟）
kubectl get pods -n kube-system -w
# 等 calico-node 全部 Running 后 Ctrl+C
```

验证节点状态：

```bash
kubectl get nodes
# NAME          STATUS   ROLES           AGE   VERSION
# k8s-master    Ready    control-plane   5m    v1.29.3
# Ready 了！
```

### 3.4 Worker 节点加入集群

**在 k8s-worker1 和 k8s-worker2 上分别执行** 3.1 步骤中保存的 join 命令：

```bash
sudo kubeadm join 192.168.1.10:6443 --token xxxxxx.xxxxxxxxxxxxxxxx \
        --discovery-token-ca-cert-hash sha256:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

> 如果 join 命令丢失或 token 过期，在 Master 上重新生成：
> ```bash
> kubeadm token create --print-join-command
> ```

**回到 Master 验证：**

```bash
kubectl get nodes
# NAME           STATUS   ROLES           AGE   VERSION
# k8s-master     Ready    control-plane   10m   v1.29.3
# k8s-worker1    Ready    <none>          60s   v1.29.3
# k8s-worker2    Ready    <none>          60s   v1.29.3
```

三个节点都是 Ready，集群搭建完成！

### 3.5 允许 Master 调度 Pod（可选）

默认 Master 有 `node-role.kubernetes.io/control-plane` 污点，不调度业务 Pod。如果你只有 3 台机器且想让 Master 也跑 Pod：

```bash
kubectl taint nodes k8s-master node-role.kubernetes.io/control-plane-
```

> 生产环境不建议。去掉污点后，业务 Pod 可能占用 Master 资源，影响控制面稳定性。

---

## 4. 集群验证与常用操作

### 4.1 验证集群健康

```bash
# 查看所有系统 Pod
kubectl get pods -n kube-system

# 查看组件状态
kubectl get componentstatuses

# 查看集群信息
kubectl cluster-info

# 查看节点详情
kubectl describe node k8s-master
```

所有 kube-system Pod 应为 Running 或 Completed。

### 4.2 常用 kubectl 命令速查

```bash
# 资源查看
kubectl get nodes                      # 节点列表
kubectl get pods -A                     # 所有命名空间的 Pod
kubectl get pods -n gim                 # gim 命名空间的 Pod
kubectl get svc -n gim                  # Service 列表
kubectl get deploy -n gim               # Deployment 列表
kubectl get sts -n gim                  # StatefulSet 列表

# 详情查看
kubectl describe pod <pod-name> -n gim  # Pod 详情（排错必备）
kubectl logs <pod-name> -n gim          # Pod 日志
kubectl logs <pod-name> -n gim -f       # 实时跟踪日志
kubectl logs <pod-name> -n gim --tail=100  # 最近 100 行

# 操作
kubectl apply -f xxx.yaml               # 应用配置
kubectl delete -f xxx.yaml              # 删除配置
kubectl exec -it <pod-name> -n gim -- bash  # 进入容器
kubectl port-forward svc/mysql 3306:3306 -n gim  # 端口转发到本地

# 扩缩容
kubectl scale deploy gim-api --replicas=3 -n gim

# 标签和选择
kubectl get pods -n gim -l app=gim-api  # 按标签筛选
```

💡 **`-n gim` 是指定命名空间。** 我们把所有 gim 相关资源放在 `gim` 命名空间，和系统资源隔离。忘记加 `-n` 会查 `default` 命名空间，可能看不到你的 Pod。

---

## 5. 存储方案：NFS

> K8S 中 MySQL/MongoDB/Kafka 需要持久化存储。我们用 NFS 作为共享存储，Master 节点当 NFS Server。

### 5.1 Master 安装 NFS Server

**仅在 k8s-master 上执行：**

```bash
sudo apt-get install -y nfs-kernel-server

# 创建共享目录
sudo mkdir -p /data/nfs/{mysql,mongodb,kafka,redis,minio}

# 配置导出
sudo tee -a /etc/exports <<EOF
/data/nfs 192.168.1.0/24(rw,sync,no_subtree_check,no_root_squash)
EOF

# 生效
sudo exportfs -av
sudo systemctl restart nfs-kernel-server
sudo systemctl enable nfs-kernel-server
```

验证：

```bash
showmount -e k8s-master
# Export list for k8s-master:
# /data/nfs 192.168.1.0/24
```

### 5.2 Worker 安装 NFS Client

**在 k8s-worker1 和 k8s-worker2 上执行：**

```bash
sudo apt-get install -y nfs-common
```

测试挂载：

```bash
sudo mkdir -p /mnt/nfs
sudo mount -t nfs k8s-master:/data/nfs /mnt/nfs
ls /mnt/nfs
# 应看到 mysql mongodb kafka redis minio 目录
sudo umount /mnt/nfs
```

### 5.3 创建 NFS Provisioner

**在 k8s-master 上执行：**

```bash
# 安装 NFS Subdir External Provisioner
helm repo add nfs-subdir-external-provisioner https://kubernetes-sigs.github.io/nfs-subdir-external-provisioner/
helm repo update

helm install nfs-provisioner nfs-subdir-external-provisioner/nfs-subdir-external-provisioner \
  --namespace kube-system \
  --set nfs.server=k8s-master \
  --set nfs.path=/data/nfs \
  --set storageClass.name=nfs \
  --set storageClass.defaultClass=true \
  --set replicaCount=1
```

验证：

```bash
kubectl get storageclass
# NAME            PROVISIONER                                     RECLAIMPOLICY   VOLUMEBINDINGMODE
# nfs (default)   cluster.local/nfs-subdir-external-provisioner   Delete          Immediate
```

💡 **StorageClass 是什么？** K8S 中 PV（持久卷）的"自动创建器"。当 Pod 声明需要存储时，StorageClass 会自动在 NFS 上创建一个目录并挂载给 Pod，无需手动创建 PV。

---

## 6. Helm 安装

**在 k8s-master 上执行：**

```bash
# 安装 Helm 3
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 验证
helm version
# version.BuildInfo{Version:"v3.14.4", ...}

# 添加常用仓库
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

---

## 7. 基础设施部署

> MySQL、MongoDB、Redis、Kafka、MinIO 是 gim 运行的依赖，先部署这些。

### 7.1 创建命名空间

```bash
kubectl create namespace gim
```

### 7.2 部署 MySQL 8.0

```bash
helm install mysql bitnami/mysql \
  --namespace gim \
  --set auth.rootPassword=root123456 \
  --set auth.database=gim \
  --set auth.username=gim \
  --set auth.password=gim_pass \
  --set primary.persistence.size=20Gi \
  --set primary.persistence.storageClass=nfs \
  --set primary.resources.requests.cpu=500m \
  --set primary.resources.requests.memory=512Mi \
  --set primary.resources.limits.cpu=1000m \
  --set primary.resources.limits.memory=1Gi \
  --set architecture=standalone \
  --set image.tag=8.0
```

验证：

```bash
kubectl get pods -n gim -l app.kubernetes.io/name=mysql
# NAME       READY   STATUS    RESTARTS   AGE
# mysql-0    1/1     Running   0          2m
```

连接测试：

```bash
kubectl run mysql-client --rm -it --restart=Never --image=mysql:8.0 -n gim -- \
  mysql -h mysql.gim.svc.cluster.local -ugim -pgim_pass gim -e "SELECT 1;"
# 输出：+---+
#       | 1 |
#       +---+
```

### 7.3 部署 Redis 7

```bash
helm install redis bitnami/redis \
  --namespace gim \
  --set auth.password=redis_pass123 \
  --set master.persistence.size=5Gi \
  --set master.persistence.storageClass=nfs \
  --set master.resources.requests.cpu=250m \
  --set master.resources.requests.memory=256Mi \
  --set replica.replicaCount=1 \
  --set replica.persistence.size=5Gi \
  --set replica.persistence.storageClass=nfs \
  --set architecture=replication
```

验证：

```bash
kubectl get pods -n gim -l app.kubernetes.io/name=redis
# NAME         READY   STATUS    RESTARTS   AGE
# redis-master-0   1/1     Running   0          2m
# redis-replica-0  1/1     Running   0          2m
```

连接测试：

```bash
kubectl run redis-client --rm -it --restart=Never --image=redis:7-alpine -n gim -- \
  redis-cli -h redis-master.gim.svc.cluster.local -a redis_pass123 ping
# 输出：PONG
```

### 7.4 部署 MongoDB（第二阶段需要）

```bash
helm install mongodb bitnami/mongodb \
  --namespace gim \
  --set auth.rootPassword=mongo_pass123 \
  --set auth.username=gim \
  --set auth.password=gim_mongo_pass \
  --set auth.database=gim \
  --set persistence.size=20Gi \
  --set persistence.storageClass=nfs \
  --set resources.requests.cpu=500m \
  --set resources.requests.memory=512Mi \
  --set architecture=standalone
```

### 7.5 部署 Kafka（第二阶段需要）

```bash
helm install kafka bitnami/kafka \
  --namespace gim \
  --set controller.replicaCount=1 \
  --set broker.replicaCount=2 \
  --set persistence.size=10Gi \
  --set persistence.storageClass=nfs \
  --set controller.resources.requests.cpu=500m \
  --set controller.resources.requests.memory=512Mi \
  --set broker.resources.requests.cpu=500m \
  --set broker.resources.requests.memory=512Mi \
  --set extraConfig."log\.retention\.hours"=72
```

验证：

```bash
kubectl get pods -n gim -l app.kubernetes.io/name=kafka
# NAME          READY   STATUS    RESTARTS   AGE
# kafka-broker-0   1/1     Running   0          3m
# kafka-broker-1   1/1     Running   0          3m
```

### 7.6 部署 MinIO（第二阶段需要）

```bash
helm install minio bitnami/minio \
  --namespace gim \
  --set auth.rootUser=admin \
  --set auth.rootPassword=minio_pass123 \
  --set mode=standalone \
  --set persistence.size=50Gi \
  --set persistence.storageClass=nfs \
  --set resources.requests.cpu=250m \
  --set resources.requests.memory=256Mi
```

### 7.7 部署 etcd（第二阶段需要）

```bash
helm install etcd bitnami/etcd \
  --namespace gim \
  --set auth.rbac.create=false \
  --set replicaCount=3 \
  --set persistence.size=5Gi \
  --set persistence.storageClass=nfs \
  --set resources.requests.cpu=250m \
  --set resources.requests.memory=256Mi
```

### 7.8 基础设施连接信息汇总

| 组件 | K8S Service 地址 | 端口 | 用户名 | 密码 |
|------|-----------------|------|--------|------|
| MySQL | `mysql.gim.svc.cluster.local` | 3306 | gim | gim_pass |
| Redis | `redis-master.gim.svc.cluster.local` | 6379 | - | redis_pass123 |
| MongoDB | `mongodb.gim.svc.cluster.local` | 27017 | gim | gim_mongo_pass |
| Kafka | `kafka-broker-0.kafka-broker-headless.gim.svc.cluster.local` | 9092 | - | - |
| MinIO | `minio.gim.svc.cluster.local` | 9000 | admin | minio_pass123 |
| etcd | `etcd-headless.gim.svc.cluster.local` | 2379 | - | - |

> `gim.svc.cluster.local` 是 K8S 集群内部 DNS 域名。集群内的 Pod 可以直接用 Service 名（如 `mysql`）访问，跨命名空间用全限定名（如 `mysql.gim.svc.cluster.local`）。

---

## 8. gim 服务部署

### 8.1 Helm Chart 结构

在项目中创建 Helm Chart：

```bash
mkdir -p deploy/k8s/helm/gim
cd deploy/k8s/helm/gim
```

完整的 Chart 目录结构：

```
deploy/k8s/helm/gim/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── api-deployment.yaml
│   ├── api-service.yaml
│   ├── ws-statefulset.yaml
│   ├── ws-service.yaml
│   ├── rpc-auth-deployment.yaml
│   ├── rpc-auth-service.yaml
│   ├── rpc-user-deployment.yaml
│   ├── rpc-user-service.yaml
│   ├── rpc-msg-deployment.yaml
│   ├── rpc-msg-service.yaml
│   ├── push-deployment.yaml
│   ├── push-service.yaml
│   ├── msgtransfer-deployment.yaml
│   ├── admin-deployment.yaml
│   ├── admin-service.yaml
│   ├── ai-deployment.yaml
│   ├── ai-service.yaml
│   ├── ingress.yaml
│   └── hpa.yaml
└── .helmignore
```

### 8.2 Chart.yaml

```yaml
# deploy/k8s/helm/gim/Chart.yaml
apiVersion: v2
name: gim
description: GIM - Go Instant Messaging System
type: application
version: 0.1.0
appVersion: "1.0.0"
maintainers:
  - name: gim-team
```

### 8.3 values.yaml

```yaml
# deploy/k8s/helm/gim/values.yaml

# 全局配置
global:
  namespace: gim
  imageRegistry: ""
  imagePullSecrets: []
  storageClass: nfs

# 镜像标签（统一版本）
image:
  tag: "latest"
  pullPolicy: IfNotPresent

# API 网关
api:
  enabled: true
  replicas: 2
  image:
    repository: gim/api
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    type: ClusterIP
    port: 10002

# WS Gateway
ws:
  enabled: true
  replicas: 2
  image:
    repository: gim/ws
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 1000m
      memory: 1Gi
  service:
    type: NodePort
    wsPort: 10001
    grpcPort: 10140
    nodePortWs: 30001
  maxConnections: 50000

# Auth RPC
rpcAuth:
  enabled: true
  replicas: 1
  image:
    repository: gim/rpc-auth
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    port: 10100

# User RPC
rpcUser:
  enabled: true
  replicas: 1
  image:
    repository: gim/rpc-user
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    port: 10110

# Msg RPC
rpcMsg:
  enabled: true
  replicas: 1
  image:
    repository: gim/rpc-msg
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    port: 10120

# Push 服务
push:
  enabled: true
  replicas: 2
  image:
    repository: gim/push
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    port: 10170

# MsgTransfer
msgtransfer:
  enabled: true
  replicas: 2
  image:
    repository: gim/msgtransfer
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 1000m
      memory: 1Gi

# Admin API
admin:
  enabled: true
  replicas: 1
  image:
    repository: gim/admin
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    type: ClusterIP
    port: 10004

# AI 服务
ai:
  enabled: true
  replicas: 1
  image:
    repository: gim/ai
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    port: 10180

# 基础设施连接（引用前面部署的服务）
mysql:
  host: mysql.gim.svc.cluster.local
  port: 3306
  username: gim
  password: gim_pass
  database: gim
  maxOpenConns: 100
  maxIdleConns: 20

redis:
  host: redis-master.gim.svc.cluster.local
  port: 6379
  password: redis_pass123

mongodb:
  host: mongodb.gim.svc.cluster.local
  port: 27017
  username: gim
  password: gim_mongo_pass
  database: gim

kafka:
  brokers: "kafka-broker-0.kafka-broker-headless.gim.svc.cluster.local:9092,kafka-broker-1.kafka-broker-headless.gim.svc.cluster.local:9092"

minio:
  endpoint: minio.gim.svc.cluster.local:9000
  accessKey: admin
  secretKey: minio_pass123
  bucket: gim

etcd:
  endpoints: "etcd-0.etcd-headless.gim.svc.cluster.local:2379,etcd-1.etcd-headless.gim.svc.cluster.local:2379,etcd-2.etcd-headless.gim.svc.cluster.local:2379"

# JWT 密钥（生产环境务必更换）
jwt:
  privateKey: |-
    -----BEGIN RSA PRIVATE KEY-----
    （替换为你的私钥内容，用 kubectl create secret 更安全）
    -----END RSA PRIVATE KEY-----
  publicKey: |-
    -----BEGIN PUBLIC KEY-----
    （替换为你的公钥内容）
    -----END PUBLIC KEY-----

# Ingress
ingress:
  enabled: true
  className: nginx
  host: gim.example.com
  wsHost: ws.gim.example.com
  tls: false

# HPA
hpa:
  enabled: true
  ws:
    minReplicas: 2
    maxReplicas: 10
    targetCPU: 70
    targetConnections: 5000
  api:
    minReplicas: 2
    maxReplicas: 5
    targetCPU: 70
```

### 8.4 _helpers.tpl

```gotemplate
{{/* deploy/k8s/helm/gim/templates/_helpers.tpl */}}

{{- define "gim.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "gim.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "gim.labels" -}}
app.kubernetes.io/name: {{ include "gim.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "gim.selectorLabels" -}}
app: {{ include "gim.name" . }}
{{- end }}

{{/* 通用容器环境变量 */}}
{{- define "gim.commonEnv" -}}
- name: GIM_MYSQL_HOST
  value: {{ .Values.mysql.host }}
- name: GIM_MYSQL_PORT
  value: {{ .Values.mysql.port | quote }}
- name: GIM_MYSQL_USERNAME
  value: {{ .Values.mysql.username }}
- name: GIM_MYSQL_PASSWORD
  valueFrom:
    secretKeyRef:
      name: gim-secret
      key: mysql-password
- name: GIM_MYSQL_DATABASE
  value: {{ .Values.mysql.database }}
- name: GIM_REDIS_HOST
  value: {{ .Values.redis.host }}
- name: GIM_REDIS_PORT
  value: {{ .Values.redis.port | quote }}
- name: GIM_REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: gim-secret
      key: redis-password
{{- end }}
```

### 8.5 Secret

```yaml
# deploy/k8s/helm/gim/templates/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: gim-secret
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "gim.labels" . | nindent 4 }}
type: Opaque
stringData:
  mysql-password: {{ .Values.mysql.password | quote }}
  redis-password: {{ .Values.redis.password | quote }}
  mongodb-password: {{ .Values.mongodb.password | quote }}
  minio-secret-key: {{ .Values.minio.secretKey | quote }}
  jwt-private-key: |
    {{ .Values.jwt.privateKey | nindent 4 }}
  jwt-public-key: |
    {{ .Values.jwt.publicKey | nindent 4 }}
```

### 8.6 ConfigMap

```yaml
# deploy/k8s/helm/gim/templates/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gim-config
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "gim.labels" . | nindent 4 }}
data:
  config.yaml: |
    server:
      httpPort: {{ .Values.api.service.port }}
      wsPort: {{ .Values.ws.service.wsPort }}
      readTimeout: 10s
      writeTimeout: 10s

    mysql:
      host: {{ .Values.mysql.host }}
      port: {{ .Values.mysql.port }}
      user: {{ .Values.mysql.username }}
      password: ""      # 从 Secret 读取，这里留空
      dbname: {{ .Values.mysql.database }}
      maxOpenConns: {{ .Values.mysql.maxOpenConns }}
      maxIdleConns: {{ .Values.mysql.maxIdleConns }}
      connMaxLifetime: 300s

    redis:
      host: {{ .Values.redis.host }}
      port: {{ .Values.redis.port }}
      password: ""      # 从 Secret 读取
      db: 0
      poolSize: 100

    mongodb:
      host: {{ .Values.mongodb.host }}
      port: {{ .Values.mongodb.port }}
      username: {{ .Values.mongodb.username }}
      password: ""      # 从 Secret 读取
      database: {{ .Values.mongodb.database }}

    kafka:
      brokers: {{ .Values.kafka.brokers }}

    minio:
      endpoint: {{ .Values.minio.endpoint }}
      accessKey: {{ .Values.minio.accessKey }}
      secretKey: ""     # 从 Secret 读取
      bucket: {{ .Values.minio.bucket }}

    etcd:
      endpoints: {{ .Values.etcd.endpoints }}

    jwt:
      accessTokenExpire: 2h
      refreshTokenExpire: 168h
      privateKeyPath: /etc/gim/secrets/jwt-private-key
      publicKeyPath: /etc/gim/secrets/jwt-public-key

    websocket:
      maxConnPerUser: 5
      maxMessageSize: 4096
      writeWait: 10s
      pongWait: 60s
      pingPeriod: 30s

    log:
      level: info
      format: json
      output: stdout
```

### 8.7 API 网关 Deployment + Service

```yaml
# deploy/k8s/helm/gim/templates/api-deployment.yaml
{{- if .Values.api.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gim-api
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-api
    {{- include "gim.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.api.replicas }}
  selector:
    matchLabels:
      app: gim-api
  template:
    metadata:
      labels:
        app: gim-api
    spec:
      containers:
      - name: gim-api
        image: "{{ .Values.global.imageRegistry }}{{ .Values.api.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
        - containerPort: {{ .Values.api.service.port }}
        env:
          {{- include "gim.commonEnv" . | nindent 10 }}
        envFrom:
        - configMapRef:
            name: gim-config
        volumeMounts:
        - name: config
          mountPath: /etc/gim/configs
        - name: secrets
          mountPath: /etc/gim/secrets
          readOnly: true
        resources:
          {{- toYaml .Values.api.resources | nindent 10 }}
        livenessProbe:
          httpGet:
            path: /healthz
            port: {{ .Values.api.service.port }}
          initialDelaySeconds: 10
          periodSeconds: 15
        readinessProbe:
          httpGet:
            path: /healthz
            port: {{ .Values.api.service.port }}
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: gim-config
      - name: secrets
        secret:
          secretName: gim-secret
{{- end }}
```

```yaml
# deploy/k8s/helm/gim/templates/api-service.yaml
{{- if .Values.api.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: gim-api
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-api
spec:
  type: {{ .Values.api.service.type }}
  selector:
    app: gim-api
  ports:
  - port: {{ .Values.api.service.port }}
    targetPort: {{ .Values.api.service.port }}
    protocol: TCP
    name: http
{{- end }}
```

### 8.8 WS Gateway StatefulSet + Service

💡 **为什么 WS Gateway 用 StatefulSet 而不是 Deployment？** Push 服务需要通过 gRPC 调用持有目标用户连接的 Gateway 实例。StatefulSet 提供稳定的网络标识（pod-0、pod-1 不变），Push 服务可以逐个尝试每个 Gateway 实例。Deployment 的 Pod 名字是随机的，每次重建都变。

```yaml
# deploy/k8s/helm/gim/templates/ws-statefulset.yaml
{{- if .Values.ws.enabled }}
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: gim-ws
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-ws
    {{- include "gim.labels" . | nindent 4 }}
spec:
  serviceName: gim-ws-headless
  replicas: {{ .Values.ws.replicas }}
  selector:
    matchLabels:
      app: gim-ws
  template:
    metadata:
      labels:
        app: gim-ws
    spec:
      containers:
      - name: gim-ws
        image: "{{ .Values.global.imageRegistry }}{{ .Values.ws.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
        - containerPort: {{ .Values.ws.service.wsPort }}
          name: ws
        - containerPort: {{ .Values.ws.service.grpcPort }}
          name: grpc
        env:
          {{- include "gim.commonEnv" . | nindent 10 }}
        envFrom:
        - configMapRef:
            name: gim-config
        volumeMounts:
        - name: config
          mountPath: /etc/gim/configs
        - name: secrets
          mountPath: /etc/gim/secrets
          readOnly: true
        resources:
          {{- toYaml .Values.ws.resources | nindent 10 }}
        livenessProbe:
          tcpSocket:
            port: {{ .Values.ws.service.wsPort }}
          initialDelaySeconds: 10
          periodSeconds: 15
        readinessProbe:
          tcpSocket:
            port: {{ .Values.ws.service.wsPort }}
          initialDelaySeconds: 5
          periodSeconds: 10
      terminationGracePeriodSeconds: 60  # WS 连接优雅关闭
      volumes:
      - name: config
        configMap:
          name: gim-config
      - name: secrets
        secret:
          secretName: gim-secret
{{- end }}
```

```yaml
# deploy/k8s/helm/gim/templates/ws-service.yaml
{{- if .Values.ws.enabled }}
# Headless Service（StatefulSet 必需，提供稳定 DNS）
apiVersion: v1
kind: Service
metadata:
  name: gim-ws-headless
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-ws
spec:
  clusterIP: None
  selector:
    app: gim-ws
  ports:
  - port: {{ .Values.ws.service.wsPort }}
    name: ws
  - port: {{ .Values.ws.service.grpcPort }}
    name: grpc
---
# NodePort Service（外部访问 WS）
apiVersion: v1
kind: Service
metadata:
  name: gim-ws
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-ws
spec:
  type: {{ .Values.ws.service.type }}
  selector:
    app: gim-ws
  ports:
  - port: {{ .Values.ws.service.wsPort }}
    targetPort: {{ .Values.ws.service.wsPort }}
    nodePort: {{ .Values.ws.service.nodePortWs }}
    name: ws
  - port: {{ .Values.ws.service.grpcPort }}
    targetPort: {{ .Values.ws.service.grpcPort }}
    name: grpc
{{- end }}
```

### 8.9 RPC 服务 Deployment 模板

其余 RPC 服务（auth/user/msg）结构相同，只是名字和端口不同。以 rpc-msg 为例：

```yaml
# deploy/k8s/helm/gim/templates/rpc-msg-deployment.yaml
{{- if .Values.rpcMsg.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gim-rpc-msg
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-rpc-msg
    {{- include "gim.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.rpcMsg.replicas }}
  selector:
    matchLabels:
      app: gim-rpc-msg
  template:
    metadata:
      labels:
        app: gim-rpc-msg
    spec:
      containers:
      - name: gim-rpc-msg
        image: "{{ .Values.global.imageRegistry }}{{ .Values.rpcMsg.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
        - containerPort: {{ .Values.rpcMsg.service.port }}
          name: grpc
        env:
          {{- include "gim.commonEnv" . | nindent 10 }}
          - name: GIM_ETCD_ENDPOINTS
            value: {{ .Values.etcd.endpoints }}
        envFrom:
        - configMapRef:
            name: gim-config
        volumeMounts:
        - name: config
          mountPath: /etc/gim/configs
        - name: secrets
          mountPath: /etc/gim/secrets
          readOnly: true
        resources:
          {{- toYaml .Values.rpcMsg.resources | nindent 10 }}
      volumes:
      - name: config
        configMap:
          name: gim-config
      - name: secrets
        secret:
          secretName: gim-secret
{{- end }}
```

```yaml
# deploy/k8s/helm/gim/templates/rpc-msg-service.yaml
{{- if .Values.rpcMsg.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: gim-rpc-msg
  namespace: {{ .Values.global.namespace }}
  labels:
    app: gim-rpc-msg
spec:
  type: ClusterIP
  selector:
    app: gim-rpc-msg
  ports:
  - port: {{ .Values.rpcMsg.service.port }}
    targetPort: {{ .Values.rpcMsg.service.port }}
    name: grpc
{{- end }}
```

### 8.10 HPA

```yaml
# deploy/k8s/helm/gim/templates/hpa.yaml
{{- if .Values.hpa.enabled }}
# WS Gateway HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gim-ws-hpa
  namespace: {{ .Values.global.namespace }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: gim-ws
  minReplicas: {{ .Values.hpa.ws.minReplicas }}
  maxReplicas: {{ .Values.hpa.ws.maxReplicas }}
  metrics:
  - type: Pods
    pods:
      metric:
        name: gim_ws_connections
      target:
        type: AverageValue
        averageValue: {{ .Values.hpa.ws.targetConnections | quote }}
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: {{ .Values.hpa.ws.targetCPU }}
---
# API HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gim-api-hpa
  namespace: {{ .Values.global.namespace }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gim-api
  minReplicas: {{ .Values.hpa.api.minReplicas }}
  maxReplicas: {{ .Values.hpa.api.maxReplicas }}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: {{ .Values.hpa.api.targetCPU }}
{{- end }}
```

---

## 9. Ingress 与外部访问

### 9.1 安装 Nginx Ingress Controller

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.kind=DaemonSet \
  --set controller.service.type=NodePort \
  --set controller.service.nodePorts.http=30080 \
  --set controller.service.nodePorts.https=30443 \
  --set tcp.10001=gim/gim-ws:10001
```

> `tcp.10001=gim/gim-ws:10001` 让 Ingress Controller 在 10001 端口做 TCP 透传，用于 WebSocket 连接。

### 9.2 Ingress 资源

```yaml
# deploy/k8s/helm/gim/templates/ingress.yaml
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gim-ingress
  namespace: {{ .Values.global.namespace }}
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  ingressClassName: {{ .Values.ingress.className }}
  rules:
  - host: {{ .Values.ingress.host }}
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: gim-api
            port:
              number: {{ .Values.api.service.port }}
      - path: /admin
        pathType: Prefix
        backend:
          service:
            name: gim-admin
            port:
              number: {{ .Values.admin.service.port }}
{{- end }}
```

### 9.3 访问方式

部署完成后，通过以下方式访问：

```bash
# HTTP API（通过 Ingress NodePort）
curl http://k8s-master:30080/api/v1/auth/login

# WebSocket（通过 Ingress TCP 透传）
wscat -c ws://k8s-master:10001/ws?token=xxx

# 或直接通过 WS Service NodePort
wscat -c ws://k8s-master:30001/ws?token=xxx
```

如果配置了域名和 DNS：

```bash
# HTTP API
curl http://gim.example.com/api/v1/auth/login

# WebSocket
wscat -c ws://ws.gim.example.com/ws?token=xxx
```

---

## 10. 监控部署

### 10.1 Prometheus + Grafana

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.storageClassName=nfs \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.resources.requests.storage=10Gi \
  --set grafana.persistence.enabled=true \
  --set grafana.persistence.storageClassName=nfs \
  --set grafana.persistence.size=5Gi \
  --set grafana.service.type=NodePort \
  --set grafana.service.nodePort=30030
```

访问 Grafana：

```bash
# 获取 admin 密码
kubectl get secret kube-prometheus-grafana -n monitoring -o jsonpath='{.data.admin-password}' | base64 -d

# 浏览器访问
# http://k8s-master:30030
# 用户名: admin  密码: (上面获取的)
```

### 10.2 gim 指标接入

在 gim 服务的 Pod 中暴露 Prometheus 指标端口，然后创建 ServiceMonitor：

```yaml
# deploy/k8s/helm/gim/templates/servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: gim-metrics
  namespace: {{ .Values.global.namespace }}
  labels:
    release: kube-prometheus
spec:
  selector:
    matchLabels:
      app: gim-api
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: gim-ws-metrics
  namespace: {{ .Values.global.namespace }}
  labels:
    release: kube-prometheus
spec:
  selector:
    matchLabels:
      app: gim-ws
  endpoints:
  - port: ws
    path: /metrics
    interval: 15s
```

---

## 11. 日常运维操作

### 11.1 构建并推送镜像

在开发机上构建 gim 各服务镜像并推送到镜像仓库：

```bash
# 设置镜像仓库地址（假设用 Docker Hub 或私有仓库）
export REGISTRY=yourregistry.com/gim

# 构建各服务镜像
docker build -f deploy/docker/Dockerfile.api -t ${REGISTRY}/api:latest .
docker build -f deploy/docker/Dockerfile.ws -t ${REGISTRY}/ws:latest .
docker build -f deploy/docker/Dockerfile.rpc-auth -t ${REGISTRY}/rpc-auth:latest .
docker build -f deploy/docker/Dockerfile.rpc-user -t ${REGISTRY}/rpc-user:latest .
docker build -f deploy/docker/Dockerfile.rpc-msg -t ${REGISTRY}/rpc-msg:latest .
docker build -f deploy/docker/Dockerfile.push -t ${REGISTRY}/push:latest .
docker build -f deploy/docker/Dockerfile.msgtransfer -t ${REGISTRY}/msgtransfer:latest .
docker build -f deploy/docker/Dockerfile.admin -t ${REGISTRY}/admin:latest .
docker build -f deploy/docker/Dockerfile.ai -t ${REGISTRY}/ai:latest .

# 推送
docker push ${REGISTRY}/api:latest
docker push ${REGISTRY}/ws:latest
docker push ${REGISTRY}/rpc-auth:latest
docker push ${REGISTRY}/rpc-user:latest
docker push ${REGISTRY}/rpc-msg:latest
docker push ${REGISTRY}/push:latest
docker push ${REGISTRY}/msgtransfer:latest
docker push ${REGISTRY}/admin:latest
docker push ${REGISTRY}/ai:latest
```

> 如果没有私有镜像仓库，可以用 Docker Hub（`docker login` 后推送）或搭建 Harbor。

### 11.2 使用 Helm 部署/升级

```bash
# 首次部署
cd deploy/k8s/helm
helm install gim ./gim \
  --namespace gim \
  --set global.imageRegistry=yourregistry.com/ \
  --set global.storageClass=nfs

# 查看部署状态
helm status gim -n gim
kubectl get pods -n gim -w

# 升级（更新镜像后）
helm upgrade gim ./gim \
  --namespace gim \
  --set global.imageRegistry=yourregistry.com/ \
  --set image.tag=v1.1.0

# 回滚
helm rollback gim 1 -n gim    # 回滚到第 1 个版本

# 卸载
helm uninstall gim -n gim
```

### 11.3 查看服务日志

```bash
# 查看 API 网关日志
kubectl logs -f deployment/gim-api -n gim

# 查看 WS Gateway 日志（StatefulSet）
kubectl logs -f gim-ws-0 -n gim

# 查看某个 Pod 最近 200 行日志
kubectl logs gim-ws-0 -n gim --tail=200

# 查看所有 Pod 的日志
kubectl logs -l app=gim-api -n gim --all-containers=true
```

### 11.4 扩缩容

```bash
# 手动扩容 API 网关到 3 个副本
kubectl scale deployment gim-api --replicas=3 -n gim

# 手动扩容 WS Gateway
kubectl scale statefulset gim-ws --replicas=3 -n gim

# 查看 HPA 状态（如果启用了自动扩容）
kubectl get hpa -n gim
```

### 11.5 数据库迁移

```bash
# 创建迁移 Job
kubectl create job --from=cronjob/gim-migrate db-migrate-$(date +%s) -n gim

# 或者用临时 Pod 执行迁移
kubectl run migrate --rm -it --restart=Never \
  --image=yourregistry.com/gim/api:latest \
  -n gim -- ./gim migrate up
```

### 11.6 进入容器调试

```bash
# 进入 API Pod
kubectl exec -it deployment/gim-api -n gim -- bash

# 进入 MySQL
kubectl exec -it mysql-0 -n gim -- mysql -ugim -pgim_pass gim

# 进入 Redis
kubectl exec -it redis-master-0 -n gim -- redis-cli -a redis_pass123
```

### 11.7 端口转发（本地调试）

```bash
# 将 MySQL 端口转发到本地
kubectl port-forward svc/mysql 3306:3306 -n gim

# 将 Redis 端口转发到本地
kubectl port-forward svc/redis-master 6379:6379 -n gim

# 将 API 转发到本地
kubectl port-forward svc/gim-api 8080:10002 -n gim
```

---

## 12. 故障排查

### 12.1 Pod 一直 Pending

```bash
kubectl describe pod <pod-name> -n gim
# 看 Events 部分，常见原因：
# - Insufficient cpu/memory：节点资源不够
# - PersistentVolumeClaim pending：没有可用的 PV（检查 NFS Provisioner）
# - Node didn't have free resource：Pod 调度不上去
```

### 12.2 Pod CrashLoopBackOff

```bash
# 查看退出日志
kubectl logs <pod-name> -n gim --previous

# 常见原因：
# - 配置错误（数据库连不上、配置文件缺失）
# - 健康检查失败（livenessProbe 连续失败会杀 Pod）
# - 依赖服务未就绪（API 启动时 MySQL 还没好）
```

### 12.3 Service 无法访问

```bash
# 检查 Service 是否有 Endpoints
kubectl get endpoints gim-api -n gim
# 如果 ENDPOINTS 为 <none>，说明没有 Pod 匹配 Service 的 selector

# 检查 Service selector 和 Pod labels 是否匹配
kubectl get pod -n gim --show-labels
kubectl describe svc gim-api -n gim | grep Selector
```

### 12.4 PVC 挂载失败

```bash
# 查看 PVC 状态
kubectl get pvc -n gim

# 查看 NFS Provisioner 日志
kubectl logs -n kube-system -l app=nfs-subdir-external-provisioner

# 常见原因：
# - NFS Server 不可达
# - /etc/exports 配置错误
# - NFS Provisioner 未安装或崩溃
```

### 12.5 WS 连接断开

```bash
# 查看 WS Gateway 日志
kubectl logs gim-ws-0 -n gim --tail=200 | grep -i "disconnect\|error\|close"

# 常见原因：
# - 心跳超时（检查 pongWait 配置）
# - Ingress 超时（增加 proxy-read-timeout）
# - 网络策略阻止（检查 NetworkPolicy）
# - Pod 被 OOMKilled（增加 memory limit）
```

### 12.6 镜像拉取失败

```bash
kubectl describe pod <pod-name> -n gim | grep -A5 "Events"

# 常见原因：
# - ImagePullBackOff：镜像地址错误或仓库需要认证
# - 解决：创建 imagePullSecret
kubectl create secret docker-registry regcred \
  --docker-server=yourregistry.com \
  --docker-username=tianlu1990s \
  --docker-password=yourpass \
  -n gim

# 在 values.yaml 中引用
# global:
#   imagePullSecrets:
#     - regcred
```

### 12.7 常用排错命令速查

```bash
# Pod 级别
kubectl get pods -n gim -o wide             # 查看 Pod 所在节点和 IP
kubectl describe pod <name> -n gim          # Pod 详情和事件
kubectl logs <name> -n gim --previous       # 上次崩溃的日志
kubectl top pods -n gim                     # Pod 资源使用

# 节点级别
kubectl describe node k8s-worker1           # 节点资源和 Pod 分配
kubectl top nodes                           # 节点资源使用

# 网络级别
kubectl get svc -n gim                      # Service 列表
kubectl get endpoints -n gim                # Endpoints 列表
kubectl run test --rm -it --image=busybox -n gim -- wget -qO- http://gim-api:10002/healthz  # 测试内部连通性

# 存储级别
kubectl get pv                              # 持久卷列表
kubectl get pvc -n gim                      # 持久卷声明列表
```

---

## 附录 A：一键部署脚本

```bash
#!/bin/bash
# scripts/deploy_k8s.sh — 一键部署 gim 到 K8S

set -e

echo "=== 1. 创建命名空间 ==="
kubectl create namespace gim 2>/dev/null || true

echo "=== 2. 部署基础设施 ==="
# MySQL
helm upgrade --install mysql bitnami/mysql --namespace gim \
  --set auth.rootPassword=root123456 \
  --set auth.database=gim \
  --set auth.username=gim \
  --set auth.password=gim_pass \
  --set primary.persistence.size=20Gi \
  --set primary.persistence.storageClass=nfs \
  --set architecture=standalone \
  --set image.tag=8.0

echo "等待 MySQL 就绪..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mysql -n gim --timeout=120s

# Redis
helm upgrade --install redis bitnami/redis --namespace gim \
  --set auth.password=redis_pass123 \
  --set master.persistence.size=5Gi \
  --set master.persistence.storageClass=nfs \
  --set architecture=standalone

echo "等待 Redis 就绪..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=redis -n gim --timeout=120s

# MongoDB (第二阶段)
helm upgrade --install mongodb bitnami/mongodb --namespace gim \
  --set auth.rootPassword=mongo_pass123 \
  --set auth.username=gim \
  --set auth.password=gim_mongo_pass \
  --set auth.database=gim \
  --set persistence.size=20Gi \
  --set persistence.storageClass=nfs \
  --set architecture=standalone

echo "等待 MongoDB 就绪..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mongodb -n gim --timeout=120s

# Kafka (第二阶段)
helm upgrade --install kafka bitnami/kafka --namespace gim \
  --set controller.replicaCount=1 \
  --set broker.replicaCount=2 \
  --set persistence.size=10Gi \
  --set persistence.storageClass=nfs

echo "等待 Kafka 就绪..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka -n gim --timeout=180s

# etcd (第二阶段)
helm upgrade --install etcd bitnami/etcd --namespace gim \
  --set auth.rbac.create=false \
  --set replicaCount=3 \
  --set persistence.size=5Gi \
  --set persistence.storageClass=nfs

# MinIO (第二阶段)
helm upgrade --install minio bitnami/minio --namespace gim \
  --set auth.rootUser=admin \
  --set auth.rootPassword=minio_pass123 \
  --set mode=standalone \
  --set persistence.size=50Gi \
  --set persistence.storageClass=nfs

echo "=== 3. 部署 gim 服务 ==="
cd deploy/k8s/helm
helm upgrade --install gim ./gim --namespace gim \
  --set global.imageRegistry=${REGISTRY:-} \
  --set global.storageClass=nfs

echo "=== 4. 等待服务就绪 ==="
kubectl rollout status deployment/gim-api -n gim --timeout=120s
kubectl rollout status statefulset/gim-ws -n gim --timeout=120s

echo "=== 部署完成 ==="
echo "API:  http://k8s-master:30080/api/v1"
echo "WS:   ws://k8s-master:30001/ws"
echo "Grafana: http://k8s-master:30030"
kubectl get pods -n gim
```

---

## 附录 B：Dockerfile 模板

每个微服务需要独立的 Dockerfile。以下是模板：

```dockerfile
# deploy/docker/Dockerfile.api
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gim-api cmd/gim-api/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/gim-api /usr/local/bin/gim-api
COPY --from=builder /app/configs /etc/gim/configs
EXPOSE 10002
ENTRYPOINT ["gim-api"]
```

```dockerfile
# deploy/docker/Dockerfile.ws
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gim-ws cmd/gim-ws/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/gim-ws /usr/local/bin/gim-ws
COPY --from=builder /app/configs /etc/gim/configs
EXPOSE 10001 10140
ENTRYPOINT ["gim-ws"]
```

其他服务（rpc-auth、rpc-user、rpc-msg、push、msgtransfer、admin、ai）同理，只需改 `cmd/` 路径和 `EXPOSE` 端口。
