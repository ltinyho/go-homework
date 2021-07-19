遇到一个`sql.ErrNoRows`时是否应该 `Wrap` 这个 `error`,抛给上层?

**我认为不应该抛给上层.**

 - 第一: `sql.ErrNoRows` 其实 `db` 执行 `sql` 是正常的,并不是 `db` 执行出错,只是没有符合查询条件的row.
   跟 `io.EOF` 类似,是一个特定的错误,不表示程序错误.
 - 第二: 如果返回给上层,上层通过 `sql.ErrNoRows` 来判断错误. 如果后续更换 `db`,上层代码需要依赖这个 `sentinel` 错误.
   可能其他的 `db` 根本没有这个错误, 还得强行转成 `sql.ErrNoRows`,或者修改上层代码.

具体代码参见: [errors.go](./errors.go)   
