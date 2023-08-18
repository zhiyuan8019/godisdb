package godis

type GodisType int

const (
	GODIS_STRING GodisType = iota
	GODIS_LIST
	GODIS_HASH
	GODIS_SET
	GODIS_ZSET
)

type GodisVal interface{}

type GodisObj struct {
	Type_ GodisType
	Ptr_  GodisVal
}

func CreateObj(t GodisType, ptr GodisVal) (o *GodisObj) {
	o = new(GodisObj)
	o.Type_ = t
	o.Ptr_ = ptr
	return
}
