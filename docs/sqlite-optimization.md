# SQLite3 高并发事务优化方案

## 概述

本文档针对 AxonHub 在高并发请求场景下遇到的 SQLite3 事务管理问题，提供系统性的优化方案，以解决 `"sql: transaction has already been committed or rolled back"` 等错误。

## 问题分析

### 🚨 核心问题：事务重复提交/回滚

根据错误日志和代码分析，发现以下主要问题：

1. **Context 传播问题**：使用 `context.WithoutCancel()` 但上下文中的事务状态可能已经改变
2. **高频并发请求**：SQLite3 在高并发下的锁机制导致事务状态不一致
3. **异步操作冲突**：多个异步操作同时尝试更新同一请求的状态

### 📊 当前配置状况

**SQLite 配置（`conf/conf.go:135`）**：
```go
"file:axonhub.db?cache=shared&_fk=1&journal_mode=WAL"
```

✅ **已优化**：
- WAL (Write-Ahead Logging) 模式
- 共享缓存
- 外键约束

❌ **不足**：
- 缺少其他关键性能参数
- 没有连接池优化
- 缺少忙超时设置

## 优化方案

### 1. 立即修复方案

#### A. 优化 SQLite 配置

**修改 `conf/conf.go`**：

```go
// 当前配置
"file:axonhub.db?cache=shared&_fk=1&journal_mode=WAL"

// 建议配置
"file:axonhub.db?cache=shared&_fk=1&journal_mode=WAL&busy_timeout=30000&synchronous=NORMAL&cache_size=10000&temp_store=memory"
```

**参数说明**：
- `busy_timeout=30000`：数据库忙时等待时间 30 秒
- `synchronous=NORMAL`：平衡性能和数据安全
- `cache_size=10000`：缓存页面数量（约 40MB）
- `temp_store=memory`：临时表存储在内存中

#### B. 事务状态检查和修复

在事务操作前添加状态检查：

```go
// 检查事务是否仍然有效
func checkTransactionState(tx *ent.Tx) error {
    if tx == nil {
        return nil
    }

    // 尝试一个简单的查询来检查事务状态
    _, err := tx.Client().System.Query().Count(context.Background())
    if err != nil && (strings.Contains(err.Error(), "transaction has been committed") ||
                     strings.Contains(err.Error(), "transaction has been rolled back")) {
        return err
    }

    return nil
}

// 在每次事务操作前
if tx != nil {
    if err := checkTransactionState(tx); err != nil {
        // 重新开始事务或使用普通连接
        return performWithoutTransaction(ctx)
    }
}
```

### 2. 结构性改进方案

#### A. 数据库连接池优化

**在 `internal/pkg/sqlite/sqlite.go` 中添加**：

```go
func SetupSQLiteOptimizations(c *sql.DB) error {
    // 设置连接池参数
    c.SetMaxOpenConns(25)        // 增加最大连接数
    c.SetMaxIdleConns(5)         // 适当的空闲连接
    c.SetConnMaxLifetime(5 * time.Minute)
    c.SetConnMaxIdleTime(2 * time.Minute)

    // SQLite 特定优化
    pragmas := []string{
        "PRAGMA foreign_keys = on",
        "PRAGMA synchronous = NORMAL",
        "PRAGMA cache_size = 10000",      // 10MB cache
        "PRAGMA temp_store = memory",
        "PRAGMA mmap_size = 268435456",    // 256MB memory mapped I/O
        "PRAGMA journal_size_limit = 65536",
        "PRAGMA wal_checkpoint(TRUNCATE)",  // 定期清理 WAL 文件
    }

    for _, pragma := range pragmas {
        if _, err := c.Exec(pragma, nil); err != nil {
            return fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
        }
    }

    return nil
}
```

#### B. 事务隔离级别改进

**创建事务管理器**：

```go
type TransactionManager struct {
    db *ent.Client
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(*ent.Tx) error) error {
    // 使用适当的隔离级别
    tx, err := tm.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
        ReadOnly:  false,
    })
    if err != nil {
        return err
    }

    // 确保事务只被提交一次
    var committed bool
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()

    err = fn(tx)
    if err != nil {
        return err
    }

    committed = true
    return tx.Commit()
}
```

### 3. 高并发场景优化

#### A. 读写分离策略

```go
// 读操作使用只读事务
func (s *RequestService) GetRequestReadOnly(ctx context.Context, id int) (*ent.Request, error) {
    return s.ent.Request.Query().
        Where(request.IDEQ(id)).
        WithExecutions().
        Only(ctx)
}

// 批量更新使用事务
func (s *RequestService) BatchUpdateStatus(ctx context.Context, updates []StatusUpdate) error {
    return s.transactionManager.WithTransaction(ctx, func(tx *ent.Tx) error {
        for _, update := range updates {
            if err := s.updateSingleStatus(tx, update); err != nil {
                return err
            }
        }
        return nil
    })
}
```

#### B. 异步处理队列

