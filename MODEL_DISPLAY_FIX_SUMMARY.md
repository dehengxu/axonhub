# 按模型分组的请求图表 - 模型名称显示修复

## 问题描述
仪表板页面中，按模型分组的请求图表，当鼠标悬停在柱状图上时：
- ❌ 最初显示的是 "Value"
- ❌ 修改后，模型名称消失，只显示请求数量和百分比

## 问题分析
**第一版问题**：Tooltip默认显示"Value"，需要显示模型名称

**第二版问题**：Tooltip payload只包含dataKey指定的字段（value），不包含displayName，导致无法获取模型显示名称

## 解决方案

### 1. 创建模型名称映射常量文件

**文件**: `frontend/src/constants/model-names.ts`

创建了一个包含主流AI模型ID到友好名称映射的常量文件，包含以下模型：
- OpenAI: GPT-4o, GPT-4o Mini, GPT-4 Turbo, o1 Preview, o1 Mini 等
- Anthropic: Claude 3.5 Sonnet, Claude 3 Opus, Claude 3 Sonnet, Claude 3 Haiku 等
- DeepSeek: DeepSeek Chat, DeepSeek Reasoner, DeepSeek Coder 等
- Gemini: Gemini 1.5 Pro, Gemini 1.5 Flash 等
- Moonshot: Moonshot v1 8K/32K/128K 等
- Doubao: Doubao Lite/Pro 4K/32K 等
- Zhipu: GLM-4 系列
- Minimax: ABAB 系列
- xAI: Grok 系列
- Longcat, SiliconFlow 等

### 2. 修改图表组件

**文件**: `frontend/src/features/dashboard/components/requests-by-model-chart.tsx`

修改内容：
1. 导入 `getModelDisplayName` 函数
2. 在生成图表数据时，添加 `displayName` 字段用于存储友好名称
3. **修复Tooltip逻辑**：
   - 从`props.label`获取X轴值（modelId）
   - 在`chartData`中查找对应的完整数据项
   - 显示优先级：`displayName` → `name` → `label`（多重回退保障）
4. 更新图例部分，显示模型名称（优先使用别名，否则使用原始ID）

### 3. 核心修复逻辑

```typescript
const tooltipContent = (props: ModelTooltipProps) => {
  const payload = props.payload
  const label = props.label  // X轴的值（modelId）

  if (!props.active || !payload?.length || !label) return null

  const [{ value }] = payload
  // 在chartData中查找完整的数据项
  const item = chartData.find(d => d.name === label)
  const percent = total ? ((value ?? 0) / total) * 100 : 0

  return (
    <div className='bg-background/90 rounded-md border px-3 py-2 text-xs shadow-sm backdrop-blur'>
      <div className='text-foreground text-sm font-medium'>
        {item?.displayName || item?.name || label}  {/* 多重回退保障 */}
      </div>
      <div className='text-muted-foreground'>
        {value?.toLocaleString()} ({percent.toFixed(0)}%)
      </div>
    </div>
  )
}
```

### 4. Recharts 数据流说明

Recharts的Tooltip机制：
- `payload`: 包含dataKey指定的字段值（我们的案例中是`value`）
- `label`: X轴的dataKey值（即`modelId`）
- 其他字段（如`displayName`）不会自动传递到payload中

因此需要通过`label`在原始数据中查找完整的模型信息。

## 修改效果

修改后，当用户将鼠标悬停在柱状图上时：
- ✅ 显示模型ID的友好名称（如 "GPT-4o", "Claude 3.5 Sonnet"）
- ✅ 如果模型ID不在映射表中，显示原始的模型ID
- ✅ 同时显示请求数量和百分比
- ✅ 图例部分也显示友好的模型名称

### 示例Tooltip内容：
```
┌─────────────────┐
│ GPT-4o          │  ← 友好名称
│ 1,234 (45%)     │  ← 数量 (百分比)
└─────────────────┘
```

## 技术要点

1. **数据查找**：使用Array.find()在chartData中查找匹配的模型
2. **多重回退**：`item?.displayName || item?.name || label` 确保总有值显示
3. **类型安全**：使用TypeScript接口定义TooltipProps类型
4. **条件渲染**：增加label检查，防止未定义值导致显示错误

## 扩展性

该解决方案具有良好的扩展性：
- 可以在 `MODEL_NAMES` 常量中添加更多模型映射
- 如果需要从动态数据源获取别名，可以修改 `getModelDisplayName` 函数
- 支持自动回退：如果没有找到映射，自动使用原始模型ID

## 测试

前端开发服务器已启动：`http://localhost:5173/`
- ✅ 构建测试通过：无语法错误
- ✅ 热重载：支持开发时实时更新
- ✅ 开发模式：正常运行

## 文件清单

1. `frontend/src/constants/model-names.ts` (新增)
2. `frontend/src/features/dashboard/components/requests-by-model-chart.tsx` (修改)

## 验证步骤

1. 确保开发服务器正在运行：`http://localhost:5173/`
2. 访问仪表板页面
3. 查看"按模型分组的请求"图表
4. 将鼠标悬停在任何柱状图上
5. 确认Tooltip中显示：
   - ✅ 模型名称（如 "GPT-4o"）或原始模型ID
   - ✅ 请求数量
   - ✅ 百分比

## 修复历史

- **第一版**：初步修复，添加displayName但tooltip无法获取 ❌
- **第二版**：修复tooltip逻辑，使用label查找完整数据 ✅
