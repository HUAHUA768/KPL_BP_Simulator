# Git 仓库推送实战指南

> 💡 **本文档专注于实际工作中真正会用到的推送方式**

---

## ✅ 当前状态

```bash
git log --oneline
```

```
e8d1415 Add detailed GitHub push guide
c28e60b Add learning documentation in 边学边做/README.md
f6810bd Initial commit: KPL BP Simulator project (React + Go + MySQL + Redis)
```

远程仓库已关联：
```bash
git remote -v
# origin	https://github.com/HUAHUA768/KPL_BP_Simulator.git (fetch)
# origin	https://github.com/HUAHUA768/KPL_BP_Simulator.git (push)
```

---

## 🚀 最终推送：选择你的认证方式

根据你的开发环境和公司类型，选择合适的推送方式。

---

### 场景一：大厂 / 正规公司企业部署（SSH 密钥）⭐ 推荐

**特点**：公司有专门的基础设施团队，配置了统一的 SSH Key 管理系统

#### 适用情况
- 公司 IT 部门提供了统一的代码仓库服务器
- 使用 GitLab、Bitbucket 等企业级平台
- 有多人协作、权限管理的正式项目

#### 操作步骤

**1. 生成 SSH Key（如果还没有）**

```bash
ssh-keygen -t ed25519 -C "your.name@company.com"
# 直接回车，使用默认路径~/.ssh/id_ed25519
```

> 💡 **提示**: `ed25519` 比 `rsa` 更安全更快，推荐使用。

**2. 查看公钥内容**

```bash
cat ~/.ssh/id_ed25519.pub
```

输出类似：
```text
ed25519 your.name@company.com
```

**3. 将公钥添加到 Git 平台**

| 平台 | 添加路径 |
|------|----------|
| GitHub | Settings → SSH and GPG keys → New SSH key |
| GitLab | User Settings → SSH Keys |
| Bitbucket | Settings → SSH keys |

- **Title**: 描述该设备（如 "My Mac Book Pro"）
- **Key**: 粘贴整个公钥内容

**4. 修改远程仓库为 SSH 地址**

```bash
cd /home/xiaohua/KPL_BP_Simulator

# 检查当前远程地址
git remote -v

# 修改为 SSH 格式
git remote set-url origin git@github.com:HUAHUA768/KPL_BP_Simulator.git

# 验证修改成功
git remote -v
# origin	git@github.com:HUAHUA768/KPL_BP_Simulator.git (fetch)
# origin	git@github.com:HUAHUA768/KPL_BP_Simulator.git (push)
```

**5. 测试连接**

```bash
ssh -T git@github.com
```

成功输出：
```
Hi HUAHUA768! You've successfully authenticated, but GitHub does not provide shell access.
```

**6. 推送代码**

```bash
git push -u origin main
```

✅ **完成！之后每次只需 `git push` 即可，无需输入密码！**

---

### 场景二：小厂 / 个人项目（HTTPS + Token）

**特点**：小型创业公司或个人开发，没有复杂的运维体系

#### 适用情况
- 初创团队快速启动项目
- 个人开源项目
- 临时性的 Demo 演示

#### 操作步骤

**1. 创建个人访问令牌（Personal Access Token）**

