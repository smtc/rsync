# rdiff用法

rdiff主要用来测试rsync库，及演示rsync的工作原理。

假设源文件为src.txt, 目标文件为dst.txt，使用如下的步骤后，生成中间文件
1 src.txt.sig, src.txt的签名文件
2 src-dst.delta, src与dst的差异文件

最终文件src.new.txt，src.new.txt与dst.txt完全相同。

## 生成signature文件

rdiff signature src.txt src.txt.sign

## 生成delta文件

rdiff delta src.txt.sign dst.txt src-dst.delta

## 使用delta文件patch 源文件，生成新的源文件，新的源文件与目标文件相同

rdiff patch src-dst.delta src.txt
