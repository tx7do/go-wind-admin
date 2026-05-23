# 脚本工作流和最佳实践指南

## 📚 脚本工作流详解

### 工作流 1：完整本地开发环境设置（推荐初次使用）

#### 第一次使用（Linux/macOS）

```bash
# 步骤 1：准备开发环境（只需一次）
# 安装所有必要工具：基础工具、Node.js、Docker、Go、开发插件
./scripts/env/install_unix_dev.sh

# 步骤 2：验证安装
go version
docker --version
npm -v

# 步骤 3：启动完整环境
./scripts/docker/full_deploy.sh

# 步骤 4：开发
gow run admin
```

#### 后续开发（每天）

```bash
# 方式 A：启动所有服务（如果需要完整环境）
./scripts/docker/full_deploy.sh
gow run admin

# 方式 B：启动仅依赖（推荐本地开发）
./scripts/docker/libs_only.sh
gow run admin
```

---

### 工作流 2：本地开发 + 远程依赖

**场景：** 仅在本地开发，数据库等依赖在远程服务器

```bash
# 步骤 1：准备开发环境
./scripts/env/install_unix_dev.sh

# 步骤 2：仅启动本地依赖
./scripts/docker/libs_only.sh

# 步骤 3：在 IDE 中开发（不需要通过命令行运行）
# 在 VS Code / GoLand 中打开项目，直接按 F5 调试
```

**优点：**

- ✅ 调试体验更好
- ✅ 快速迭代（改代码立即生效）
- ✅ 节省系统资源

---

### 工作流 3：生产环境部署

```bash
# 步骤 1：登录服务器
ssh user@server.com

# 步骤 2：克隆项目
git clone https://github.com/tx7do/go-wind-admin.git
cd go-wind-admin/backend

# 步骤 3：准备生产环境
./scripts/env/install_unix_prod.sh

# 步骤 4：启动完整应用
./scripts/docker/full_deploy.sh

# 步骤 5：通过 PM2 管理应用
./scripts/deploy/pm2_service.sh

# 步骤 6：查看日志
pm2 logs
```

---

### 工作流 4：Windows 开发

#### 第一次使用（Windows）

```powershell
# 步骤 1：以管理员身份打开 PowerShell

# 步骤 2：准备开发环境
.\scripts\env\install_windows_dev.ps1

# 步骤 3：重启 PowerShell（加载环境变量）

# 步骤 4：启动依赖（推荐 - 使用 PowerShell 脚本）
.\scripts\docker\libs_only.ps1

# 步骤 5：在 IDE 中开发（VS Code / GoLand）
# 打开项目，按 F5 开始调试
```

#### 后续开发（每天）

```powershell
# 方式 A：启动仅依赖（推荐本地开发）
.\scripts\docker\libs_only.ps1
# 然后在 IDE 中按 F5 调试

# 方式 B：启动完整应用
.\scripts\docker\full_deploy.ps1

# 方式 C：自定义数据目录
.\scripts\docker\libs_only.ps1 -AppRoot "D:\app"
```

**Windows 特有功能：**

- ✅ 原生 PowerShell 脚本支持
- ✅ 完整的参数化控制
- ✅ 彩色化日志输出
- ✅ 内置帮助系统（Get-Help）

---

### 工作流 5：多人协作开发

**场景：** 团队开发，需要统一的依赖环境

```bash
# 项目负责人
git clone https://github.com/tx7do/go-wind-admin.git
cd go-wind-admin/backend
./scripts/env/install_unix_dev.sh
./scripts/docker/full_deploy.sh
git push  # 确保项目配置已推送

# 其他开发者
git clone https://github.com/tx7do/go-wind-admin.git
cd go-wind-admin/backend
./scripts/env/install_unix_dev.sh  # 一次性
./scripts/docker/libs_only.sh      # 每天开发时
gow run admin
```

---

## 🎯 脚本选择指南

### 环境准备脚本选择

