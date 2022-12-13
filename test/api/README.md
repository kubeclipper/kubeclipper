# Kubeclipper 接口自动化测试

## 1. 准备

```bash
pip3 install gabbi
```

## 2. 运行

```bash
ls test/api/*.yaml | xargs gabbi-run 139.196.13.9:28300 --
ls test/api/*.yaml | xargs gabbi-run localhost:28300 --

# show verbose
gabbi-run -v all 139.196.13.9:28300 -- test/api/*.yaml
gabbi-run -v all localhost:28300 -- test/api/*.yaml
```

## 3. 覆盖率

### 3.1 覆盖范围

[创建集群](./create_get_cluster.yaml)
[查看集群列表](./create_get_cluster.yaml)
