from fastapi import FastAPI, File, UploadFile, Form, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import os
import tempfile
import shutil
import subprocess
from typing import Optional
import markdown

# 处理Word文档
try:
    import docx
    has_docx = True
except ImportError:
    has_docx = False

# 处理PDF文档
try:
    from PyPDF2 import PdfReader
    has_pdf = True
except ImportError:
    has_pdf = False

app = FastAPI(title="文件解析辅助服务")

# 添加CORS中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/")
async def root():
    return {"message": "文件解析辅助服务已启动"}

@app.post("/parse")
async def parse_file(
    file: UploadFile = File(...),
    file_type: str = Form(...)
):
    # 创建临时文件
    temp_file_path = ""
    try:
        # 保存上传的文件到临时目录
        with tempfile.NamedTemporaryFile(delete=False, suffix=file_type) as temp_file:
            temp_file_path = temp_file.name
            shutil.copyfileobj(file.file, temp_file)
        
        # 根据文件类型解析
        file_type = file_type.lower()
        
        if file_type == '.docx':
            content = parse_docx(temp_file_path)
        elif file_type == '.doc':
            content = parse_doc(temp_file_path)
        elif file_type == '.pdf':
            content = parse_pdf(temp_file_path)
        elif file_type == '.md':
            content = parse_markdown(temp_file_path)
        elif file_type == '.txt':
            content = parse_text(temp_file_path)
        else:
            raise HTTPException(status_code=400, detail=f"不支持的文件类型: {file_type}")
        
        return {"content": content}
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    
    finally:
        # 清理临时文件
        if temp_file_path and os.path.exists(temp_file_path):
            os.unlink(temp_file_path)

def parse_text(file_path: str) -> str:
    """解析文本文件"""
    with open(file_path, 'r', encoding='utf-8', errors='replace') as file:
        return file.read()

def parse_markdown(file_path: str) -> str:
    """解析Markdown文件"""
    with open(file_path, 'r', encoding='utf-8', errors='replace') as file:
        md_content = file.read()
        # 可选：将Markdown转换为HTML
        # html_content = markdown.markdown(md_content)
        # return html_content
        return md_content

def parse_docx(file_path: str) -> str:
    """解析Word文档(.docx格式)"""
    if not has_docx:
        raise HTTPException(status_code=500, detail="未安装python-docx库")
    
    doc = docx.Document(file_path)
    full_text = []
    for para in doc.paragraphs:
        full_text.append(para.text)
    return '\n'.join(full_text)

def parse_doc(file_path: str) -> str:
    """解析Word文档(.doc格式)，使用antiword工具"""
    try:
        # 使用antiword命令行工具解析.doc文件
        result = subprocess.run(['antiword', file_path], capture_output=True, text=True, check=True)
        if result.returncode == 0:
            return result.stdout
        else:
            raise HTTPException(status_code=500, detail=f"解析.doc文件失败: {result.stderr}")
    except FileNotFoundError:
        raise HTTPException(status_code=500, detail="未安装antiword工具，无法解析.doc格式")
    except subprocess.CalledProcessError as e:
        raise HTTPException(status_code=500, detail=f"解析.doc文件失败: {e.stderr}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"解析.doc文件失败: {str(e)}")

def parse_pdf(file_path: str) -> str:
    """解析PDF文档"""
    if has_pdf:
        try:
            with open(file_path, 'rb') as file:
                reader = PdfReader(file)
                text = ""
                for page in reader.pages:
                    text += page.extract_text() + "\n"
                return text
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"解析PDF文件失败: {str(e)}")
    
    raise HTTPException(status_code=500, detail="无法解析PDF文件，缺少必要的库")

if __name__ == "__main__":
    import uvicorn
    import os
    
    # 从环境变量获取端口，默认为4002
    port = int(os.environ.get("PORT", 4002))
    uvicorn.run("main:app", host="0.0.0.0", port=port, reload=True) 