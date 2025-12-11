# File Server with Token Authentication

一个支持大文件上传和下载的Go语言服务器程序，具有基于Token的身份验证机制。

## 功能特点

- 支持大文件上传和下载（支持200GB以上文件）
- 基于Token的身份验证，确保传输安全
- 可自定义上传文件存储目录
- 支持通过命令行参数配置服务器地址和端口
- 自动清理过期Token（1小时有效期）

## 编译

```bash
go build server-file.go
```

## 使用方法

### 启动服务器

```bash
./server-file --server 0.0.0.0 --port 8080 --upload-dir /path/to/upload/dir
```

参数说明：
- `--server`: 服务器绑定地址（默认: localhost）
- `--port`: 服务器端口（默认: 8080）
- `--upload-dir`: 上传文件存储目录（默认: uploads）

### 上传文件流程

1. 生成Token:
   ```bash
   curl -X POST http://localhost:8080/token
   ```
   返回示例: `a1b2c3d4e5f67890`

2. 使用Token上传文件:
   ```bash
   curl -X POST -F "file=@your-large-file.bin" http://localhost:8080/upload?token=a1b2c3d4e5f67890
   ```

### 下载文件流程

```bash
curl -X GET http://localhost:8080/download/your-large-file.bin?token=a1b2c3d4e5f67890 -o downloaded-file.bin
```

## systemd服务配置

项目包含一个systemd服务配置文件 [server-file.service](file:///Users/pangguanglong/vscode/src/long1318737396/mytest/server-file.service)，可用于将应用程序作为系统服务运行。

### 安装步骤

1. 编译程序:
   ```bash
   go build server-file.go
   ```

2. 将二进制文件复制到系统目录:
   ```bash
   sudo cp server-file /usr/local/bin/
   ```

3. 将service文件复制到systemd目录:
   ```bash
   sudo cp server-file.service /etc/systemd/system/
   ```

4. 创建运行用户和目录:
   ```bash
   sudo useradd -r -s /bin/false www-data
   sudo mkdir -p /var/lib/server-file/uploads
   sudo chown www-data:www-data /var/lib/server-file/uploads
   ```

5. 重新加载systemd并启用服务:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable server-file.service
   ```

### 管理服务

```bash
# 启动服务
sudo systemctl start server-file

# 停止服务
sudo systemctl stop server-file

# 重启服务
sudo systemctl restart server-file

# 查看服务状态
sudo systemctl status server-file

# 查看服务日志
sudo journalctl -u server-file -f
```

## API接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `/token` | POST | 生成新的访问Token |
| `/upload` | POST | 上传文件（需提供Token） |
| `/download/{filename}` | GET | 下载文件（需提供Token） |
| `/` | GET | 显示帮助页面 |

## 安全机制

- Token有效期为1小时
- Token只能使用一次（在有效期内）
- 上传和下载操作都需要提供有效的Token
- 防止路径遍历攻击

## 注意事项

- 上传的文件会被存储在指定的上传目录中
- 确保上传目录有足够的磁盘空间来存储大文件
- Token在生成后1小时自动过期
- 为了安全起见，请勿将Token泄露给他人

## 许可证

MIT License