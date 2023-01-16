# Kubeclipper 接口自动化测试

## 1. 准备

```bash
pip3 install gabbi
```

## 2. 运行

```bash
find test/api -name '*.yaml' | xargs gabbi-run 139.196.13.9 --
find test/api -name '*.yaml' | xargs gabbi-run localhost --

# show verbose
gabbi-run -v all 139.196.13.9 -- test/api/*.yaml
gabbi-run -v all localhost -- test/api/*.yaml
```

## 3. 覆盖率

### 3.1 覆盖范围


1.  [创建集群](./create_get_cluster.yaml)
2.  [查看集群列表](./create_get_cluster.yaml)
3.  [集群添加节点和移除节点](./add_remove_node/)
4.  [集群备份和恢复-存储类型为 fs](./fs_backup_recovery.yaml)
5.  [集群备份和恢复-存储类型为 s3](./s3_backup_recovery.yaml)
6.  [集群模版&插件模版](./template)(共9个API)
