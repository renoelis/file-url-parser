# 文件URL解析服务

这是一个基于Go和Python的文件URL解析服务，能够解析多种格式的文件内容。

## 功能特点

- 支持解析Excel文件为JSON数组对象，自动识别日期格式
- 支持解析Word、PDF、Markdown、TXT等文本文件
- 提供简单的RESTful API接口
- 使用Go作为主服务，Python作为辅助服务处理复杂文件格式
- 智能处理日期格式：将"1998/9/9 12:30:05"格式转换为"1998-09-09 12:30:05"
- 自动将逗号分隔的内容转换为JSON数组
- 限制Excel/CSV文件最多解析200行数据，超出限制将返回错误
- 支持控制是否使用表头作为键，可选择使用统一格式的键名（Col_1, Col_2...）
- 按表头顺序输出数据，保证JSON响应中的字段顺序与Excel/CSV表头一致
- 当使用统一格式键名（Col_X）时，确保按照数字顺序排序，而非字典序
- 智能检测表格数据的实际起始位置，支持解析不从左上角开始的表格数据
- 内置API速率限制，默认限制为240次/秒，可通过配置调整

## 数据处理特性

### 日期格式处理
- 只对格式为带有"/"的日期进行转换为"-"分隔的标准格式，支持以下格式：
  - 年/月/日 时:分:秒 → 年-月-日 时:分:秒（如 1998/9/9 12:30:05 → 1998-09-09 12:30:05）
  - 年/月/日 时:分 → 年-月-日 时:分（如 1998/9/9 12:30 → 1998-09-09 12:30）
  - 年/月/日 时 → 年-月-日 时（如 1998/9/9 12 → 1998-09-09 12）
  - 年/月/日 → 年-月-日（如 1998/9/9 → 1998-09-09）
  - 年/月 → 年-月（如 1998/9 → 1998-09）
  - 年/ → 年（如 1998/ → 1998）
- 如果已经是带有"-"的标准格式则保持不变
- 其他格式的日期（如 "2023年4月5日"）直接原样输出

### 逗号分隔内容处理
- 自动检测逗号分隔的内容并转换为数组格式
- 例如 "member1@company.com,member2@company.com,member3@company.com" 会被转换为 ["member1@company.com","member2@company.com","member3@company.com"]
- 特别适用于URL字段，如 "https://example.com/1.jpg,https://example.com/2.jpg" 会被转换为 ["https://example.com/1.jpg","https://example.com/2.jpg"]
- 系统会在Excel/CSV解析过程中自动处理所有包含逗号的字符串字段，无需额外配置

### 表格起始位置自动检测
- 智能检测Excel/CSV表格中数据的实际起始位置，不要求数据必须从A1单元格开始
- 自动识别表头行和数据起始列，适用于各种布局的表格
- 能够处理在表格中间位置（如D10单元格附近）开始的数据
- 自动跳过空行和空列，只处理有效数据
- 确保正确识别表头和对应的数据列，保证解析结果的准确性

## 支持的文件格式

- Excel (.xlsx, .xls)：解析为数组对象，表头为键，每行为对应的键值对
- Word (.docx, .doc)：解析为文本内容
- PDF (.pdf)：解析为文本内容
- Markdown (.md)：解析为原始Markdown文本
- 文本文件 (.txt)：解析为文本内容

## 项目结构

```
file-url-parser/
├── cmd/
│   └── main.go               # 启动入口
├── config/
│   └── config.go             # 配置加载
├── controller/
│   └── handler.go            # 路由处理逻辑
├── model/
│   └── model.go              # 数据结构定义
├── service/
│   ├── excel_parser.go       # Excel解析服务
│   ├── text_parser.go        # 文本解析服务
│   └── parser_service.go     # 解析服务主逻辑
├── router/
│   └── router.go             # 路由注册
├── utils/
│   └── helper.go             # 工具函数
├── python_ext/               # Python辅助服务
│   ├── app/
│   │   └── main.py           # Python服务入口
│   └── requirements.txt      # Python依赖
├── Dockerfile                # Go服务Dockerfile
├── Dockerfile.python         # Python服务Dockerfile
├── docker-compose-file-url-parser.yml # Docker Compose配置
├── .dockerignore
└── README.md
```

