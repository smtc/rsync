# 差异同步

如果两个文件，src和dst，如果在同一台机器上，我们想让dst的内容完全与src相同，可以使用cp来覆盖：

cp src dst

由于在同一台机器上，而且通常来说文件不会很大，这样做在99%的情况下没有问题；

同理，如果src和dst在两台机器上，我们可以使用scp来做：

scp src dst

那么，如果src文件很大，而且dst原来是和src相同的，src现在只是追加(或删除，或修改)了一些数据，有没
有更好更快的方法，让dst与src保持一致呢？

有，答案是rsync同步算法。

工作步骤：
1 对dst文件做signature，得到dst.sig文件；
2 使用dst.sig文件与src做delta，得到src-dst.delta文件；
3 使用src-dst.delta文件对dst做patch，使dst与src文件一致；


# delta算法

# delta文件格式
序号 名称            字节   说明
1   delta magic     4     无
2   匹配或不匹配block 变长   无

## 匹配block格式  
序号 名称        字节               说明
1   cmd         1                根据匹配位置where_byte和匹配长度来定义
2   匹配位置     变长，
               可能取值：1,2,4,8 
3   匹配长度     变长              
               可能取值：1,2,4,8 

对应值如下：
       RS_OP_COPY_N1_N1 = 0x45,
       RS_OP_COPY_N1_N2 = 0x46,
       RS_OP_COPY_N1_N4 = 0x47,
       RS_OP_COPY_N1_N8 = 0x48,
       RS_OP_COPY_N2_N1 = 0x49,
       RS_OP_COPY_N2_N2 = 0x4a,
       RS_OP_COPY_N2_N4 = 0x4b,
       RS_OP_COPY_N2_N8 = 0x4c,
       RS_OP_COPY_N4_N1 = 0x4d,
       RS_OP_COPY_N4_N2 = 0x4e,
       RS_OP_COPY_N4_N4 = 0x4f,
       RS_OP_COPY_N4_N8 = 0x50,
       RS_OP_COPY_N8_N1 = 0x51,
       RS_OP_COPY_N8_N2 = 0x52,
       RS_OP_COPY_N8_N4 = 0x53,
       RS_OP_COPY_N8_N8 = 0x54,

## 不匹配block格式
序号   名称         字节          说明
1     cmd          1字节         根据不匹配的长度决定
2     miss len     变长
      不匹配的长度   取值：1,2,4,8
3     压缩方式      1字节          后面的不匹配数据的压缩方式(librsync中没有此字段)

       RS_OP_LITERAL_N1 = 0x41,
       RS_OP_LITERAL_N2 = 0x42,
       RS_OP_LITERAL_N4 = 0x43,
       RS_OP_LITERAL_N8 = 0x44,
