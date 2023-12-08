# BackUpTool 

## Using for backup files to BaiduNetDisk


# How To Run?
0. Require `Redis`, Please Install it first.
1. Get the `config.template.yaml` and rename to `config.yaml`
2. Download Release File and copy `config.yaml` and `Backuptool` into same folder
3. Run backuptool such as
```shell
Usage of BackUpTool:
  -auth
        Is Open Auth Mode(default false)
  -config string
        config file path (default "./config.yaml")
  -sync
        Is Sync Mode(default false)
```

# How to Develop?
```shell
mv config.template.yaml config.yaml
go mod tidy
```

## ToDoList
- [x] Support Download File API
- [x] Support Chunk Upload API
- [x] Sync Serivce(bidirectional)
- [x] Multi thread upload
- [ ] Rewrite the same file check algorithm
- [ ] Resumable transfer
- [ ] Error Handler, import reliability.


https://pan.baidu.com/union/doc/Zl0jb6i29

通过接口获取到的md5值和通过文件下载下来的md5值不一致
您好，这是已有的产品策略。想要校验的话，可以把云端的文件下载下来，在本地再重新计算下md5

