package service

var gService map[string]*Service

func init() {
	gService = make(map[string]*Service)
}

func Register(name string, svc *Service) {
	gService[name] = svc
}

func Lookup(name string) (*Service, bool) {
	s, ok := gService[name]
	return s, ok
}

func MustLookup(name string) *Service {
	return gService[name]
}
