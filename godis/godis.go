package godis

type GodisDb struct {
	Dict    map[string]*GodisObj
	Expires map[string]uint64
}

type GodisClient struct {
}

func initServer()
