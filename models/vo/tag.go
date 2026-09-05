package vo

// TagItem 是标签云中的一个标签及其已发布文章数。
type TagItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
