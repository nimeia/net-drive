# Iter 35 — WinFsp native callback 实装矩阵最后收口

把剩余写侧/元数据侧 WinFsp native callback 从“表里声明为 read-only”推进到“callbacks → dispatcher bridge → ABI → service warmup 全链路显式只读拒绝”。
