# haojianhao

这里新增了一个「标准体围统计 + 个人测量打分」的小工具（Go）。

## 生成标准体围（P30 中位数 + P40/P55 标准范围）

输入：MLD VAPro-5 数据库导出的 CSV（支持有表头自动识别；识别失败则回退到固定列索引）

输出：`standard_body_metrics.csv`

```bash
go run ./cmd/standard-body-metrics -in data.csv -out standard_body_metrics.csv -header=true
```

规则（按需求实现）：

- 男：63-99 岁统一用 63-99 合并数据源计算（但仍输出 63..99 每岁一行，统计值相同）
- 女：70-99 岁统一用 70-99 合并数据源计算（同上）
- 其他：同年龄同性交叉分组
- “中位数”：取目标数据源升序后 **30% 分位点（P30）**，四舍五入 1 位小数
- 标准范围：取升序后 **40%（a）** 与 **55%（b）** 分位点，四舍五入 1 位小数

## 个人测量值打分（新增“超越人群比例”字段）

输入：个人测量 CSV + 上一步生成的标准体围 CSV

输出：`scored.csv`（每个维度都会输出 measured/median(P30)/surpass_diff/surpass_ratio_percent/a(P40)/b(P55)/level）

```bash
go run ./cmd/score-metrics -standards standard_body_metrics.csv -in measurements.csv -out scored.csv -header=true
```

其中：

- `*_surpass_diff = measured - median(P30)`（“超越人群”差值）
- `*_surpass_ratio_percent = (measured-median)/median * 100`（“超越人群比例”百分比，保留 1 位小数）