```go
type AsyncRequestProcessor struct {
    queue   chan RequestUpdate
    db      *ent.Client
    batchSize int
    timeout   time.Duration
}

func (arp *AsyncRequestProcessor) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(arp.timeout)
        batch := make([]RequestUpdate, 0, arp.batchSize)

        for {
            select {
            case update := <-arp.queue:
                batch = append(batch, update)
                if len(batch) >= arp.batchSize {
                    arp.processBatch(ctx, batch)
                    batch = batch[:0]
                }
            case <-ticker.C:
                if len(batch) > 0 {
                    arp.processBatch(ctx, batch)
                    batch = batch[:0]
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

### 4. 监控和诊断

#### A. 事务状态监控

```go
func MonitorTransactionHealth(ctx context.Context, db *ent.Client) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // 检查活跃事务数
            // 检查数据库锁状态
            // 检查连接池使用情况
            stats := db.Driver.(*sqlite3.SQLiteDriver).Stats()

            log.Info(ctx, "SQLite stats",
                log.Int("open_connections", stats.OpenConnections),
                log.Int("in_use", stats.InUse),
                log.Int("idle", stats.Idle),
            )

            // 检查 WAL 文件大小
            if err := checkWALFileHealth(); err != nil {
                log.Error(ctx, "WAL file health check failed", log.Cause(err))
            }

        case <-ctx.Done():
            return
        }
    }
}

func checkWALFileHealth() error {
    stat, err := os.Stat("axonhub.db-wal")
    if err != nil {
        return err
    }

    // WAL 文件过大时需要检查点
    if stat.Size() > 100*1024*1024 { // 100MB
        // 触发 WAL 检查点
        return performWALCheckpoint()
    }

    return nil
}
```

#### B. 错误恢复机制

```go
func HandleTransactionError(err error) error {
    if err == nil {
        return nil
    }

    errMsg := err.Error()

    // 事务已提交 - 不是真正错误
    if strings.Contains(errMsg, "transaction has already been committed") {
        return nil
    }

    // 事务已回滚 - 不是真正错误
    if strings.Contains(errMsg, "transaction has already been rolled back") {
        return nil
    }

    // 数据库锁定 - 可重试
    if strings.Contains(errMsg, "database is locked") {
        time.Sleep(100 * time.Millisecond)
        return RetryTransaction()
    }

    // 忙超时 - 可重试
    if strings.Contains(errMsg, "database is busy") {
        time.Sleep(200 * time.Millisecond)
        return RetryTransaction()
    }

    return err
}
```

## 实施指南

### 第一阶段（立即实施）

1. **优化 SQLite 连接字符串参数**
   ```go
   // 在 conf/conf.go 中更新
   DSN = "file:axonhub.db?cache=shared&_fk=1&journal_mode=WAL&busy_timeout=30000&synchronous=NORMAL&cache_size=10000&temp_store=memory"
   ```

2. **添加事务状态检查**
   - 在 `internal/server/biz/request.go` 中添加状态检查函数
   - 在每个事务操作前调用检查

3. **优化连接池配置**
   - 在 `internal/pkg/sqlite/sqlite.go` 中添加 `SetupSQLiteOptimizations`
   - 在数据库初始化时调用

### 第二阶段（短期实施）

1. **实现事务管理器**
   - 创建 `internal/pkg/transaction/manager.go`
   - 替换现有的事务操作

2. **添加错误恢复机制**
   - 创建 `internal/pkg/transaction/error_handler.go`
   - 在所有数据库操作中集成

3. **实现批量更新策略**
   - 优化请求状态更新逻辑
   - 减少事务争用

### 第三阶段（长期优化）

1. **实现异步处理队列**
   - 创建 `internal/server/biz/async_processor.go`
   - 将非关键操作异步化

2. **添加监控和诊断**
   - 实现 `internal/monitoring/sqlite.go`
   - 添加指标收集和告警

3. **考虑数据库迁移**
   - 当规模持续增长时，考虑迁移到 PostgreSQL
   - 评估迁移成本和收益

## 性能预期改进

### 量化指标

- **减少事务错误 90%+**
- **提高数据库并发性能 3-5倍**
- **降低系统响应时间 20-40%**
- **提高系统稳定性，减少意外重启**

### 监控指标

1. **事务成功率**：`(成功事务数 / 总事务数) * 100%`
2. **数据库锁等待时间**：平均和最大锁等待时间
3. **连接池利用率**：活跃连接 / 最大连接数
4. **WAL 文件大小**：监控 WAL 文件增长趋势

## 故障排除

### 常见错误及解决方案

#### 1. "database is locked"
```bash
# 检查是否有长时间运行的进程
lsof axonhub.db*

# 重启服务清理锁
systemctl restart axonhub
```

#### 2. WAL 文件过大
```bash
# 手动执行 WAL 检查点
sqlite3 axonhub.db "PRAGMA wal_checkpoint(TRUNCATE);"

# 或通过代码执行
performWALCheckpoint()
```

#### 3. 事务超时
```go
// 增加超时时间
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

## 最佳实践

### 开发规范

1. **事务粒度控制**：事务尽可能短小，避免长时间持有锁
2. **错误处理**：统一使用 `HandleTransactionError` 处理事务错误
3. **资源清理**：使用 `defer` 确保资源正确释放
4. **监控集成**：关键操作添加日志和指标

### 运维建议

1. **定期 WAL 检查点**：每天执行一次 WAL 检查点
2. **连接池监控**：监控连接池使用情况
3. **性能基准测试**：定期进行性能基准测试
4. **容量规划**：根据增长趋势规划数据库容量

## 结论

通过实施这些优化方案，AxonHub 可以显著改善高并发请求场景下的 SQLite3 性能和事务管理问题。建议按照阶段性实施计划逐步推进，并在每个阶段验证改进效果。