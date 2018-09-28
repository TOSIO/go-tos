package synchronise

type TimeSliceResp struct {
	id        string
	timeSlice int64
}

func (ts *TimeSliceResp) NodeId() string {
	return ts.id
}

func (ts *TimeSliceResp) Items() int {
	return 1
}
