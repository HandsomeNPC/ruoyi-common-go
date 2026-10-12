# excelhelper

结构体 tag 驱动的通用 xlsx 导出/导入，底层走 [xuri/excelize](https://github.com/qax-os/excelize)。

## 标记字段

在 VO 字段上打 tag 即可，无需写映射配置：

| tag         | 作用                             | 示例                        |
|-------------|----------------------------------|-----------------------------|
| `excel`     | 表头文案；没打 tag 的字段不导出  | `excel:"状态"`              |
| `excelDict` | 值映射，导出取标签、导入反取原值 | `excelDict:"0=正常,1=停用"` |

```go
type SysClientVo struct {
ID     int64  `excel:"id"`
Status string `excel:"状态" excelDict:"0=正常,1=停用"`
Inner  []string // 无 tag，不导出
}
```

列序即字段声明序（不按字母序）；导入时按表头文本定位列，模板被人工调列序也能读回。雪花 ID 超过 15 位自动按字符串存取，低位不丢。

## 导出

```go
import "github.com/HandsomeNPC/ruoyi-common-go/excelhelper"

// Export 把 rows 写成 xlsx 附件直接写进 gin 响应，sheetName 同时作下载文件名主体。
if err := excelhelper.Export(c, rows, "客户端管理"); err != nil {
// 交给你项目的异常中间件渲染。本仓 RuoYi-Go-Vue-Plus 用 _ = c.Error(err) + Recover。
_ = c.Error(err)
return
}
```

`Export` 把整个工作簿建进内存再落第一个字节——失败须在写响应之前完成，否则一旦抢先写字节，后续错误会被静默吞成 200 + 半截文件。

只想要字节流（不写 HTTP）用 `Write`：

```go
buf, err := excelhelper.Write(rows, excelhelper.Options{SheetName: "客户端管理"})
```

## 导入

```go
rows, err := excelhelper.Read[SysUserImportVo](r) // r: io.Reader
```

带 `excelDict` 的列做反向映射：单元格写"正常"换回"0"，直接写"0"也原样保留，两种填法都吃。

## 行数上限

`excelhelper.MaxRows = 100000`（不含表头）。超限时 `Write`/`Export` 返回 `*excelhelper.TooManyRowsError{Count, Limit}`，
**不静默截断**
——审计表少几行比导出失败更危险。

建议查询时直接按 `excelhelper.MaxRows+1` 取数提前判定，免得查了一大堆再被导出层拦掉：

```go
rows, err := clientservice.QueryList(ctx, q, excelhelper.MaxRows+1)
```

## 错误如何回前端

`excel` 包不掺合响应层语义，`TooManyRowsError` 是普通 error。若想把超限错误以友好文案回前端，由调用方包成业务异常。RuoYi-Go-Vue-Plus
的做法（供参考）：

```go
var over *excelhelper.TooManyRowsError
if errors.As(err, &over) {
_ = c.Error(errs.New(response.CodeFail,
fmt.Sprintf("导出数据量过大，最多 %d 行，请缩小查询范围后重试", over.Limit),
err.Error()))
return
}
_ = c.Error(err)
```

其余失败（工作簿构建异常等）按系统异常兜底即可，不把原始技术信息回前端。

## 下载文件名响应头

附件名同时写进 `Content-Disposition` 的 `filename` 与 `filename*`，并额外设一个 `HeaderDownloadFilename` 头。前端读这个头拿文件名比解析
Content-Disposition 省事，但跨域时须在 CORS `exposedHeaders` 里放行它。

## 限制

工作簿全程驻留内存，单次上限 `MaxRows`。真要导出更大的量需改用 excelize 的 `StreamWriter`
，但它要求列宽在写行前定好，与本包按内容自适应列宽的做法冲突，故未采用。
