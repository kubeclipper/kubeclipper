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

[创建集群](./create_get_cluster.yaml)
[查看集群列表](./create_get_cluster.yaml)
[集群添加节点和移除节点](./add_remove_node/)
