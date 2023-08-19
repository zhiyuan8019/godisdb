package godis

type GodisType int

const (
	GODIS_STRING GodisType = 1
	GODIS_LIST   GodisType = 2
	GODIS_HASH   GodisType = 3
	GODIS_SET    GodisType = 4
	GODIS_ZSET   GodisType = 5
)

type GodisVal interface{}

type GodisObj struct {
	obj_type GodisType
	val      GodisVal
}

type GodisHash map[string]*GodisObj

func CreateObj(t GodisType, val GodisVal) (o *GodisObj) {
	o = new(GodisObj)
	o.obj_type = t
	if t == GODIS_STRING {
		o.val = val
	} else if t == GODIS_HASH {
		o.val = make(GodisHash)
	}

	return
}