```
我在哪里开发？
├─ macOS → install_unix_dev.sh
├─ Ubuntu/Debian → install_unix_dev.sh
├─ CentOS/RHEL → install_unix_dev.sh
├─ Rocky/AlmaLinux → install_unix_dev.sh
├─ Fedora → install_unix_dev.sh
└─ Windows → install_windows_dev.ps1

我在准备什么环境？
├─ 本地开发 → install_unix_dev.sh（包含所有工具）
└─ 生产服务器 → install_unix_prod.sh（精简版）
```

### Docker 启动脚本选择

```
我需要什么？
├─ 完整应用 + 所有依赖
│  ├─ Linux/macOS → ./scripts/docker/full_deploy.sh
│  └─ Windows → .\scripts\docker\full_deploy.ps1
│
├─ 仅依赖（本地开发）
│  ├─ Linux/macOS → ./scripts/docker/libs_only.sh
│  └─ Windows → .\scripts\docker\libs_only.ps1
│
└─ 自定义配置
   ├─ Bash → APP_ROOT=/path ./scripts/docker/libs_only.sh
   └─ PowerShell → .\scripts\docker\libs_only.ps1 -AppRoot "D:\path"
```

**脚本版本选择：**

- **Linux/macOS** - 使用 `.sh` Bash 脚本
- **Windows** - 优先使用 `.ps1` PowerShell 脚本
- **跨平台** - Windows 也可以使用 Git Bash 运行 `.sh` 脚本

---

## 💡 最佳实践

### 1. 环境隔离

**推荐做法：**

*Linux/macOS (Bash):*

```bash
# 为每个项目/环境创建独立的数据目录
APP_ROOT=/opt/project1 ./scripts/docker/libs_only.sh
APP_ROOT=/opt/project2 ./scripts/docker/libs_only.sh
```

*Windows (PowerShell):*

```powershell
# 为每个项目/环境创建独立的数据目录
.\scripts\docker\libs_only.ps1 -AppRoot "D:\project1"
.\scripts\docker\libs_only.ps1 -AppRoot "D:\project2"
```

**优点：** 避免数据混污，易于切换

---

### 2. 版本管理

**推荐做法：**

```bash
# 定期更新工具
./scripts/env/install_unix_dev.sh  # 重新运行可更新工具

# 检查版本
go version
docker --version
npm -v
```

---

### 3. 日志管理

**推荐做法：**

```bash
# 查看应用日志
docker logs -f <container-name>

# 查看 PM2 日志
pm2 logs <app-name>

# 查看系统日志
journalctl -u docker -f
```

---

### 4. 性能优化

**推荐做法：**

```bash
# 限制 Docker 资源（可选）
docker run -m 2G --cpus 2 ...

# 使用本地开发模式（不启动应用容器）
bash scripts/docker/libs_only.sh
# 在 IDE 中直接运行应用
```

---

### 5. 数据备份

**推荐做法：**

```bash
# 备份数据库
docker exec postgres pg_dump > backup.sql

# 备份数据卷
tar -czf app_data.tar.gz /root/app/

# 恢复数据
docker exec -i postgres psql < backup.sql
```

---

## 🔄 常见场景处理

### 场景 1：更新项目代码后重启

*Linux/macOS (Bash):*

```bash
# 方式 1：最简单（推荐）
./scripts/docker/libs_only.sh  # 重启依赖
# 在 IDE 中重启应用

# 方式 2：完整重启
./scripts/docker/full_deploy.sh  # 重启所有
gow run admin

# 方式 3：PM2 重启
pm2 restart all
```

*Windows (PowerShell):*

```powershell
# 方式 1：最简单（推荐）
.\scripts\docker\libs_only.ps1  # 重启依赖
# 在 IDE 中重启应用

# 方式 2：完整重启
.\scripts\docker\full_deploy.ps1  # 重启所有
gow run admin

# 方式 3：PM2 重启（如已安装）
pm2 restart all
```

### 场景 2：切换不同的 Go 版本

```bash
# 编辑 install_unix_dev.sh 中的 Go 版本
# 然后重新运行
./scripts/env/install_unix_dev.sh
```

### 场景 3：添加新的 Go 依赖

```bash
# 编辑 app/go.mod
# 运行
go mod tidy
gow run admin
```

### 场景 4：Docker 卷满了

