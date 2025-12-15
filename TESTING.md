# VStats 自动化测试指南

本文档详细介绍了 VStats 项目的自动化测试方案，包括单元测试、集成测试和端到端测试。

## 目录

- [概述](#概述)
- [测试架构](#测试架构)
- [快速开始](#快速开始)
- [Go 后端测试](#go-后端测试)
- [前端测试](#前端测试)
- [E2E 测试](#e2e-测试)
- [CI/CD 集成](#cicd-集成)
- [最佳实践](#最佳实践)

## 概述

VStats 采用多层测试策略：

| 测试类型 | 工具 | 目的 |
|---------|------|------|
| Go 单元测试 | `go test` | 测试后端业务逻辑 |
| Go 集成测试 | `go test` + SQLite | 测试数据库交互 |
| 前端单元测试 | Vitest + React Testing Library | 测试组件和工具函数 |
| E2E 测试 | Playwright | 测试完整用户流程 |

## 测试架构

```
vstats/
├── server-go/
│   └── cmd/server/
│       ├── *_test.go          # Go 单元/集成测试
│       └── testdata/          # 测试数据文件
├── web/
│   ├── src/
│   │   ├── components/*.test.tsx  # 组件测试
│   │   ├── context/*.test.tsx     # Context 测试
│   │   ├── utils/*.test.ts        # 工具函数测试
│   │   └── test/
│   │       ├── setup.ts           # 测试设置
│   │       └── utils.tsx          # 测试工具函数
│   └── vitest.config.ts       # Vitest 配置
├── e2e/
│   ├── tests/
│   │   ├── dashboard.spec.ts  # Dashboard E2E 测试
│   │   ├── login.spec.ts      # 登录 E2E 测试
│   │   └── fixtures/          # 测试 fixtures
│   └── playwright.config.ts   # Playwright 配置
└── .github/workflows/
    └── ci.yml                 # CI/CD 配置
```

## 快速开始

### 安装依赖

```bash
# 安装所有依赖
make install

# 或分别安装
cd web && npm ci
cd e2e && npm ci
cd server-go && go mod download
```

### 运行所有测试

```bash
# 运行单元测试（Go + 前端）
make test

# 运行所有测试（包括 E2E）
make test-all
```

## Go 后端测试

### 运行测试

```bash
# 运行所有 Go 测试
make go-test

# 带覆盖率
make go-test-cover

# 快速模式（跳过慢测试）
make go-test-short

# 直接使用 go test
cd server-go
go test -v ./...
go test -v ./cmd/server/...
go test -run TestLogin ./cmd/server/...
```

### 测试文件结构

```go
// db_test.go - 数据库层测试
func TestMetricsBuffer(t *testing.T) { ... }
func TestDBWriter(t *testing.T) { ... }

// handlers_auth_test.go - API 处理器测试
func TestLogin(t *testing.T) { ... }
func TestChangePassword(t *testing.T) { ... }

// cache_test.go - 缓存层测试
func TestHistoryCache(t *testing.T) { ... }
```

### 编写测试示例

```go
package main

import (
    "testing"
    "net/http/httptest"
    "github.com/gin-gonic/gin"
)

func TestMyHandler(t *testing.T) {
    // 设置
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/api/test", myHandler)

    // 执行
    req := httptest.NewRequest("GET", "/api/test", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // 验证
    if w.Code != http.StatusOK {
        t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
    }
}

// 表驱动测试
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty", "", "default"},
        {"valid", "test", "TEST"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := myFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}

// 基准测试
func BenchmarkMyFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        myFunction("test")
    }
}
```

## 前端测试

### 运行测试

```bash
# 运行测试
make web-test

# 监视模式
make web-test-watch

# 带覆盖率
make web-test-cover

# UI 模式
make web-test-ui

# 直接使用 npm
cd web
npm run test
npm run test:run
npm run test:coverage
```

### 测试配置

Vitest 配置位于 `web/vitest.config.ts`：

```typescript
export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      thresholds: {
        statements: 50,
        branches: 50,
        functions: 50,
        lines: 50,
      }
    },
  },
})
```

### 编写组件测试

```tsx
// ComponentName.test.tsx
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MyComponent } from './MyComponent'

describe('MyComponent', () => {
  it('renders correctly', () => {
    render(<MyComponent title="Test" />)
    expect(screen.getByText('Test')).toBeInTheDocument()
  })

  it('handles click events', () => {
    const onClick = vi.fn()
    render(<MyComponent onClick={onClick} />)
    
    fireEvent.click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalled()
  })

  it('updates on user input', async () => {
    render(<MyComponent />)
    
    const input = screen.getByRole('textbox')
    await userEvent.type(input, 'hello')
    
    expect(input).toHaveValue('hello')
  })
})
```

### 使用测试工具函数

```tsx
import { render, createMockServer } from '../test/utils'

describe('ServerCard', () => {
  it('displays server info', () => {
    const server = createMockServer({ 
      server_name: 'Test Server',
      online: true 
    })
    
    render(<ServerCard server={server} />)
    expect(screen.getByText('Test Server')).toBeInTheDocument()
  })
})
```

## E2E 测试

### 安装 Playwright

```bash
# 安装浏览器
make e2e-install

# 或直接使用
cd e2e && npx playwright install --with-deps
```

### 运行测试

```bash
# 运行所有 E2E 测试
make e2e-test

# 带 UI
make e2e-test-ui

# 有头模式（可见浏览器）
make e2e-test-headed

# 查看报告
make e2e-report

# 生成测试代码
make e2e-codegen
```

### 编写 E2E 测试

```typescript
// tests/my-feature.spec.ts
import { test, expect } from '@playwright/test'

test.describe('My Feature', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('should display dashboard', async ({ page }) => {
    await expect(page).toHaveTitle(/VStats/)
    await expect(page.locator('main')).toBeVisible()
  })

  test('should login successfully', async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="password"]', 'admin')
    await page.click('button[type="submit"]')
    
    await page.waitForURL('/')
    await expect(page).toHaveURL('/')
  })

  test('should be responsive', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await expect(page.locator('body')).toBeVisible()
  })
})
```

### 使用 Fixtures

```typescript
// tests/fixtures/auth.ts
import { test as base } from '@playwright/test'

export const test = base.extend({
  authenticatedPage: async ({ page }, use) => {
    await page.goto('/')
    await page.evaluate(() => {
      localStorage.setItem('token', 'test-token')
    })
    await use(page)
  },
})
```

## CI/CD 集成

### GitHub Actions 工作流

CI 配置位于 `.github/workflows/ci.yml`：

| Job | 描述 |
|-----|------|
| `go-test` | 运行 Go 测试和静态分析 |
| `go-build` | 在多平台构建 Go 二进制 |
| `frontend-test` | 运行前端测试和 lint |
| `frontend-build` | 构建前端 |
| `e2e-test` | 运行 E2E 测试 |
| `docker-build` | 构建 Docker 镜像 |
| `security-scan` | 运行安全扫描 |

### 本地运行 CI 检查

```bash
# 运行完整 CI 检查
make lint
make test
make build

# 快速检查
make go-lint
make web-lint
```

### 覆盖率报告

- **Go**: 使用 `go test -coverprofile` 生成
- **前端**: 使用 Vitest V8 provider 生成
- **上传**: 自动上传到 Codecov（如果配置）

## 最佳实践

### 1. 测试命名规范

```go
// Go: TestFunctionName_Scenario_ExpectedBehavior
func TestLogin_ValidPassword_ReturnsToken(t *testing.T) { }
func TestLogin_InvalidPassword_ReturnsUnauthorized(t *testing.T) { }
```

```typescript
// TypeScript: describe 块 + it/test 描述
describe('Login', () => {
  it('should return token for valid password', () => {})
  it('should return 401 for invalid password', () => {})
})
```

### 2. 测试隔离

- 每个测试独立运行
- 使用 `beforeEach`/`afterEach` 重置状态
- 使用内存数据库或 mock
- 避免测试间依赖

### 3. Mock 使用原则

```go
// Go: 使用接口进行依赖注入
type DBInterface interface {
    Query(query string) error
}

func NewService(db DBInterface) *Service {
    return &Service{db: db}
}
```

```typescript
// TypeScript: 使用 vi.mock 或 vi.fn
vi.mock('../api/client', () => ({
  fetchData: vi.fn().mockResolvedValue({ data: [] })
}))
```

### 4. 测试数据

- 使用工厂函数创建测试数据
- 避免硬编码 magic numbers
- 使用有意义的测试数据

```typescript
// test/factories.ts
export const createMockServer = (overrides = {}) => ({
  server_id: 'test-1',
  server_name: 'Test Server',
  online: true,
  ...overrides,
})
```

### 5. 异步测试

```go
// Go: 使用 channel 和 timeout
func TestAsync(t *testing.T) {
    done := make(chan bool)
    go func() {
        // async work
        done <- true
    }()
    
    select {
    case <-done:
        // success
    case <-time.After(5 * time.Second):
        t.Fatal("timeout")
    }
}
```

```typescript
// TypeScript: 使用 async/await
it('should fetch data', async () => {
  const result = await fetchData()
  expect(result).toBeDefined()
})
```

### 6. 快照测试

```typescript
// 用于 UI 回归测试
it('should match snapshot', () => {
  const { container } = render(<MyComponent />)
  expect(container).toMatchSnapshot()
})
```

### 7. 性能测试

```go
// Go benchmark
func BenchmarkFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        myFunction()
    }
}

// 运行: go test -bench=. -benchmem
```

## 常见问题

### Q: 测试运行太慢？

1. 使用 `-short` 标志跳过慢测试
2. 并行运行测试 `go test -parallel 4`
3. 使用内存数据库而非真实数据库

### Q: E2E 测试不稳定？

1. 增加等待时间和超时
2. 使用 `waitForLoadState('networkidle')`
3. 添加重试机制
4. 检查测试隔离

### Q: Mock 不生效？

1. 确保 mock 在导入前设置
2. 检查模块路径是否正确
3. 使用 `vi.resetAllMocks()` 重置

## 相关资源

- [Go Testing](https://golang.org/pkg/testing/)
- [Vitest Documentation](https://vitest.dev/)
- [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/)
- [Playwright Documentation](https://playwright.dev/)