## 接口说明

### 🧩 接口 /fileProcess/parse

- 方法：POST
- 描述：解析文件URL的内容
- 请求体：
  ```json
  {
    "url": "https://example.com/path/to/file.xlsx",
    "use_header_as_key": true
  }
  ```
  > `use_header_as_key` 参数为可选，默认为 true。设置为 false 时，将使用统一格式的键名（Col_1, Col_2...）代替原始表头。

- 响应（Excel/CSV文件）：
  ```json
  {
    "data": [
      {
        "列1": "值1",
        "列2": "值2",
        "日期列": "2023-01-01"
      },
      {
        "列1": "值3",
        "列2": "值4",
        "日期列": "2023-01-02"
      }
    ],
    "headers": ["列1", "列2", "日期列"],
    "original_headers": ["列1", "列2", "日期列"]
  }
  ```
  > 注意：响应中的数据字段顺序与表头顺序一致，便于前端展示。当 `use_header_as_key=false` 时，`headers` 将是 `["Col_1", "Col_2", "Col_3"]`，而 `original_headers` 将保留原始表头。系统确保 Col_X 格式的键名按照数字顺序排列（如 Col_1, Col_2, ..., Col_10, Col_11），而不是字典序（Col_1, Col_10, Col_11, Col_2...）。

- 响应（文本文件）：
  ```json
  {
    "content": "文件的文本内容..."
  }
  ```

- 错误响应：
  ```json
  {
    "error": "错误信息"
  }
  ```

## 🔧 模块说明

### controller/handler.go
- 功能：处理HTTP请求，验证URL参数
- 调用链：router → controller → service

### service/parser_service.go
- 功能：主要的解析逻辑，根据文件类型调用不同的解析器
- 调用链：controller → parser_service → excel_parser/text_parser

### service/excel_parser.go
- 功能：解析Excel文件为数组对象
- 特点：自动识别日期格式，支持数值转换

### service/text_parser.go
- 功能：处理文本文件和复杂文件格式
- 特点：调用Python辅助服务处理Word和PDF等格式

### python_ext/app/main.py
- 功能：Python辅助服务，处理复杂文件格式
- 支持：Word、PDF、Markdown等格式解析

## 部署说明

### 环境要求

- Docker 和 Docker Compose
- 外部网络：api-proxy_proxy_net（可根据需要修改）

### Docker 部署

#### 构建和启动服务

1. 构建Docker镜像：

```bash
docker-compose -f docker-compose-file-url-parser.yml build
```

2. 启动服务：

```bash
docker-compose -f docker-compose-file-url-parser.yml down
docker-compose -f docker-compose-file-url-parser.yml up -d
```

#### 服务说明

- Go主服务：
  - 容器名称：file-url-parser-go
  - 端口：4001
  - 环境变量：
    - PORT：服务端口
    - PYTHON_SERVICE_URL：Python辅助服务URL
    - MAX_FILE_SIZE：最大文件大小（字节）
    - MAX_ALLOWED_ROWS：Excel/CSV文件最大允许解析的数据行数，默认为 200

- Python辅助服务：
  - 容器名称：file-url-parser-python
  - 端口：4002

### 本地开发与测试

如果您想在本地运行该服务进行开发或测试，可以按照以下步骤操作：

#### 1. 准备环境

首先，您需要安装以下软件：

- Go 语言环境（建议 Go 1.16 或更高版本）
- Python 3.7 或更高版本（用于辅助服务）
- 相关依赖包

#### 2. 启动 Go 主服务

1. 进入项目根目录

2. 安装 Go 依赖：
   ```bash
   go mod tidy
   ```