```bash
# 清理未使用的资源
docker system prune -a
docker volume prune

# 查看磁盘使用
docker system df
```

### 场景 5：端口被占用

*Linux/macOS:*

```bash
# 查看占用的端口
lsof -i :8080

# 杀死进程
kill -9 <PID>

# 更改 docker-compose 端口配置
# 编辑 docker-compose.yml 后重启
./scripts/docker/libs_only.sh
```

*Windows:*

```powershell
# 查看占用的端口
netstat -ano | findstr :8080

# 杀死进程
taskkill /F /PID <PID>

# 或使用 Stop-Process
Stop-Process -Id <PID> -Force

# 更改 docker-compose 端口配置后重启
.\scripts\docker\libs_only.ps1
```

---

## 📊 脚本执行时间参考

| 脚本                   | 首次执行     | 后续执行   | 说明       |
|----------------------|----------|--------|----------|
| install_unix_dev.sh  | 10-30 分钟 | 2-5 分钟 | 取决于网络    |
| install_unix_prod.sh | 5-15 分钟  | 1-3 分钟 | 精简版本     |
| full_deploy.sh       | 2-5 分钟   | 2-5 分钟 | 首次下载镜像较慢 |
| libs_only.sh         | 1-3 分钟   | 1-3 分钟 | 通常很快     |
| pm2_service.sh       | 1-2 分钟   | 1-2 分钟 | 取决于构建    |

---

## ✅ 验证检查清单

在开始开发前，确保以下项目都已完成：

### 环境检查

- ✅ Go 已安装：`go version`
- ✅ Docker 已安装：`docker --version`
- ✅ npm 已安装：`npm -v`
- ✅ Git 已安装：`git --version`

### 依赖检查

- ✅ 数据库容器运行：`docker ps | grep postgres`
- ✅ Redis 容器运行：`docker ps | grep redis`

### 应用检查

- ✅ 应用可启动：`go run main.go`
- ✅ 应用监听端口：`lsof -i :8080` (Linux/macOS) 或 `netstat -ano | findstr :8080` (Windows)
- ✅ 应用连接数据库：查看日志

---

## 🆘 常见问题排除

### 问题：脚本权限被拒绝

*Linux/macOS:*

```bash
# 解决
chmod +x scripts/**/*.sh
./scripts/env/install_unix_dev.sh
```

*Windows (PowerShell):*

```powershell
# 解决 - 修改执行策略
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# 验证
Get-ExecutionPolicy
```

---

### 问题：Docker 权限错误

*Linux/macOS:*

```bash
# 解决
sudo usermod -aG docker $USER
newgrp docker
# 或重启终端
```

*Windows:*

```powershell
# 确保 Docker Desktop 正在运行
Get-Service -Name com.docker.service

# 重启 Docker Desktop
Restart-Service com.docker.service
```

---

### 问题：Go 找不到

*Linux/macOS:*

```bash
# 解决
source ~/.bashrc  # 或 ~/.zshrc
go version
```

*Windows:*

```powershell
# 解决 - 刷新环境变量
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

# 或重启 PowerShell
go version
```

---

### 问题：端口已被占用

*Linux/macOS:*

```bash
# 解决
lsof -i :8080  # 找到进程
kill -9 <PID>
```

*Windows:*

```powershell
# 解决
netstat -ano | findstr :8080  # 找到进程
taskkill /F /PID <PID>
```

---

### 问题：Docker 镜像找不到

```bash
# 解决（通用）
docker pull postgres  # 手动拉取
docker pull redis
# 然后重新运行脚本
```

---

### 问题：PowerShell 脚本找不到（Windows）

```powershell
# 确保在项目根目录
Get-Location

# 检查脚本是否存在
Test-Path .\scripts\docker\libs_only.ps1

# 使用绝对路径
& "C:\path\to\project\scripts\docker\libs_only.ps1"
```

---

## 📞 技术支持

遇到问题时的排查步骤：

1. **查看脚本输出** - 脚本有详细的日志输出
2. **查看容器日志** - `docker logs <container>`
3. **查看应用日志** - 应用输出信息
4. **重新运行脚本** - 脚本具有幂等性
5. **查看文档** - 参考 README.md

---
