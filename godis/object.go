package godis

type GodisType int

const (
	GODIS_NONE   GodisType = 0
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

type GodisSet map[string]*GodisObj

type GodisZset struct {
	dict      map[string]float64
	zskiplist *GodisSkipList
}

func CreateObj(t GodisType, val GodisVal) (o *GodisObj) {
	o = new(GodisObj)
	o.obj_type = t
	// if t == GODIS_STRING {
	// 	o.val = val
	// } else if t == GODIS_HASH {
	// 	o.val = make(GodisHash)
	// }
	switch t {
	case GODIS_STRING:
		o.val = val
	case GODIS_HASH:
		o.val = make(GodisHash)
	case GODIS_LIST:
		l := ListCreate()
		o.val = &l
	case GODIS_SET:
		o.val = make(GodisSet)
	case GODIS_ZSET:
		o.val = &GodisZset{
			dict:      make(map[string]float64),
			zskiplist: CreateSkipList(),
		}
	}
	return
}

func CompareStrObj(a *GodisObj, b *GodisObj) bool {
	if a.obj_type != b.obj_type || a.obj_type != GODIS_STRING {
		return false
	}
	if strVal1, ok := a.val.(string); ok {
		if strVal2, ok := b.val.(string); ok {
			return strVal1 == strVal2
		}
	}
	return false
}
