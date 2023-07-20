# Kubeclipper 接口自动化测试

## 1. 准备

```bash
pip3 install gabbi
```

## 2. 运行

```bash
#run with list.yaml
cd test/api
python3 run.py

#run all
find test/api -name '*.yaml' | xargs gabbi-run 139.196.13.9 --
find test/api -name '*.yaml' | xargs gabbi-run localhost --

# show verbose
gabbi-run -v all 139.196.13.9 -- test/api/*.yaml
gabbi-run -v all localhost -- test/api/*.yaml
```

## 3. 覆盖率

### 3.1 覆盖范围

1.  创建集群

-   [创建带存储插件 nfs 的集群](./add_remove_node/nfs_cluster.yaml)
-   [创建单机集群](./add_remove_node/one_two_nodes_cluster.yaml)
-   [创建高可用集群](./create_high_availability_cluster.yaml)
-   [创建同名集群](./add_remove_node/one_two_nodes_cluster.yaml)

2.  [查看集群列表](./add_remove_node/nfs_cluster.yaml)
3.  [集群添加节点和移除节点](./add_remove_node/)
4.  [集群备份和恢复](./backup_recovery_cluster.yaml)
5.  [定时备份](./backup_recovery_cluster.yaml)
6.  [集群模版&插件模版](./template)(共 5 个 API)