3. 编译并启动 Go 服务：
   ```bash
   go run cmd/main.go
   ```

   或者，您也可以先构建再运行：
   ```bash
   go build -o file-url-parser cmd/main.go
   ./file-url-parser
   ```

   默认情况下，Go 服务会在 4001 端口启动。您可以通过设置环境变量来修改：
   ```bash
   export PORT=4001
   export PYTHON_SERVICE_URL=http://localhost:4002
   export MAX_FILE_SIZE=10485760  # 10MB
   export MAX_ALLOWED_ROWS=200  # 默认200行
   ```

#### 3. 启动 Python 辅助服务

1. 进入 Python 服务目录：
   ```bash
   cd python_ext
   ```

2. 安装 Python 依赖：
   ```bash
   pip3 install -r requirements.txt
   ```

3. 启动 Python 服务：
   ```bash
   python3 app/main.py
   ```

   Python 服务默认在 4002 端口启动。

#### 4. 开发模式（可选）

如果您是在开发模式下运行，可以使用以下命令实现热重载：

对于 Go 服务，可以使用 air 工具：
```bash
# 安装 air
go install github.com/cosmtrek/air@latest

# 使用 air 启动服务
air
```

对于 Python 服务，可以使用 Flask 的开发模式：
```bash
export FLASK_APP=app/main.py
export FLASK_ENV=development
flask run --port=4002
```

#### 5. 常见问题处理

- 如果遇到端口冲突，可以修改 PORT 环境变量
- 确保 Go 服务能够正确连接到 Python 服务
- 检查是否安装了所有必要的依赖包
- 如果遇到函数重复声明的编译错误（如 `isLikelyDate redeclared in this block`），这是因为相同的函数在 `excel_parser.go` 和 `csv_parser.go` 中都有定义。解决方法是删除 `csv_parser.go` 中的重复函数，只保留 `excel_parser.go` 中的函数定义。
- 如果遇到 `"xxx" imported and not used` 错误，需要删除未使用的包导入。特别是在删除了 `csv_parser.go` 中的函数后，可能会导致 `strings` 和 `time` 包不再被使用，需要从导入列表中删除。

## 调用示例

使用curl发送请求：

```bash
curl -X POST http://localhost:4001/fileProcess/parse \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/path/to/file.xlsx"}'
```

## 流程图

```mermaid
graph TD
    A[接收文件URL] --> B{检查URL有效性}
    B -->|无效| C[返回错误]
    B -->|有效| D{判断文件类型}
    D -->|Excel| E[使用Go解析Excel]
    D -->|Word/PDF/其他| F{Go能处理?}
    F -->|是| G[Go直接处理]
    F -->|否| H[调用Python辅助服务]
    E --> I[转换为数组对象]
    G --> J[提取文本内容]
    H --> J
    I --> K[返回结果]
    J --> K
```

### 环境变量配置

服务支持以下环境变量配置：

- `PORT`：服务端口，默认为 4001
- `PYTHON_SERVICE_URL`：Python辅助服务URL，默认为 http://localhost:4002
- `MAX_FILE_SIZE`：最大文件大小（字节），默认为 10MB (10485760)
- `MAX_ALLOWED_ROWS`：Excel/CSV文件最大允许解析的数据行数，默认为 200
- `USE_HEADER_AS_KEY`：是否默认使用表头作为键，默认为 true
- `GIN_MODE`：Gin框架运行模式，设置为 release 用于生产环境
- `RATE_LIMIT`：API接口调用频率限制，默认为 240次/秒

这些环境变量可以在部署时设置，例如：

```bash
export RATE_LIMIT=300  # 设置API调用限制为300次/秒
export GIN_MODE=release  # 设置Gin为发布模式
export MAX_ALLOWED_ROWS=500  # 设置允许解析的最大行数为500行
go run cmd/main.go
```

或者在Docker环境中：

```yaml
# docker-compose-file-url-parser.yml
services:
  file-url-parser-go:
    environment:
      - RATE_LIMIT=300  # 设置API调用限制为300次/秒
      - GIN_MODE=release  # 设置Gin为发布模式
      - MAX_ALLOWED_ROWS=500  # 设置允许解析的最大行数为500行
``` 