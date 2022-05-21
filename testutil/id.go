package testutil

type Gen interface {
	New() string
}

// IDGen wraps another id.Gen but remembers the last
// ID that was generated in order to make assertions.
type IDGen struct {
	gen Gen
	id  string
}

// New implements the id.Gen interface.
func (s *IDGen) New() string {
	id := s.id
	s.id = s.gen.New()
	return id
}

// Last returns the last ID that was generated.
func (s *IDGen) Last() string {
	return s.id
}

func NewIDGen(gen Gen) *IDGen {
	return &IDGen{
		gen: gen,
		id:  gen.New(),
	}
}
