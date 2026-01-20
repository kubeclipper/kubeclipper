# Issue & PR 管理

## 1. issue 和 pr 使用 label 进行管理

Issue 和 PR 按照目的可以给出以下这些 label

- `kind/design`: Issue 或 PR 描述/实现了一个设计类的改动
- `kind/cleanup`: Issue 或 PR 涉及代码清理、流程优化或技术债务
- `kind/bug`: Issue 或 PR 涉及 bug 修复
- `kind/api-change`: Issue 或 PR 涉及 API 的添加、删除或修改
- `kind/workflow`: Issue 或 PR 涉及 GitHub workflow 相关改动
- `kind/feature`: Issue 或 PR 涉及新功能开发
- `kind/support`: Issue 涉及技术支持请求
- `kind/proposal`: Issue 涉及新提案
- `kind/documentation`: Issue 或 PR 涉及文档相关改动
- `kind/flake`: Issue 或 PR 涉及不稳定测试用例
- `kind/optimize`: Issue 或 PR 涉及性能优化
- `kind/console`: Issue 或 PR 涉及控制台相关改动
- `kind/feature-request`: Issue 或 PR 涉及新功能请求
- `kind/failing-test`: Issue 或 PR 涉及持续失败的测试用例
- `kind/need-verify`: Issue 或 PR 需要验证

注意事项：

1. PR 最好至少关联一个 Issue
2. PR 所有的 GitHub Action Workflow 需要都运行成功才能合并
3. merge 代码时，使用 `Squash and merge` 选项
