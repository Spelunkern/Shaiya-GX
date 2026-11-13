package scene

// Small integer resource handles keep residency rebuilds from repeatedly
// hashing the same file path for thousands of placements.
type fieldResource struct {
	file    string
	model   *Model
	vani    *vaniAsset
	batches [2]*fieldModelBatch
}

func (s *fieldStream) indexResources() {
	s.resourceIndex = map[string]int{}
	s.resources = nil
	for i := range s.objects {
		o := &s.objects[i]
		id, ok := s.resourceIndex[o.file]
		if !ok {
			id = len(s.resources)
			s.resourceIndex[o.file] = id
			s.resources = append(s.resources, fieldResource{file: o.file})
		}
		o.resourceID = id
	}
	s.resourceObjectCount = len(s.objects)
	s.objectMarks = make([]bool, len(s.objects))
	s.resourceMarks = make([]bool, len(s.resources))
}
func (s *fieldStream) resourceAlive(file string) bool {
	id, ok := s.resourceIndex[file]
	return ok && s.resourceMarks[id]
}