访问 [GitHub Tokens](https://github.com/settings/tokens)，点击：

1. **Generate new token** → **Generate new token (classic)**
2. 填写名称：`KPL_BP_Simulator`
3. 选择过期时间：建议 90 天
4. 勾选权限：**`repo`**（需要完全控制私有仓库）
5. 点击 **Generate token**
6. **立即复制生成的 Token**（如 `ghp_xxxxxxxxxxxxxxxx`），只显示一次！

> ⚠️ **重要**: 
> - Token 不是你的 GitHub 登录密码！
> - 一旦关闭窗口就无法再查看，务必保存好
> - 不要泄露给他人，等同于账号密码

**2. 推送代码**

```bash
cd /home/xiaohua/KPL_BP_Simulator
git push -u origin main
```

终端提示：

```
Username for 'https://github.com': HUAHUA768
Password for 'https://HUAHUA768@github.com': [这里粘贴 Token，不会显示字符]
```

✅ **完成！**

---

### 场景三：跨设备同步（多电脑开发）

**特点**：需要在办公室电脑、家电脑、笔记本电脑之间同步代码

#### 最佳实践：使用 SSH Key

在**每一台**设备上重复以下步骤：

**1. 生成/确认 SSH Key**

```bash
# 检查是否已有
ls ~/.ssh/id_ed25519*

# 如果没有则生成
ssh-keygen -t ed25519 -C "device-description"
```

**2. 将每台电脑的公钥都添加到 GitHub**

```bash
# 复制公钥
cat ~/.ssh/id_ed25519.pub

# 然后在 GitHub → SSH and GPG keys → New SSH key 中添加
# Title: "Office Mac", "Home PC", "Travel Laptop" 等
```

**3. 统一使用 SSH 地址**

```bash
git remote set-url origin git@github.com:HUAHUA768/KPL_BP_Simulator.git
```

**优势**：所有设备都能无密码推送，团队协作时各用各的 Key 也互不干扰。

---

## 🔧 切换 HTTPS 和 SSH（常用技巧）

### 从 HTTPS 切换到 SSH

```bash
cd /home/xiaohua/KPL_BP_Simulator
git remote set-url origin git@github.com:用户名/仓库名.git
```

### 从 SSH 切回 HTTPS

```bash
git remote set-url origin https://github.com/用户名/仓库名.git
```

---

## 📊 不同场景对比表

| 场景 | 推荐方式 | 优点 | 缺点 |
|------|---------|------|------|
| 大厂企业部署 | SSH Key | 安全、方便、可审计 | 需要 IT 配合配置 |
| 小厂快速启动 | HTTPS + Token | 简单、一次性配置 | Token 可能过期需更新 |
| 多设备开发 | SSH Key | 每台设备独立 Key | 每台设备都要配置 |
| 公共 CI/CD | SSH Key | 自动化友好 | 需要安全存储私钥 |

---

## ❓ 常见问题

### Q1: SSH 连接失败 `Permission denied (publickey).`

**解决步骤**：

```bash
# 1. 重启 SSH agent
eval "$(ssh-agent -s)"

# 2. 添加 Key
ssh-add ~/.ssh/id_ed25519

# 3. 测试连接
ssh -T git@github.com
```

如果还是失败，检查：
- SSH Key 是否已添加到 GitHub
- Key 文件权限是否正确：`chmod 600 ~/.ssh/id_ed25519`

---

### Q2: HTTPS 推送时不知道用户名/Token

**用户名**: 就是你的 GitHub 用户名（如 `HUAHUA768`）

**Token**: 在 GitHub → Settings → Developer settings → Personal access tokens 生成

---

### Q3: Token 过期了怎么办？

重新生成一个新的 Token 即可：

1. 访问 https://github.com/settings/tokens
2. 找到即将过期的 Token，点击右上角... → Regenerate token
3. 复制新 Token
4. 下次推送时使用新 Token

或者切换到 SSH 方式，一劳永逸。

---

## 🎯 推送后的下一步

```bash
# 查看推送结果
open https://github.com/HUAHUA768/KPL_BP_Simulator

# 或在小厂环境直接使用内网地址
# open http://gitlab.internal.company.com/team/repo
```

在网页上你可以：
- ✅ 查看所有提交历史
- ✅ 下载 ZIP 代码包
- ✅ 开启 Issues、Projects
- ✅ 邀请团队成员

---

## 📝 总结

| 你属于哪种场景 | 应该用的方式 |
|---------------|-------------|
| 大公司正规项目 | **SSH Key**（联系 IT 部门获取统一 Key） |
| 小厂快速启动 | **HTTPS + Token**（5 分钟搞定） |
| 个人开发 / 学习 | **SSH Key**（一劳永逸） |
| 多台设备协同 | **SSH Key**（每台设备配一个 Key） |

---

**准备好推送了吗？**

```bash
cd /home/xiaohua/KPL_BP_Simulator && git push -u origin main
```

根据你的实际情况选择对应的认证方式，然后执行上面的命令！🚀

---

## 🛠️ 额外工具：管理多个 SSH Key

如果你需要在同一个电脑上访问多个 Git 平台（GitHub + GitLab），可以配置 SSH Config：

**~/\.ssh/config**:

```config
# GitHub
Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_github

# GitLab (公司内部)
Host gitlab.internal.company.com
    HostName gitlab.internal.company.com
    User git
    IdentityFile ~/.ssh/id_ed25519_company
```

这样不同类型的仓库会自动使用对应的 Key！
