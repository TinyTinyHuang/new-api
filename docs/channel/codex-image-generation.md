# Codex 生图能力实测与对外 Images 接口

渠道类型：**ChatGPT Subscription (Codex)**（类型 `57`）。上游为 `https://chatgpt.com/backend-api/codex/responses`，不是 OpenAI Platform 的 `/v1/images/*`。

实测时间：2026-09-06。模型：`gpt-5.6-sol`。上游生图工具模型：`gpt-image-2-codex`。

## 1. 实测结论

### 1.1 能做什么

| 能力 | 结果 |
| --- | --- |
| 文本 `/v1/responses` | 可以。Codex 要求 `input` 为数组，且必须 `"stream": true`。字符串 `input` 会报 `Input must be a list`；非流式会报 `Stream must be set to true`。 |
| `/v1/chat/completions` | 不行。适配器拒绝。 |
| 原生 `/v1/images/generations`（改造前） | 不行。适配器拒绝。 |
| Responses 内置工具 `image_generation` 文生图 | 可以。约 20–80 秒返回 PNG，`image_generation_call.result` 为 base64。 |
| 参考图 + 描述改图 | 可以。`input_image`（`data:image/png;base64,...`）+ 文本，工具 `action` 为 `edit`。输入侧会出现 `image_gen.input_tokens_details.image_tokens > 0`。像素也跟得上参考构图（实测左青右品红、中心白圆）。 |

### 1.2 分辨率边界（提示词不能指定精确像素）

质量回显始终是 `low`。`quality: high` / `medium` 无效。默认预算约 **1,572,516 像素（1254²）**。

提示词要精确宽高时，实际输出：

| 提示词要求 | 实际 PNG |
| --- | --- |
| 64×64 / 256×256 / 1024×1024 | 1254×1254 |
| 2048×2048 / 4096×4096 | 1254×1254 |
| 16×16 | 1254×1254 |

提示词要比例时，像素按同一预算取整：

| 提示词比例 | 实际 PNG |
| --- | --- |
| 1:1 | 1254×1254 |
| 16:9 / 1920×1080 | 1672×941 |
| 9:16 | 941×1672 |
| 4:3 | 1448×1086 |

工具字段 `tools[].size` 回显仍是 `auto`，但部分枚举会打中：

| 请求 `size` | 实际结果 |
| --- | --- |
| `1024x1024` / `512x512` / `2048x2048` | 仍是 1254×1254 |
| `1536x1024` | 1536×1024 |
| `1024x1536` | 1024×1536 |

结论：不能靠提示词拿到 512 / 1024 / 2K / 4K。对外接口只能把 OpenAI 常见尺寸映射到上述真实档位。

### 1.3 可用的 Responses 生图请求（改造前的底层协议）

```json
{
  "model": "gpt-5.6-sol",
  "stream": true,
  "input": [
    {
      "role": "user",
      "content": [
        { "type": "input_image", "image_url": "data:image/png;base64,...." },
        { "type": "input_text", "text": "Keep the composition and add a white circle." }
      ]
    }
  ],
  "tools": [{ "type": "image_generation" }],
  "tool_choice": { "type": "image_generation" }
}
```

无参考图时去掉 `input_image` 即可。

## 2. 对外接口（改造后）

Codex 渠道把 OpenAI Images API 转成上面的 Responses 调用，再把 `image_generation_call.result` 包装成 gpt-image 风格返回。客户端不必再拼 `tools`。

### 2.1 文生图

`POST /v1/images/generations`

```json
{
  "model": "gpt-image-1",
  "prompt": "A red square on a white background",
  "size": "1024x1024",
  "n": 1
}
```

也可用渠道上已有的 Codex 聊天模型，例如 `"model": "gpt-5.6-sol"`。

返回：

```json
{
  "created": 1788684087,
  "data": [
    {
      "b64_json": "iVBORw0KGgo...",
      "revised_prompt": "..."
    }
  ]
}
```

只返回 `b64_json`，不返回 `url`。`n` 仅支持 `1`。上游强制流式，网关会收齐后再以非流式 JSON 返回。

### 2.2 参考图改图

`POST /v1/images/edits`

JSON：

```json
{
  "model": "gpt-image-1",
  "prompt": "Add a white circle in the center",
  "image": "data:image/png;base64,...."
}
```

或标准 multipart：`prompt` + 文件字段 `image`。不支持 `mask`。

### 2.3 尺寸映射

| 客户端 `size` | 发给上游的工具 size | 实际大致输出 |
| --- | --- | --- |
| 空 / `auto` / `1024x1024` / `256x256` / `512x512` | 省略（auto） | 1254×1254 |
| `1536x1024` / `1792x1024` | `1536x1024` | 1536×1024 |
| `1024x1536` / `1024x1792` | `1024x1536` | 1024×1536 |
| 其他 | 省略（auto） | 按提示词比例落在约 157 万像素 |

### 2.4 渠道配置

1. 渠道类型选 **ChatGPT Subscription (Codex)**，密钥为 Codex OAuth JSON。
2. 模型列表加上 `gpt-image-1`（以及可选的 `gpt-image-2`），或直接用 `gpt-5.6-sol` 打 Images 接口。
3. 若对外模型名是 `gpt-image-1`，建议在渠道模型映射里写成 `gpt-image-1 → gpt-5.6-sol`（或 `gpt-5.6-terra`）。未映射时，适配器会把 `gpt-image-*` 默认改写成 `gpt-5.6-sol` 再发给 Codex。
4. 令牌的模型范围要包含客户端实际传入的模型名。

计费走 Images 路径的 usage（含 `image_gen` 的输入/输出 token），不再额外按 Responses 的 `image_generation` 工具价收一次，避免双计费。
